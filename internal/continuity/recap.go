package continuity

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/manuscript"
	"github.com/poiesic/nib/internal/storydb"
)

// RecapOptions configures the recap command.
type RecapOptions struct {
	Range      string    // chapter range expression (e.g. "1", "3-5", "1,3,5")
	Characters []string  // if set, only include scenes where these characters appear
	Detailed   bool      // if true, include facts, location, date, time, and mentioned characters
	Pretty     bool      // if true, pretty-print JSON output
	Stdout     io.Writer // nil = os.Stdout
	Stderr     io.Writer // nil = os.Stderr
}

// RecapOutput is the top-level JSON output for a recap.
type RecapOutput struct {
	Chapters []RecapChapter `json:"chapters"`
}

// RecapChapter holds recap data for a single chapter.
type RecapChapter struct {
	Chapter   int          `json:"chapter"`
	Name      string       `json:"name,omitempty"`
	Interlude bool         `json:"interlude,omitempty"`
	Scenes    []RecapScene `json:"scenes"`
}

// RecapScene holds recap data for a single scene within a chapter.
type RecapScene struct {
	Slug       string           `json:"slug"`
	Position   int              `json:"position"`
	POV        string           `json:"pov,omitempty"`
	Location   string           `json:"location,omitempty"`
	Date       string           `json:"date,omitempty"`
	Time       string           `json:"time,omitempty"`
	Summary    string           `json:"summary,omitempty"`
	Indexed    bool             `json:"indexed"`
	Facts      []RecapFact      `json:"facts,omitempty"`
	Characters []RecapCharacter `json:"characters,omitempty"`
}

// RecapFact is a slimmed-down fact for recap output.
type RecapFact struct {
	Category string `json:"category"`
	Summary  string `json:"summary"`
	Detail   string `json:"detail,omitempty"`
}

// RecapCharacter is a slimmed-down character appearance for recap output.
type RecapCharacter struct {
	Character string `json:"character"`
	Role      string `json:"role"`
}

