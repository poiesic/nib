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
	Range string
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

	prompt := fmt.Sprintf("/continuity-check %s", strings.Join(paths, " "))

	text, err := agent.Complete(prompt, "high", []string{"Read", "Bash"}, projectRoot)
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

	prompt := buildAskPrompt(opts.Question, opts.Range)

	text, err := agent.Complete(prompt, "high", []string{"Read", "Bash"}, projectRoot)
	if err != nil {
		return err
	}

	fmt.Fprint(stdout, strings.TrimLeft(text, "\n"))
	return nil
}

func buildAskPrompt(question, rangeExpr string) string {
	var b strings.Builder

	b.WriteString("You are a research assistant for a novel manuscript managed by `scrib`.\n\n")

	b.WriteString("## How to find information\n\n")
	b.WriteString("You have access to these tools for locating information in the manuscript:\n\n")

	b.WriteString("### Scrib commands (run via Bash)\n")
	b.WriteString("- `nib ct recap <range>` -- JSON summaries of scenes in a chapter range (e.g. `1-5`, `3`, `1,3,5`)\n")
	b.WriteString("- `nib ct recap <range> --detailed` -- includes facts, locations, dates, times, and all character appearances\n")
	b.WriteString("- `nib ct recap <range> -c <character-slug>` -- filter recap to scenes involving a character (repeatable)\n")
	b.WriteString("- `nib ct characters` -- list all known character slugs from indexed data\n")
	b.WriteString("- `nib ct characters <range>` -- characters in a specific chapter range\n")
	b.WriteString("- `nib ct chapters <character> [character...]` -- find scenes where characters appear together (AND)\n")
	b.WriteString("- `nib ct chapters --or <character> [character...]` -- find scenes per character (OR)\n")
	b.WriteString("- `nib ma status` -- manuscript statistics (chapters, scenes, word count)\n\n")

	b.WriteString("### File access (via Read)\n")
	b.WriteString("- `scenes/<slug>.md` -- scene prose files\n")
	b.WriteString("- `characters/<slug>.yaml` -- character profiles\n")
	b.WriteString("- `book.yaml` -- chapter structure and scene ordering\n")
	b.WriteString("- `storydb/` -- CSV files with indexed continuity data\n\n")

	b.WriteString("## Strategy\n\n")
	b.WriteString("1. Start with the scrib commands to locate relevant scenes and data.\n")
	b.WriteString("2. Read specific scene files only when you need the actual prose (exact wording, dialogue, descriptions).\n")
	b.WriteString("3. Use character profiles when the question involves character traits, background, or relationships.\n")
	b.WriteString("4. Synthesize a clear, direct answer. Cite specific scenes (by slug or chapter.scene number) when relevant.\n\n")

	b.WriteString("## Rules\n\n")
	b.WriteString("- Answer the question directly. No preamble, no restating the question.\n")
	b.WriteString("- Cite evidence from the manuscript. Quote relevant passages when they strengthen the answer.\n")
	b.WriteString("- If the indexed data doesn't cover the answer, say so -- don't guess.\n")
	b.WriteString("- Keep the answer concise but complete.\n\n")

	if rangeExpr != "" {
		fmt.Fprintf(&b, "## Scope\n\nLimit your search to chapters/scenes in range: %s\n\n", rangeExpr)
	}

	fmt.Fprintf(&b, "## Question\n\n%s\n", question)

	return b.String()
}
