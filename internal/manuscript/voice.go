package manuscript

import (
	"fmt"
	"os"
	"strings"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/storydb"
)

// VoiceOptions configures the voice check operation.
type VoiceOptions struct {
	Characters []string
	Thorough   bool // sample 60% instead of 30%
	Effort     agent.Effort
}

// Voice checks character voice consistency across sampled scenes.
func Voice(opts VoiceOptions) error {
	if len(opts.Characters) == 0 {
		return fmt.Errorf("at least one character slug is required")
	}

	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	db, err := storydb.Open(projectRoot)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, slug := range opts.Characters {
		scenes, err := db.QuerySceneSlugsForCharacters([]string{slug})
		if err != nil {
			return fmt.Errorf("querying scenes for %s: %w", slug, err)
		}
		if len(scenes) == 0 {
			return fmt.Errorf("no indexed scenes found for character %q; run nib ct index first", slug)
		}

		sampled := sampleScenes(scenes, opts.Thorough)

		// Resolve sampled slugs to file paths
		spec, err := slugsToSpec(book, sampled)
		if err != nil {
			return err
		}
		paths, err := ResolveScenePaths(projectRoot, book, spec)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Checking %s: sampling %d of %d scenes\n",
			slug, len(sampled), len(scenes))

		text, err := agent.VoiceCheck(slug, paths, projectRoot, opts.Effort)
		if err != nil {
			return err
		}

		fmt.Print(strings.TrimLeft(text, "\n"))
	}

	return nil
}

// sampleScenes selects scenes for voice checking, distributed across the
// manuscript timeline (first, last, and evenly spaced in between).
//
// Default: ~30% with floor of 6, cap of 12.
// Thorough: ~60% with floor of 6, cap of 24.
// < 6 scenes: always returns all.
func sampleScenes(scenes []string, thorough bool) []string {
	n := len(scenes)
	if n <= 5 {
		return scenes
	}

	pct := 30
	cap := 12
	if thorough {
		pct = 60
		cap = 24
	}

	target := n * pct / 100
	if target < 6 {
		target = 6
	}
	if target > cap {
		target = cap
	}
	if target >= n {
		return scenes
	}

	sampled := make([]string, 0, target)
	sampled = append(sampled, scenes[0])

	// Evenly spaced interior scenes
	interior := target - 2
	for i := 1; i <= interior; i++ {
		idx := i * (n - 1) / (interior + 1)
		sampled = append(sampled, scenes[idx])
	}

	sampled = append(sampled, scenes[n-1])
	return sampled
}

// slugsToSpec converts a list of scene slugs into a RangeSpec by looking
// up each slug's chapter and position in the book.
func slugsToSpec(book *binder.Book, slugs []string) (RangeSpec, error) {
	refs := make([]SceneRef, 0, len(slugs))
	for _, slug := range slugs {
		found := false
		for i, ch := range book.Chapters {
			for j, s := range ch.Scenes {
				if s == slug {
					refs = append(refs, SceneRef{Chapter: i + 1, Position: j + 1})
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return RangeSpec{}, fmt.Errorf("scene %q not found in book.yaml", slug)
		}
	}
	return RangeSpec{Kind: "list", Refs: refs}, nil
}
