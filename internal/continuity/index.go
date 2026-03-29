package continuity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/manuscript"
	"github.com/poiesic/nib/internal/scene"
	"github.com/poiesic/nib/internal/storydb"
	"github.com/schollz/progressbar/v3"
)

// ExtractFunc is the function signature for agent extraction. Override in tests.
type ExtractFunc func(prompt string, schema json.RawMessage, dir string) (json.RawMessage, error)

// IndexOptions configures the continuity index operation.
// Specify Range (e.g. "3.2", "1-3", "1.1-2.3", "1,3,5"). If empty, falls back to focus.
type IndexOptions struct {
	Range     string      // range expression (e.g. "3.2", "1-3", "1.1-2.3", "1,3,5")
	Verbose   bool        // print prompt, command, and raw response
	Force     bool        // index even if scene checksum hasn't changed
	ExtractFn ExtractFunc // nil = agent.ContinuityIndex
	Stdout    io.Writer   // nil = os.Stdout
	Stdin     io.Reader   // nil = os.Stdin
}

// ExtractionResult holds the structured data extracted from a scene by Claude.
type ExtractionResult struct {
	Scene      storydb.Scene            `json:"scene"`
	Facts      []storydb.Fact           `json:"facts"`
	Characters []storydb.SceneCharacter `json:"characters"`
	Locations  []storydb.Location       `json:"locations"`
}

// jsonSchema is the JSON schema passed to claude --json-schema for structured extraction.
var jsonSchema = `{
  "type": "object",
  "required": ["scene", "facts", "characters", "locations"],
  "properties": {
    "scene": {
      "type": "object",
      "required": ["pov", "scene_type", "location", "date", "time", "summary"],
      "properties": {
        "pov":        {"type": "string", "description": "POV character slug (lowercase-hyphenated)"},
        "scene_type": {"type": "string", "enum": ["regular", "interlude", "document"]},
        "location":   {"type": "string", "description": "Primary location slug"},
        "date":       {"type": "string", "description": "ISO date in narrative (YYYY-MM-DD) or empty"},
        "time":       {"type": "string", "description": "Time of day or empty"},
        "summary":    {"type": "string", "description": "One-line scene summary"}
      }
    },
    "facts": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["category", "summary", "detail", "source_text", "date", "time"],
        "properties": {
          "category":    {"type": "string", "enum": ["event", "description", "relationship", "state"]},
          "summary":     {"type": "string", "description": "One-line summary"},
          "detail":      {"type": "string", "description": "Full description"},
          "source_text": {"type": "string", "description": "Direct quote from scene text"},
          "date":        {"type": "string", "description": "When in narrative (ISO date or empty)"},
          "time":        {"type": "string", "description": "Time if applicable or empty"}
        }
      }
    },
    "characters": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["character", "role"],
        "properties": {
          "character": {"type": "string", "description": "Character slug (lowercase-hyphenated)"},
          "role":      {"type": "string", "enum": ["pov", "present", "mentioned"]}
        }
      }
    },
    "locations": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "name", "type", "description"],
        "properties": {
          "id":          {"type": "string", "description": "Location slug (lowercase-hyphenated)"},
          "name":        {"type": "string", "description": "Display name"},
          "type":        {"type": "string", "enum": ["workplace", "home", "public", "outdoor", "vehicle", "other"]},
          "description": {"type": "string", "description": "Physical description"}
        }
      }
    }
  }
}`

