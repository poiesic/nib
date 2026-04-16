package manuscript

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/config"
)

// FullManuscriptFile is the name of the assembled single-file manuscript
// written into build/ for book-scope critique.
const FullManuscriptFile = "manuscript-full.md"

// CritiqueBookOptions configures a whole-manuscript critique.
type CritiqueBookOptions struct {
	Effort agent.Effort
}

// CritiqueBook assembles the full manuscript into a single markdown file and
// launches an interactive manuscript-critique session. Assembly mirrors
// `nib ma build md`: binder wipes the output dir, writes per-chapter files,
// and we concatenate them in order into build/manuscript-full.md so the agent
// can read the whole book as one object.
func CritiqueBook(opts CritiqueBookOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectRoot, err := config.FindProjectRoot(cwd)
	if err != nil {
		return err
	}

	bookFile := filepath.Join(projectRoot, config.BookFile)
	outputDir := filepath.Join(projectRoot, "build")

	if _, _, err := binder.AssembleMarkdown(binder.AssemblyConfig{
		InputFile: bookFile,
		OutputDir: outputDir,
	}); err != nil {
		return fmt.Errorf("assembling manuscript: %w", err)
	}

	chapterFiles, err := binder.OutputFiles(outputDir)
	if err != nil {
		return fmt.Errorf("listing chapter files: %w", err)
	}
	if len(chapterFiles) == 0 {
		return fmt.Errorf("no chapters to critique: assembly produced no files")
	}

	fullPath := filepath.Join(outputDir, FullManuscriptFile)
	if err := concatChapters(chapterFiles, fullPath); err != nil {
		return fmt.Errorf("writing full manuscript: %w", err)
	}

	return agent.ManuscriptCritique([]string{fullPath}, projectRoot, opts.Effort)
}

// concatChapters writes the contents of chapterFiles (in order) to destPath,
// separating adjacent chapters with a blank line. destPath is overwritten.
func concatChapters(chapterFiles []string, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	for i, path := range chapterFiles {
		if i > 0 {
			if _, err := io.WriteString(out, "\n\n"); err != nil {
				return err
			}
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
	}
	return nil
}