// Recap generates a JSON recap of the specified chapter range.
func Recap(opts RecapOptions) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	spec, err := manuscript.ParseRange(opts.Range)
	if err != nil {
		return err
	}

	// Resolve which chapters are in range
	chapters, err := resolveChapters(book, spec)
	if err != nil {
		return err
	}

	// Collect all slugs across all chapters for batch DB queries
	var allSlugs []string
	for _, ch := range chapters {
		allSlugs = append(allSlugs, ch.slugs...)
	}

	// Open storydb and query
	db, err := storydb.Open(projectRoot)
	if err != nil {
		return fmt.Errorf("opening storydb: %w", err)
	}
	defer db.Close()

	// Validate character slugs if filtering by character
	if len(opts.Characters) > 0 {
		allChars, err := db.QueryDistinctCharacters(nil)
		if err != nil {
			return fmt.Errorf("querying known characters: %w", err)
		}
		known := make(map[string]bool, len(allChars))
		for _, c := range allChars {
			known[c] = true
		}
		for _, c := range opts.Characters {
			if !known[c] {
				return fmt.Errorf("character slug %q not found. Make sure to use full names in slug format: \"john-doe\" not \"doe\", \"John Doe\", or \"john doe\"", c)
			}
		}
	}

	// If filtering by character, narrow allSlugs to only scenes involving those characters
	if len(opts.Characters) > 0 {
		charSlugs, err := db.QuerySceneSlugsForCharacters(opts.Characters)
		if err != nil {
			return fmt.Errorf("querying scenes for characters: %w", err)
		}
		charSlugSet := make(map[string]bool, len(charSlugs))
		for _, s := range charSlugs {
			charSlugSet[s] = true
		}
		filtered := allSlugs[:0]
		for _, s := range allSlugs {
			if charSlugSet[s] {
				filtered = append(filtered, s)
			}
		}
		allSlugs = filtered

		// Also filter chapter slugs so empty chapters are dropped later
		for i := range chapters {
			var kept []string
			for _, s := range chapters[i].slugs {
				if charSlugSet[s] {
					kept = append(kept, s)
				}
			}
			chapters[i].slugs = kept
		}
	}

	scenes, err := db.QueryScenesBySlugs(allSlugs)
	if err != nil {
		return fmt.Errorf("querying scenes: %w", err)
	}

	var facts []storydb.Fact
	if opts.Detailed {
		facts, err = db.QueryFactsBySlugs(allSlugs)
		if err != nil {
			return fmt.Errorf("querying facts: %w", err)
		}
	}

	chars, err := db.QueryCharactersBySlugs(allSlugs)
	if err != nil {
		return fmt.Errorf("querying characters: %w", err)
	}

	// Index DB results by scene slug for fast lookup
	sceneMap := make(map[string]storydb.Scene, len(scenes))
	for _, s := range scenes {
		sceneMap[s.Scene] = s
	}

	factMap := make(map[string][]storydb.Fact)
	for _, f := range facts {
		factMap[f.Scene] = append(factMap[f.Scene], f)
	}

	charMap := make(map[string][]storydb.SceneCharacter)
	for _, c := range chars {
		charMap[c.Scene] = append(charMap[c.Scene], c)
	}

	// Build output
	output := RecapOutput{
		Chapters: make([]RecapChapter, 0, len(chapters)),
	}

	unindexed := 0
	for _, ch := range chapters {
		if len(ch.slugs) == 0 {
			continue
		}
		rc := RecapChapter{
			Chapter:   ch.number,
			Name:      ch.name,
			Interlude: ch.interlude,
			Scenes:    make([]RecapScene, 0, len(ch.slugs)),
		}

		for i, slug := range ch.slugs {
			rs := RecapScene{
				Slug:     slug,
				Position: i + 1,
			}

			if s, ok := sceneMap[slug]; ok {
				rs.Indexed = true
				rs.POV = s.POV
				rs.Summary = s.Summary
				if opts.Detailed {
					rs.Location = s.Location
					rs.Date = s.Date
					rs.Time = s.Time
				}
			} else {
				unindexed++
			}

			if opts.Detailed {
				if sceneFacts, ok := factMap[slug]; ok {
					for _, f := range sceneFacts {
						rs.Facts = append(rs.Facts, RecapFact{
							Category: f.Category,
							Summary:  f.Summary,
							Detail:   f.Detail,
						})
					}
				}
			}

			if sceneChars, ok := charMap[slug]; ok {
				for _, c := range sceneChars {
					if !opts.Detailed && c.Role == "mentioned" {
						continue
					}
					rs.Characters = append(rs.Characters, RecapCharacter{
						Character: c.Character,
						Role:      c.Role,
					})
				}
			}

			rc.Scenes = append(rc.Scenes, rs)
		}

		output.Chapters = append(output.Chapters, rc)
	}

	if unindexed > 0 {
		fmt.Fprintf(stderr, "warning: %d scene(s) not yet indexed; run nib ct index to populate\n", unindexed)
	}

	enc := json.NewEncoder(stdout)
	if opts.Pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(output)
}

// chapterInfo holds resolved chapter data for building the recap.
type chapterInfo struct {
	number    int
	name      string
	interlude bool
	slugs     []string
}

// resolveChapters expands a RangeSpec into chapter-grouped data.
// Only chapter-level ranges are supported (not dotted scene refs).
func resolveChapters(book *binder.Book, spec manuscript.RangeSpec) ([]chapterInfo, error) {
	var chapterNums []int

	switch spec.Kind {
	case "list":
		for _, ref := range spec.Refs {
			if ref.Position != 0 {
				return nil, fmt.Errorf("recap operates on whole chapters, not individual scenes; use %d instead of %d.%d", ref.Chapter, ref.Chapter, ref.Position)
			}
			chapterNums = append(chapterNums, ref.Chapter)
		}
	case "range":
		start, end := spec.Refs[0], spec.Refs[1]
		if start.Position != 0 || end.Position != 0 {
			return nil, fmt.Errorf("recap operates on whole chapters, not individual scenes; use %d-%d instead", start.Chapter, end.Chapter)
		}
		for ch := start.Chapter; ch <= end.Chapter; ch++ {
			chapterNums = append(chapterNums, ch)
		}
	default:
		return nil, fmt.Errorf("unknown range kind: %q", spec.Kind)
	}

	chapters := make([]chapterInfo, 0, len(chapterNums))
	for _, num := range chapterNums {
		idx := num - 1
		if idx < 0 || idx >= len(book.Chapters) {
			return nil, fmt.Errorf("chapter %d is out of range (1-%d)", num, len(book.Chapters))
		}
		ch := book.Chapters[idx]
		chapters = append(chapters, chapterInfo{
			number:    num,
			name:      ch.Name,
			interlude: ch.Interlude,
			slugs:     ch.Scenes,
		})
	}

	return chapters, nil
}
