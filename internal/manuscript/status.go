package manuscript

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/config"
)

// Status holds manuscript statistics.
type Status struct {
	Scenes           int
	Chapters         int
	Interludes       int
	WordCount        int
	EstPages         int
	UnassignedScenes []string
}

const wordsPerPage = 250

// GetStatus loads the book and computes manuscript statistics.
func GetStatus() (*Status, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return nil, err
	}

	bookFile := filepath.Join(projectRoot, config.BookFile)
	_, book, err := binder.LoadBook(bookFile)
	if err != nil {
		return nil, fmt.Errorf("loading book: %w", err)
	}

	s := &Status{}
	assignedScenes := make(map[string]bool)

	for chapter := range book.GetChapters() {
		if chapter.Heading == "" {
			s.Interludes++
		} else {
			s.Chapters++
		}
		for _, scenePath := range chapter.Scenes {
			s.Scenes++
			assignedScenes[filepath.Base(scenePath)] = true

			wc, err := countWords(scenePath)
			if err != nil {
				// Scene might not exist yet; skip
				continue
			}
			s.WordCount += wc
		}
	}

	s.EstPages = s.WordCount / wordsPerPage

	// Find unassigned scenes
	manuscriptDir := filepath.Join(projectRoot, "manuscript")
	s.UnassignedScenes, err = findUnassignedScenes(manuscriptDir, assignedScenes)
	if err != nil {
		// manuscript/ might not exist yet
		s.UnassignedScenes = nil
	}

	return s, nil
}

// FormatStatus returns a human-readable status string.
func FormatStatus(s *Status) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Scenes: %d\n", s.Scenes)
	if s.Interludes > 0 {
		fmt.Fprintf(&b, "Chapters: %d + %d interludes\n", s.Chapters, s.Interludes)
	} else {
		fmt.Fprintf(&b, "Chapters: %d\n", s.Chapters)
	}
	fmt.Fprintf(&b, "Word count: %s\n", formatNumber(s.WordCount))
	fmt.Fprintf(&b, "Est. pages: %d (%d words/page)\n", s.EstPages, wordsPerPage)
	if len(s.UnassignedScenes) > 0 {
		fmt.Fprintf(&b, "Unassigned scenes: %d (in manuscript/ but not in book.yaml)\n", len(s.UnassignedScenes))
		for _, scene := range s.UnassignedScenes {
			fmt.Fprintf(&b, "  - %s\n", scene)
		}
	}
	return b.String()
}

func countWords(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func findUnassignedScenes(manuscriptDir string, assigned map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(manuscriptDir)
	if err != nil {
		return nil, err
	}
	var unassigned []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if !assigned[e.Name()] {
			unassigned = append(unassigned, e.Name())
		}
	}
	return unassigned, nil
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
