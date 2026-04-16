package manuscript

import (
	"fmt"
	"strings"

	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/bookio"
)

// SearchOptions configures a manuscript search operation.
type SearchOptions struct {
	Range  string
	Query  string
	Effort agent.Effort
}

// Search runs a natural-language search across scenes in the given range.
func Search(opts SearchOptions) error {
	if strings.TrimSpace(opts.Range) == "" {
		return fmt.Errorf("range is required (e.g. 1-3, 1.1-2.3, 1,2,4)")
	}
	if strings.TrimSpace(opts.Query) == "" {
		return fmt.Errorf("search query is required")
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

	text, err := agent.ManuscriptSearch(opts.Query, paths, projectRoot, opts.Effort)
	if err != nil {
		return err
	}

	fmt.Print(strings.TrimLeft(text, "\n"))
	return nil
}
