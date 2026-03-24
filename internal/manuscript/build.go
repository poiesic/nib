package manuscript

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/config"
)

// Format represents a manuscript output format.
type Format string

const (
	FormatMD   Format = "md"
	FormatDocx Format = "docx"
	FormatPDF  Format = "pdf"
	FormatEPUB Format = "epub"
	FormatAll  Format = "all"
)

// ParseFormat parses a format string. Returns FormatMD as default.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "", "md", "markdown":
		return FormatMD, nil
	case "docx":
		return FormatDocx, nil
	case "pdf":
		return FormatPDF, nil
	case "epub":
		return FormatEPUB, nil
	case "all":
		return FormatAll, nil
	default:
		return "", fmt.Errorf("unknown format %q: valid formats are md, docx, pdf, epub, all", s)
	}
}

// Build assembles the manuscript and runs pandoc for the requested format(s).
// FormatMD performs only the assembly step and skips pandoc entirely.
// When sceneHeadings is true, scene filenames are included as ## headings.
func Build(format Format, runner CommandRunner, sceneHeadings bool) error {
	if runner == nil {
		runner = DefaultCommandRunner
	}

	// Find project root
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
	projectName := filepath.Base(projectRoot)

	// Assemble markdown
	cfg := binder.AssemblyConfig{
		InputFile:     bookFile,
		OutputDir:     outputDir,
		SceneHeadings: sceneHeadings,
	}
	_, _, err = binder.AssembleMarkdown(cfg)
	if err != nil {
		return fmt.Errorf("assembling manuscript: %w", err)
	}

	// md format: assembly only, no pandoc needed
	if format == FormatMD {
		return nil
	}

	// Check pandoc is available
	if err := CheckPandoc(runner); err != nil {
		return err
	}

	// Get chapter files
	chapterFiles, err := binder.OutputFiles(outputDir)
	if err != nil {
		return fmt.Errorf("listing chapter files: %w", err)
	}

	// Build requested format(s)
	formats := []Format{format}
	if format == FormatAll {
		formats = []Format{FormatDocx, FormatPDF, FormatEPUB}
	}

	for _, f := range formats {
		cmd, err := buildCmd(f, runner, projectRoot, outputDir, projectName, chapterFiles)
		if err != nil {
			return err
		}
		if cmd == nil {
			continue
		}
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("building %s: %w", f, err)
		}
		outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.%s", projectName, f))
		fmt.Fprintf(os.Stderr, "Built %s\n", outputFile)
	}

	return nil
}

func buildCmd(format Format, runner CommandRunner, projectRoot, outputDir, projectName string, chapterFiles []string) (*exec.Cmd, error) {
	switch format {
	case FormatDocx:
		return BuildDocx(runner, projectRoot, outputDir, projectName, chapterFiles), nil
	case FormatPDF:
		return BuildPDF(runner, projectRoot, outputDir, projectName, chapterFiles)
	case FormatEPUB:
		return BuildEPUB(runner, outputDir, projectName, chapterFiles), nil
	default:
		return nil, nil
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
