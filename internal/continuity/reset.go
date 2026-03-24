package continuity

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/poiesic/nib/internal/config"
	"github.com/poiesic/nib/internal/storydb"
)

// ResetOptions configures the reset command.
type ResetOptions struct {
	Yes    bool      // skip confirmation prompt
	Stdin  io.Reader // nil = os.Stdin
	Stdout io.Writer // nil = os.Stdout
}

// Reset clears all storydb tables after prompting for confirmation.
func Reset(opts ResetOptions) error {
	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	if !opts.Yes {
		fmt.Fprint(stdout, "This will delete all indexed continuity data. Continue? [y/N] ")
		scanner := bufio.NewScanner(stdin)
		if !scanner.Scan() {
			return nil
		}
		answer := strings.TrimSpace(scanner.Text())
		if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
			fmt.Fprintln(stdout, "Aborted.")
			return nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return err
	}

	db, err := storydb.Open(projectRoot)
	if err != nil {
		return fmt.Errorf("opening storydb: %w", err)
	}
	defer db.Close()

	if err := db.Reset(); err != nil {
		return fmt.Errorf("resetting storydb: %w", err)
	}

	fmt.Fprintln(stdout, "Storydb reset.")
	return nil
}