// Index extracts structured continuity data from one or more scenes using Claude.
func Index(opts IndexOptions) error {
	extractFn := opts.ExtractFn
	if extractFn == nil {
		extractFn = agent.ContinuityIndex
	}
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	// Resolve which scenes to index
	var scenes []manuscript.ResolvedScene
	if opts.Range == "" {
		// Fall back to focus
		focus, err := scene.GetFocus(projectRoot, book)
		if err != nil {
			return err
		}
		if focus == nil {
			return fmt.Errorf("no scene specified and no focus set; usage: nib continuity index <range>")
		}
		if focus.Slug == "" {
			return fmt.Errorf("focus is set to chapter %d but no specific scene; specify a range or use nib scene focus <chapter.scene>", focus.Chapter)
		}
		// Determine interlude flag for the focused scene
		isInterlude := false
		if focus.Chapter >= 1 && focus.Chapter <= len(book.Chapters) {
			isInterlude = book.Chapters[focus.Chapter-1].Interlude
		}
		scenes = []manuscript.ResolvedScene{{Slug: focus.Slug, Interlude: isInterlude}}
	} else {
		spec, err := manuscript.ParseRange(opts.Range)
		if err != nil {
			return err
		}
		scenes, err = manuscript.ResolveSlugs(book, spec)
		if err != nil {
			return err
		}
	}

	// Open storydb once for all scenes
	db, err := storydb.Open(projectRoot)
	if err != nil {
		return fmt.Errorf("opening storydb: %w", err)
	}
	defer db.Close()

	// Single-scene fast path: no pool overhead
	if len(scenes) == 1 {
		return indexSceneSerial(db, scenes[0].Slug, scenes[0].Interlude, projectRoot, book, extractFn, stdin, stdout, opts)
	}

	// Multi-scene: sequential extraction so each scene gets context from earlier scenes
	characterSlugs := readCharacterSlugs(projectRoot)

	// Phase 1: extract all scenes with progress bar
	// Extractions run in a goroutine so the main thread can tick the bar's
	// elapsed/ETA counters while cmd.Output() blocks.
	bar := progressbar.NewOptions(len(scenes),
		progressbar.OptionSetWriter(stdout),
		progressbar.OptionSetDescription("Extracting"),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
	)

	type sceneResult struct {
		slug    string
		result  *ExtractionResult
		skipped bool
		err     error
	}

	results := make([]sceneResult, 0, len(scenes))
	var earlierExtractions []*ExtractionResult

	for _, s := range scenes {
		statusFn := func(phase string) {
			bar.Describe(fmt.Sprintf("%s %s", phase, s.Slug))
		}

		type extractResult struct {
			result  *ExtractionResult
			skipped bool
			err     error
		}
		done := make(chan extractResult, 1)
		go func() {
			result, wasSkipped, err := extractScene(
				db, s.Slug, s.Interlude, projectRoot, book, extractFn, opts,
				characterSlugs, earlierExtractions, statusFn,
			)
			done <- extractResult{result, wasSkipped, err}
		}()

		// Tick the bar every second so elapsed/ETA counters update during the LLM call
		ticker := time.NewTicker(1 * time.Second)
		var r extractResult
	waitLoop:
		for {
			select {
			case r = <-done:
				break waitLoop
			case <-ticker.C:
				bar.Add(0)
			}
		}
		ticker.Stop()
		bar.Add(1)

		results = append(results, sceneResult{
			slug:    s.Slug,
			result:  r.result,
			skipped: r.skipped,
			err:     r.err,
		})

		if r.err == nil && !r.skipped {
			earlierExtractions = append(earlierExtractions, r.result)
		}
	}
	bar.Finish()
	fmt.Fprintln(stdout)

	// Phase 2: sequential review + write
	totalFacts := 0
	totalChars := 0
	totalLocs := 0
	indexed := 0
	skipped := 0

	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(stdout, "%s: error: %v\n", r.slug, r.err)
			continue
		}
		if r.skipped {
			fmt.Fprintf(stdout, "%s: unchanged, skipping (use --force to re-index)\n", r.slug)
			skipped++
			continue
		}

		facts, chars, locs, err := reviewAndWrite(db, r.result, stdin, stdout)
		if err != nil {
			fmt.Fprintf(stdout, "%s: review error: %v\n", r.slug, err)
			continue
		}
		indexed++
		totalFacts += facts
		totalChars += chars
		totalLocs += locs
	}

	fmt.Fprintf(stdout, "\nTotal: %d scenes indexed, %d skipped, %d facts, %d characters, %d locations\n",
		indexed, skipped, totalFacts, totalChars, totalLocs)

	return nil
}

