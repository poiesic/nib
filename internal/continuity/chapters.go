package continuity

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/storydb"
)

// ChaptersOptions configures the chapters command.
type ChaptersOptions struct {
	Characters []string  // character slugs to look up
	Or         bool      // if true, query as union and output per-character results
	Pretty     bool      // if true, pretty-print JSON output
	Stdout     io.Writer // nil = os.Stdout
}

// Chapters finds scenes where the given characters are pov or present.
// In AND mode (default), outputs a JSON array of dotted refs for scenes where
// ALL characters appear. In OR mode, outputs a JSON object keyed by character
// slug with each value being that character's dotted refs.
func Chapters(opts ChaptersOptions) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	db, err := storydb.Open(projectRoot)
	if err != nil {
		return fmt.Errorf("opening storydb: %w", err)
	}
	defer db.Close()

	roles := []string{"pov", "present"}

	// Validate that all requested characters exist in storydb
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

	enc := json.NewEncoder(stdout)
	if opts.Pretty {
		enc.SetIndent("", "  ")
	}

	if opts.Or {
		return chaptersOr(db, book, opts.Characters, roles, enc)
	}
	return chaptersAnd(db, book, opts.Characters, roles, enc)
}

func chaptersAnd(db *storydb.DB, book *binder.Book, characters, roles []string, enc *json.Encoder) error {
	// Intersect: find scenes where every character appears
	var intersection map[string]bool
	for _, char := range characters {
		slugs, err := db.QuerySceneSlugsForCharactersWithRoles([]string{char}, roles)
		if err != nil {
			return fmt.Errorf("querying scenes for %s: %w", char, err)
		}
		slugSet := make(map[string]bool, len(slugs))
		for _, s := range slugs {
			slugSet[s] = true
		}
		if intersection == nil {
			intersection = slugSet
		} else {
			for s := range intersection {
				if !slugSet[s] {
					delete(intersection, s)
				}
			}
		}
	}
	if intersection == nil {
		intersection = make(map[string]bool)
	}

	refs := slugsToDotted(book, intersection)
	sort.Strings(refs)
	return enc.Encode(refs)
}

func chaptersOr(db *storydb.DB, book *binder.Book, characters, roles []string, enc *json.Encoder) error {
	result := make(map[string][]string, len(characters))
	for _, char := range characters {
		slugs, err := db.QuerySceneSlugsForCharactersWithRoles([]string{char}, roles)
		if err != nil {
			return fmt.Errorf("querying scenes for %s: %w", char, err)
		}
		slugSet := make(map[string]bool, len(slugs))
		for _, s := range slugs {
			slugSet[s] = true
		}
		refs := slugsToDotted(book, slugSet)
		sort.Strings(refs)
		result[char] = refs
	}
	return enc.Encode(result)
}

// slugsToDotted maps scene slugs to their dotted chapter.scene notation
// based on their position in the book.
func slugsToDotted(book *binder.Book, slugs map[string]bool) []string {
	var refs []string
	for ci, ch := range book.Chapters {
		for si, slug := range ch.Scenes {
			if slugs[slug] {
				refs = append(refs, fmt.Sprintf("%d.%d", ci+1, si+1))
			}
		}
	}
	return refs
}
