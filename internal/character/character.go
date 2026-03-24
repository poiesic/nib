package character

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/continuity"
	"github.com/poiesic/nib/internal/scene"
)

// CommandRunner creates an exec.Cmd. Injected for testing.
type CommandRunner func(name string, args ...string) *exec.Cmd

var validSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var ErrEditorNotSet = errors.New("no editor set; set NIB_EDITOR, VISUAL, or EDITOR")

// CharacterInfo holds display information about a character.
type CharacterInfo struct {
	Slug string
	Name string
}

// Add creates a new character YAML file with a scaffold of common fields.
func Add(slug string) (string, error) {
	if !validSlug.MatchString(slug) {
		return "", fmt.Errorf("invalid character slug %q: must be lowercase alphanumeric with hyphens", slug)
	}

	projectRoot, _, _, err := bookio.Load()
	if err != nil {
		return "", err
	}

	charDir := filepath.Join(projectRoot, "characters")
	if err := os.MkdirAll(charDir, 0755); err != nil {
		return "", fmt.Errorf("creating characters directory: %w", err)
	}

	path := filepath.Join(charDir, slug+".yaml")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("character %q already exists", slug)
	}

	scaffold := `---
name: ""
age: ""
location: ""
occupation: ""

role: |


background: |


goal: ""

values:
  - ""

personality: |


habits:
  - ""

relationships: {}
---
`

	if err := os.WriteFile(path, []byte(scaffold), 0644); err != nil {
		return "", fmt.Errorf("writing character file: %w", err)
	}

	return path, nil
}

// List returns info about all characters in the characters/ directory.
func List() ([]CharacterInfo, error) {
	projectRoot, _, _, err := bookio.Load()
	if err != nil {
		return nil, err
	}

	pattern := filepath.Join(projectRoot, "characters", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var infos []CharacterInfo
	for _, m := range matches {
		slug := strings.TrimSuffix(filepath.Base(m), ".yaml")
		name := readName(m)
		infos = append(infos, CharacterInfo{Slug: slug, Name: name})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Slug < infos[j].Slug
	})

	return infos, nil
}

// Remove deletes a character YAML file.
func Remove(slug string) error {
	projectRoot, _, _, err := bookio.Load()
	if err != nil {
		return err
	}

	path := filepath.Join(projectRoot, "characters", slug+".yaml")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("character %q not found", slug)
	}

	return os.Remove(path)
}

// EditOptions configures how a character file is opened in an editor.
type EditOptions struct {
	Slug   string
	Runner CommandRunner // nil = exec.Command
}

// Edit opens a character YAML file in the user's preferred editor.
func Edit(opts EditOptions) error {
	projectRoot, _, _, err := bookio.Load()
	if err != nil {
		return err
	}

	path := filepath.Join(projectRoot, "characters", opts.Slug+".yaml")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("character %q not found", opts.Slug)
	}

	editor := editorFromEnv()
	if editor == "" {
		return ErrEditorNotSet
	}

	runner := opts.Runner
	if runner == nil {
		runner = exec.Command
	}

	parts := strings.Fields(editor)
	cmd := runner(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// TalkOptions configures the character talk session.
type TalkOptions struct {
	Slug   string
	Scene  string // dotted notation (e.g. "37.2")
	Resume bool   // resume an existing talk session
	New    bool   // delete existing session and start fresh
}

// Talk launches an interactive agent session where the agent role-plays
// as the specified character at a specific point in the story.
func Talk(opts TalkOptions) error {
	projectRoot, _, _, err := bookio.Load()
	if err != nil {
		return err
	}

	_, _, err = scene.ParseDotted(opts.Scene)
	if err != nil {
		return fmt.Errorf("invalid scene reference: %w", err)
	}

	sessionID := talkSessionID(opts.Slug, opts.Scene)

	session := &agent.SessionOptions{
		ID:     sessionID,
		Resume: opts.Resume,
		New:    opts.New,
	}

	if opts.Resume {
		fmt.Fprintln(os.Stderr, "Resuming conversation...")
		return agent.Converse("", agent.ConverseOptions{Session: session}, projectRoot)
	}

	// Verify character exists and read profile
	profilePath := filepath.Join(projectRoot, "characters", opts.Slug+".yaml")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("character %q not found", opts.Slug)
		}
		return fmt.Errorf("reading character profile: %w", err)
	}

	name := readName(profilePath)
	if name == "" {
		name = opts.Slug
	}

	// Build character-filtered recap through the specified chapter
	chapterNum, _, _ := scene.ParseDotted(opts.Scene)
	var recapBuf bytes.Buffer
	recapRange := fmt.Sprintf("1-%d", chapterNum)
	if chapterNum == 1 {
		recapRange = "1"
	}
	recapErr := continuity.Recap(continuity.RecapOptions{
		Range:      recapRange,
		Characters: []string{opts.Slug},
		Detailed:   true,
		Stdout:     &recapBuf,
		Stderr:     io.Discard,
	})

	prompt := buildTalkPrompt(name, opts.Scene, string(profileData), recapBuf.String(), recapErr)

	fmt.Fprintf(os.Stderr, "Resume with: nib pr talk --resume %s %s\n\n", opts.Slug, opts.Scene)
	return agent.Converse(prompt, agent.ConverseOptions{Session: session}, projectRoot)
}