// indexSceneSerial processes a single scene end-to-end (extract + review + write).
// Used for the single-scene fast path where no parallelism is needed.
func indexSceneSerial(db *storydb.DB, slug string, isInterlude bool, projectRoot string, book *binder.Book, extractFn ExtractFunc, stdin io.Reader, stdout io.Writer, opts IndexOptions) error {
	characterSlugs := readCharacterSlugs(projectRoot)

	result, skipped, err := extractScene(db, slug, isInterlude, projectRoot, book, extractFn, opts, characterSlugs, nil, nil)
	if err != nil {
		return err
	}
	if skipped {
		fmt.Fprintf(stdout, "%s: unchanged, skipping (use --force to re-index)\n", slug)
		return nil
	}

	// Verbose output only in serial mode (would interleave in parallel)
	if opts.Verbose {
		fmt.Fprintf(stdout, "=== EXTRACTION COMPLETE for %s ===\n\n", slug)
	}

	_, _, _, err = reviewAndWrite(db, result, stdin, stdout)
	return err
}

// Phase indicators for progress display.
const (
	phaseBuild   = "B" // building prompt
	phaseCall    = "↑" // sending to LLM
	phaseProcess = "↓" // processing response
)

// extractScene performs the work: file read, checksum check,
// Claude CLI call, response parse, and field population.
// It does NOT do interactive review or DB writes.
// statusFn, if non-nil, is called with a phase indicator string at each stage.
func extractScene(db *storydb.DB, slug string, isInterlude bool, projectRoot string, book *binder.Book, extractFn ExtractFunc, opts IndexOptions, characterSlugs []string, earlierExtractions []*ExtractionResult, statusFn func(string)) (*ExtractionResult, bool, error) {
	setStatus := func(phase string) {
		if statusFn != nil {
			statusFn(phase)
		}
	}

	setStatus(phaseBuild)

	// Find scene in book and resolve file path
	scenePath, _, pathErr := findScenePath(projectRoot, book, slug)
	if pathErr != nil {
		return nil, false, pathErr
	}

	// Read scene file for checksum computation and empty check
	sceneBytes, err := os.ReadFile(scenePath)
	if err != nil {
		return nil, false, fmt.Errorf("reading scene file: %w", err)
	}
	if len(strings.TrimSpace(string(sceneBytes))) == 0 {
		return nil, false, fmt.Errorf("scene %q is empty; write some prose before indexing", slug)
	}

	// Compute SHA256 checksum
	hash := sha256.Sum256(sceneBytes)
	checksum := hex.EncodeToString(hash[:])

	// Check if scene has changed since last indexing
	if !opts.Force {
		existing, err := db.SceneChecksum(slug)
		if err != nil {
			return nil, false, fmt.Errorf("checking scene checksum: %w", err)
		}
		if existing == checksum {
			return nil, true, nil
		}
	}

	// Build prior chapter recap for dedup context (up to 2 chapters back)
	priorRecapJSON := buildPriorRecap(slug, book, projectRoot)

	// Build the prompt with scene file path for Claude to read
	prompt := buildPrompt(slug, scenePath, isInterlude, priorRecapJSON, characterSlugs, earlierExtractions)

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "[verbose] prompt length: %d chars\n", len(prompt))
		fmt.Fprintf(os.Stderr, "[verbose] prompt:\n%s\n", prompt)
	}

	setStatus(phaseCall)

	// Extract structured data via agent
	data, err := extractFn(prompt, json.RawMessage(jsonSchema), projectRoot)
	if err != nil {
		return nil, false, fmt.Errorf("extraction failed: %w", err)
	}

	setStatus(phaseProcess)

	// Parse response
	result := &ExtractionResult{}
	if err := json.Unmarshal(data, result); err != nil {
		return nil, false, fmt.Errorf("parsing extraction response: %w", err)
	}

	// Sanitize free-text fields: csvq doesn't properly quote newlines in CSV output
	sanitizeResult(result)

	// Fill in the scene slug, checksum, and timestamp
	now := time.Now().UTC().Format(time.RFC3339)
	result.Scene.Scene = slug
	result.Scene.Checksum = checksum
	result.Scene.IndexedAt = now

	// Fill in scene_type from book structure if interlude
	if isInterlude && result.Scene.SceneType == "regular" {
		result.Scene.SceneType = "interlude"
	}

	// Assign ULIDs and timestamps to facts
	for i := range result.Facts {
		result.Facts[i].Scene = slug
		result.Facts[i].ID = storydb.NewID()
		result.Facts[i].IndexedAt = now
	}

	// Fill timestamps for characters
	for i := range result.Characters {
		result.Characters[i].Scene = slug
		result.Characters[i].IndexedAt = now
	}

	// Assign ULIDs and timestamps to locations
	for i := range result.Locations {
		if result.Locations[i].ID == "" {
			result.Locations[i].ID = storydb.NewID()
		}
		result.Locations[i].FirstScene = slug
		result.Locations[i].IndexedAt = now
	}

	return result, false, nil
}

