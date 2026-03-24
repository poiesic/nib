package continuity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/poiesic/nib/internal/storydb"
)

// reviewProposals presents each extracted record for line-by-line approval.
// Returns a new ExtractionResult containing only accepted records.
func reviewProposals(result *ExtractionResult, stdin io.Reader, stdout io.Writer) (*ExtractionResult, error) {
	reader := bufio.NewReader(stdin)
	accepted := &ExtractionResult{}
	acceptAll := false

	// Review scene metadata
	fmt.Fprintf(stdout, "\n--- Scene Metadata ---\n")
	fmt.Fprintf(stdout, "  Scene: %s\n", result.Scene.Scene)
	fmt.Fprintf(stdout, "  POV: %s | Type: %s | Location: %s\n", result.Scene.POV, result.Scene.SceneType, result.Scene.Location)
	if result.Scene.Date != "" || result.Scene.Time != "" {
		fmt.Fprintf(stdout, "  Date: %s | Time: %s\n", result.Scene.Date, result.Scene.Time)
	}
	fmt.Fprintf(stdout, "  Summary: %s\n", result.Scene.Summary)

	action, err := promptAction(reader, stdout)
	if err != nil {
		return nil, err
	}
	switch action {
	case "a":
		accepted.Scene = result.Scene
	case "A":
		accepted.Scene = result.Scene
		acceptAll = true
	case "e":
		edited, err := editRecord(result.Scene)
		if err != nil {
			return nil, err
		}
		accepted.Scene = edited.(storydb.Scene)
	case "r":
		// Scene metadata is required — reject means skip the whole scene
		return nil, fmt.Errorf("scene metadata rejected; nothing to index")
	}

	// Review facts
	if len(result.Facts) > 0 {
		fmt.Fprintf(stdout, "\n--- Facts (%d) ---\n", len(result.Facts))
		for _, fact := range result.Facts {
			fmt.Fprintf(stdout, "  [%s] %s: %s\n", fact.ID, fact.Category, fact.Summary)
			fmt.Fprintf(stdout, "    Detail: %s\n", fact.Detail)
			fmt.Fprintf(stdout, "    Source: %q\n", fact.SourceText)

			if acceptAll {
				fmt.Fprintf(stdout, "  [auto-accepted]\n")
				accepted.Facts = append(accepted.Facts, fact)
				continue
			}

			action, err := promptAction(reader, stdout)
			if err != nil {
				return nil, err
			}
			switch action {
			case "a":
				accepted.Facts = append(accepted.Facts, fact)
			case "A":
				accepted.Facts = append(accepted.Facts, fact)
				acceptAll = true
			case "e":
				edited, err := editRecord(fact)
				if err != nil {
					return nil, err
				}
				accepted.Facts = append(accepted.Facts, edited.(storydb.Fact))
			case "r":
				// skip
			}
		}
	}

	// Review characters
	if len(result.Characters) > 0 {
		fmt.Fprintf(stdout, "\n--- Characters (%d) ---\n", len(result.Characters))
		for _, ch := range result.Characters {
			fmt.Fprintf(stdout, "  %s (%s)\n", ch.Character, ch.Role)

			if acceptAll {
				fmt.Fprintf(stdout, "  [auto-accepted]\n")
				accepted.Characters = append(accepted.Characters, ch)
				continue
			}

			action, err := promptAction(reader, stdout)
			if err != nil {
				return nil, err
			}
			switch action {
			case "a":
				accepted.Characters = append(accepted.Characters, ch)
			case "A":
				accepted.Characters = append(accepted.Characters, ch)
				acceptAll = true
			case "e":
				edited, err := editRecord(ch)
				if err != nil {
					return nil, err
				}
				accepted.Characters = append(accepted.Characters, edited.(storydb.SceneCharacter))
			case "r":
				// skip
			}
		}
	}

	// Review locations
	if len(result.Locations) > 0 {
		fmt.Fprintf(stdout, "\n--- Locations (%d) ---\n", len(result.Locations))
		for _, loc := range result.Locations {
			fmt.Fprintf(stdout, "  %s: %s (%s)\n", loc.ID, loc.Name, loc.Type)
			fmt.Fprintf(stdout, "    %s\n", loc.Description)

			if acceptAll {
				fmt.Fprintf(stdout, "  [auto-accepted]\n")
				accepted.Locations = append(accepted.Locations, loc)
				continue
			}

			action, err := promptAction(reader, stdout)
			if err != nil {
				return nil, err
			}
			switch action {
			case "a":
				accepted.Locations = append(accepted.Locations, loc)
			case "A":
				accepted.Locations = append(accepted.Locations, loc)
				acceptAll = true
			case "e":
				edited, err := editRecord(loc)
				if err != nil {
					return nil, err
				}
				accepted.Locations = append(accepted.Locations, edited.(storydb.Location))
			case "r":
				// skip
			}
		}
	}

	return accepted, nil
}

// promptAction reads a single-character action from the user.
func promptAction(reader *bufio.Reader, stdout io.Writer) (string, error) {
	fmt.Fprintf(stdout, "  [a]ccept  [A]ccept all  [r]eject  [e]dit > ")
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			// Treat EOF as accept (non-interactive mode)
			fmt.Fprintln(stdout)
			return "a", nil
		}
		return "", err
	}
	action := strings.TrimSpace(line)
	switch action {
	case "A":
		return "A", nil
	case "a", "accept":
		return "a", nil
	case "r", "R", "reject":
		return "r", nil
	case "e", "E", "edit":
		return "e", nil
	default:
		// Default to accept for unrecognized input
		return "a", nil
	}
}

// editRecord opens $EDITOR with the record as JSON. Returns the edited record
// as the same type (via type assertion by caller).
func editRecord(record any) (any, error) {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling for edit: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "scrib-edit-*.json")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return nil, err
	}
	tmpFile.Close()

	editor := editorFromEnv()
	if editor == "" {
		return nil, fmt.Errorf("no editor set; set NIB_EDITOR, VISUAL, or EDITOR to use edit mode")
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor failed: %w", err)
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	// Unmarshal back into the correct type
	switch record.(type) {
	case storydb.Scene:
		var s storydb.Scene
		if err := json.Unmarshal(edited, &s); err != nil {
			return nil, fmt.Errorf("invalid JSON after edit: %w", err)
		}
		return s, nil
	case storydb.Fact:
		var f storydb.Fact
		if err := json.Unmarshal(edited, &f); err != nil {
			return nil, fmt.Errorf("invalid JSON after edit: %w", err)
		}
		return f, nil
	case storydb.SceneCharacter:
		var sc storydb.SceneCharacter
		if err := json.Unmarshal(edited, &sc); err != nil {
			return nil, fmt.Errorf("invalid JSON after edit: %w", err)
		}
		return sc, nil
	case storydb.Location:
		var l storydb.Location
		if err := json.Unmarshal(edited, &l); err != nil {
			return nil, fmt.Errorf("invalid JSON after edit: %w", err)
		}
		return l, nil
	default:
		return nil, fmt.Errorf("unknown record type for editing")
	}
}

func editorFromEnv() string {
	for _, key := range []string{"NIB_EDITOR", "VISUAL", "EDITOR"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}
