package continuity

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/manuscript"
)

// CheckOptions configures the continuity check operation.
type CheckOptions struct {
	Range  string
	Effort agent.Effort
}

// Check runs the continuity-check skill on the specified scenes and prints findings.
func Check(opts CheckOptions) error {
	if strings.TrimSpace(opts.Range) == "" {
		return fmt.Errorf("range is required (e.g. 1-3, 1.1-2.3, 1,2,4)")
	}

	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	spec, err := manuscript.ParseRange(opts.Range)
	if err != nil {
		return err
	}

	paths, err := manuscript.ResolveScenePaths(projectRoot, book, spec)
	if err != nil {
		return err
	}

	text, err := agent.ContinuityCheck(paths, projectRoot, opts.Effort)
	if err != nil {
		return err
	}

	fmt.Print(strings.TrimLeft(text, "\n"))
	return nil
}

// AskOptions configures the ask operation.
type AskOptions struct {
	Question string
	Range    string    // optional range to scope context
	Stdout   io.Writer // nil = os.Stdout
	Effort   agent.Effort
}

// Ask sends a plain-English question about the novel to the agent and prints the answer.
func Ask(opts AskOptions) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	if strings.TrimSpace(opts.Question) == "" {
		return fmt.Errorf("question is required")
	}

	projectRoot, _, _, err := bookio.Load()
	if err != nil {
		return err
	}

	text, err := agent.ContinuityAsk(opts.Question, opts.Range, projectRoot, opts.Effort)
	if err != nil {
		return err
	}

	fmt.Fprint(stdout, strings.TrimLeft(text, "\n"))
	return nil
}
