package manuscript

import (
	"fmt"
	"strings"

	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/bookio"
)

// CritiqueOptions configures a manuscript critique session.
type CritiqueOptions struct {
	Range  string
	Effort agent.Effort
}

// Critique launches an interactive Claude Code session. When the range refers
// to whole chapters (e.g. "35", "33-35", "1,3,5") it launches one
// /review-chapter session per chapter. When the range uses dotted scene refs
// (e.g. "1.2-2.1", "1.1,2.3") it launches a single /review-scene session.
func Critique(opts CritiqueOptions) error {
	if strings.TrimSpace(opts.Range) == "" {
		return fmt.Errorf("range is required (e.g. 1-3, 1.1-2.3, 1,2,4)")
	}

	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	spec, err := ParseRange(opts.Range)
	if err != nil {
		return err
	}

	paths, err := ResolveScenePaths(projectRoot, book, spec)
	if err != nil {
		return err
	}

	if isWholeChapters(spec) {
		return agent.ChapterCritique(paths, projectRoot, opts.Effort)
	}
	return agent.SceneCritique(paths, projectRoot, opts.Effort)
}

// isWholeChapters returns true if every ref in the spec is a whole-chapter
// reference (Position==0), meaning no dotted scene refs are present.
func isWholeChapters(spec RangeSpec) bool {
	for _, ref := range spec.Refs {
		if ref.Position != 0 {
			return false
		}
	}
	return len(spec.Refs) > 0
}

// ProofOptions configures a manuscript proofing session.
type ProofOptions struct {
	Range  string
	Effort agent.Effort
}

// Proof runs the copy-edit skill on the specified scenes and prints a summary.
func Proof(opts ProofOptions) error {
	if strings.TrimSpace(opts.Range) == "" {
		return fmt.Errorf("range is required (e.g. 1-3, 1.1-2.3, 1,2,4)")
	}

	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	spec, err := ParseRange(opts.Range)
	if err != nil {
		return err
	}

	paths, err := ResolveScenePaths(projectRoot, book, spec)
	if err != nil {
		return err
	}

	var text string
	if isWholeChapters(spec) {
		text, err = agent.ChapterProof(paths, projectRoot, opts.Effort)
	} else {
		text, err = agent.SceneProof(paths, projectRoot, opts.Effort)
	}
	if err != nil {
		return err
	}

	fmt.Print(strings.TrimLeft(text, "\n"))
	return nil
}