// reviewAndWrite runs interactive review on an extraction result, then writes
// accepted records to the DB. Returns counts of accepted facts/characters/locations.
func reviewAndWrite(db *storydb.DB, result *ExtractionResult, stdin io.Reader, stdout io.Writer) (facts, chars, locs int, err error) {
	accepted, err := reviewProposals(result, stdin, stdout)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("review: %w", err)
	}

	if err := writeAccepted(db, accepted); err != nil {
		return 0, 0, 0, err
	}

	slug := result.Scene.Scene
	fmt.Fprintf(stdout, "\nIndexed %s: 1 scene, %d facts, %d characters, %d locations\n",
		slug, len(accepted.Facts), len(accepted.Characters), len(accepted.Locations))

	return len(accepted.Facts), len(accepted.Characters), len(accepted.Locations), nil
}

func writeAccepted(db *storydb.DB, result *ExtractionResult) error {
	slug := result.Scene.Scene

	// Remove existing records for this scene
	for _, table := range []string{"facts", "scene_characters"} {
		if err := db.DeleteByScene(table, slug); err != nil {
			return fmt.Errorf("clearing old records from %s: %w", table, err)
		}
	}

	// Upsert scene (handles delete + insert atomically)
	if err := db.UpsertScene(result.Scene); err != nil {
		return fmt.Errorf("writing scene: %w", err)
	}
	if err := db.InsertFacts(result.Facts); err != nil {
		return fmt.Errorf("writing facts: %w", err)
	}
	if err := db.InsertSceneCharacters(result.Characters); err != nil {
		return fmt.Errorf("writing characters: %w", err)
	}
	if err := db.InsertLocations(result.Locations); err != nil {
		return fmt.Errorf("writing locations: %w", err)
	}
	return nil
}

// findScenePath locates a scene slug in the book and returns its absolute file path.
func findScenePath(projectRoot string, book *binder.Book, slug string) (string, bool, error) {
	for _, ch := range book.Chapters {
		if slices.Contains(ch.Scenes, slug) {
			sceneDir := book.BaseDir
			if ch.Subdir != "" {
				sceneDir = filepath.Join(book.BaseDir, ch.Subdir)
			}
			if !filepath.IsAbs(sceneDir) {
				sceneDir = filepath.Join(projectRoot, sceneDir)
			}
			return filepath.Join(sceneDir, slug+".md"), ch.Interlude, nil
		}
	}
	return "", false, fmt.Errorf("scene %q not found in book.yaml", slug)
}