func buildTalkPrompt(name, sceneRef, profile, recap string, recapErr error) string {
	var b strings.Builder

	fmt.Fprintf(&b, "You are %s from the novel being written. ", name)
	fmt.Fprintf(&b, "The writer wants to interview you in character at the point in the story through scene %s.\n\n", sceneRef)

	b.WriteString("## Character Profile\n\n")
	b.WriteString("```yaml\n")
	b.WriteString(profile)
	b.WriteString("```\n\n")

	if recapErr != nil {
		fmt.Fprintf(&b, "## Story Context\n\nNo indexed continuity data available (scenes may not be indexed yet). ")
		fmt.Fprintf(&b, "Use only the character profile above for context.\n\n")
	} else if recap != "" {
		fmt.Fprintf(&b, "## Story So Far (through scene %s)\n\n", sceneRef)
		b.WriteString("The following is a detailed recap of events filtered to scenes where you appear:\n\n")
		b.WriteString("```json\n")
		b.WriteString(recap)
		b.WriteString("```\n\n")
	}

	b.WriteString("## Instructions\n\n")
	fmt.Fprintf(&b, "- Stay in character as %s at all times\n", name)
	fmt.Fprintf(&b, "- Your knowledge of events stops at scene %s — you don't know what happens after\n", sceneRef)
	fmt.Fprintf(&b, "- Answer the writer's questions as %s would, using their vocabulary, mannerisms, and worldview\n", name)
	b.WriteString("- Express emotions, opinions, and reactions consistent with the character profile\n")
	b.WriteString("- If the writer asks about events you haven't experienced yet, say you don't know\n")
	b.WriteString("- Stay grounded — no embellishment beyond what the profile and recap establish\n")
	b.WriteString("- All context you need is provided above. Do not search for files or run commands\n")

	return b.String()
}

// FormatList formats character info as tab-separated output.
func FormatList(chars []CharacterInfo) string {
	if len(chars) == 0 {
		return "No characters\n"
	}

	var b strings.Builder
	for _, c := range chars {
		fmt.Fprintf(&b, "%s\t%s\n", c.Slug, c.Name)
	}
	return b.String()
}

// readName extracts the name field from a character YAML file.
func readName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return ""
	}
	return doc.Name
}

// talkSessionID generates a deterministic UUID v5 from the character slug
// and scene reference, so the same interview always maps to the same session.
func talkSessionID(slug, sceneRef string) string {
	// Fixed namespace UUID for nib interviews
	namespace := [16]byte{
		0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1,
		0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8,
	}
	name := fmt.Sprintf("interview-%s-%s", slug, sceneRef)

	h := sha1.New()
	h.Write(namespace[:])
	h.Write([]byte(name))
	sum := h.Sum(nil)

	// Set version 5
	sum[6] = (sum[6] & 0x0f) | 0x50
	// Set variant (RFC 4122)
	sum[8] = (sum[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func editorFromEnv() string {
	for _, key := range []string{"NIB_EDITOR", "VISUAL", "EDITOR"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}
