package continuity

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/config"
	"github.com/poiesic/nib/internal/manuscript"
	"github.com/poiesic/nib/internal/storydb"
)

// CharactersOptions configures the characters command.
type CharactersOptions struct {
	Range  string    // optional range expression (e.g. "1", "3-5", "1.2")
	All    bool      // if true, include mentioned characters; default is pov+present only
	Pretty bool      // if true, pretty-print JSON output
	Stdout io.Writer // nil = os.Stdout
}

// Characters queries storydb for unique characters, sorted alphabetically,
// and writes the result as a JSON array to stdout. When Range is set, only
// characters from scenes within that range are included.
func Characters(opts CharactersOptions) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	var roles []string
	if !opts.All {
		roles = []string{"pov", "present"}
	}

	enc := json.NewEncoder(stdout)
	if opts.Pretty {
		enc.SetIndent("", "  ")
	}

	if opts.Range != "" {
		return charactersInRange(opts.Range, roles, enc)
	}
	return charactersAll(roles, enc)
}

func charactersAll(roles []string, enc *json.Encoder) error {
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

	characters, err := db.QueryDistinctCharacters(roles)
	if err != nil {
		return fmt.Errorf("querying characters: %w", err)
	}

	return enc.Encode(characters)
}

func charactersInRange(rangeExpr string, roles []string, enc *json.Encoder) error {
	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	spec, err := manuscript.ParseRange(rangeExpr)
	if err != nil {
		return err
	}

	resolved, err := manuscript.ResolveSlugs(book, spec)
	if err != nil {
		return err
	}

	slugs := make([]string, len(resolved))
	for i, r := range resolved {
		slugs[i] = r.Slug
	}

	db, err := storydb.Open(projectRoot)
	if err != nil {
		return fmt.Errorf("opening storydb: %w", err)
	}
	defer db.Close()

	characters, err := db.QueryDistinctCharactersBySlugs(slugs, roles)
	if err != nil {
		return fmt.Errorf("querying characters: %w", err)
	}

	return enc.Encode(characters)
}