// readCharacterSlugs globs characters/*.yaml and extracts slugs from filenames.
func readCharacterSlugs(projectRoot string) []string {
	pattern := filepath.Join(projectRoot, "characters", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var slugs []string
	for _, m := range matches {
		base := filepath.Base(m)
		slug := strings.TrimSuffix(base, ".yaml")
		slugs = append(slugs, slug)
	}
	return slugs
}

// buildPriorRecap generates a JSON recap of up to 2 chapters before the chapter
// containing the given slug. Returns empty string if no prior chapters exist.
func buildPriorRecap(slug string, book *binder.Book, projectRoot string) string {
	// Find which chapter this slug belongs to
	chapterNum := 0
	for i, ch := range book.Chapters {
		for _, s := range ch.Scenes {
			if s == slug {
				chapterNum = i + 1
				break
			}
		}
		if chapterNum > 0 {
			break
		}
	}
	if chapterNum <= 1 {
		return ""
	}

	// Build range for up to 2 prior chapters
	start := chapterNum - 2
	if start < 1 {
		start = 1
	}
	end := chapterNum - 1
	rangeExpr := fmt.Sprintf("%d-%d", start, end)
	if start == end {
		rangeExpr = fmt.Sprintf("%d", start)
	}

	var buf bytes.Buffer
	err := Recap(RecapOptions{
		Range:  rangeExpr,
		Stdout: &buf,
		Stderr: io.Discard,
	})
	if err != nil {
		return ""
	}
	return buf.String()
}

func buildPrompt(slug, scenePath string, isInterlude bool, priorRecapJSON string, characterSlugs []string, earlierExtractions []*ExtractionResult) string {
	var b strings.Builder

	b.WriteString("You are a continuity analyst for a novel manuscript. ")
	b.WriteString("Extract structured data from the scene for a continuity database.\n\n")

	b.WriteString("## Rules\n")
	b.WriteString("- Use lowercase-hyphenated slugs for all identifiers (e.g. 'lance-thurgood', 'wellness-center')\n")
	b.WriteString("- For facts, quote the relevant passage in source_text\n")
	b.WriteString("- Categorize facts as: event (something that happens), description (physical/sensory detail), relationship (between characters), state (character emotional/physical state)\n")
	b.WriteString("- Mark character roles as: pov (point-of-view character), present (physically in scene), mentioned (talked about but not present)\n")
	b.WriteString("- Use ISO dates (YYYY-MM-DD) when the text indicates a date; leave empty if unknown\n")
	b.WriteString("- Do not duplicate facts from the prior chapter context below\n\n")

	if isInterlude {
		b.WriteString("Note: This scene is marked as an interlude in the manuscript structure.\n\n")
	}

	if len(characterSlugs) > 0 {
		b.WriteString("## Known Characters\n")
		b.WriteString("Use these canonical slugs when referring to these characters:\n")
		for _, s := range characterSlugs {
			fmt.Fprintf(&b, "- %s\n", s)
		}
		b.WriteString("\n")
	}

	if priorRecapJSON != "" {
		b.WriteString("## Prior Chapter Context\n")
		b.WriteString("The following JSON contains indexed data from recent prior chapters. Use it for continuity and to avoid duplicating facts.\n\n")
		b.WriteString("```json\n")
		b.WriteString(priorRecapJSON)
		b.WriteString("```\n\n")
	}

	if len(earlierExtractions) > 0 {
		b.WriteString("## Earlier Scenes in This Chapter\n")
		b.WriteString("The following scenes were already extracted in this indexing batch. Do not duplicate their facts.\n\n")
		b.WriteString("```json\n")
		earlierJSON, _ := json.MarshalIndent(earlierExtractions, "", "  ")
		b.Write(earlierJSON)
		b.WriteString("\n```\n\n")
	}

	fmt.Fprintf(&b, "## Scene: %s\n\n", slug)
	fmt.Fprintf(&b, "Read the scene file at: %s\n", scenePath)

	return b.String()
}

// sanitizeResult collapses newlines to spaces in all free-text fields.
// csvq doesn't properly quote fields containing newlines when writing CSV,
// which corrupts the storydb files.
func sanitizeResult(r *ExtractionResult) {
	r.Scene.Summary = collapseNewlines(r.Scene.Summary)
	for i := range r.Facts {
		r.Facts[i].Summary = collapseNewlines(r.Facts[i].Summary)
		r.Facts[i].Detail = collapseNewlines(r.Facts[i].Detail)
		r.Facts[i].SourceText = collapseNewlines(r.Facts[i].SourceText)
	}
	for i := range r.Locations {
		r.Locations[i].Description = collapseNewlines(r.Locations[i].Description)
	}
}

// collapseNewlines replaces newline sequences with a single space.
func collapseNewlines(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
