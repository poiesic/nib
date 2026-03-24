package bookio

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/config"
	"gopkg.in/yaml.v3"
)

// Load finds the project root and loads book.yaml, returning the project root,
// front matter, and book. Callers pass projectRoot to Save after modifications.
func Load() (string, *binder.FrontMatter, *binder.Book, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, nil, err
	}
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return "", nil, nil, err
	}
	bookFile := filepath.Join(projectRoot, config.BookFile)
	fm, book, err := binder.LoadBook(bookFile)
	if err != nil {
		return "", nil, nil, fmt.Errorf("loading book: %w", err)
	}
	return projectRoot, fm, book, nil
}

type bookSpec struct {
	Book bookSpec_inner `yaml:"book"`
}

type bookSpec_inner struct {
	BaseDir  string           `yaml:"base_dir"`
	Chapters []binder.Chapter `yaml:"chapters"`
}

// Save writes front matter and book back to book.yaml as a two-document YAML stream.
// It converts book.BaseDir back to a relative path before writing.
func Save(projectRoot string, fm *binder.FrontMatter, book *binder.Book) error {
	baseDir := book.BaseDir
	if filepath.IsAbs(baseDir) {
		rel, err := filepath.Rel(projectRoot, baseDir)
		if err != nil {
			return fmt.Errorf("making base_dir relative: %w", err)
		}
		baseDir = rel
	}

	spec := bookSpec{
		Book: bookSpec_inner{
			BaseDir:  baseDir,
			Chapters: book.Chapters,
		},
	}

	bookFile := filepath.Join(projectRoot, config.BookFile)
	f, err := os.Create(bookFile)
	if err != nil {
		return fmt.Errorf("creating book file: %w", err)
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	defer encoder.Close()

	if err := encoder.Encode(fm); err != nil {
		return fmt.Errorf("encoding front matter: %w", err)
	}
	if err := encoder.Encode(spec); err != nil {
		return fmt.Errorf("encoding book: %w", err)
	}

	return nil
}
