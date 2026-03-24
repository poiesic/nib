package manuscript

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed filters/pdf.lua
var pdfFilter []byte

// CommandRunner creates an exec.Cmd. Injected for testing.
type CommandRunner func(name string, args ...string) *exec.Cmd

// DefaultCommandRunner uses os/exec directly.
func DefaultCommandRunner(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

var ErrPandocNotFound = errors.New("pandoc not found on PATH; install pandoc to build manuscripts")

// CheckPandoc verifies that pandoc is available.
func CheckPandoc(runner CommandRunner) error {
	cmd := runner("pandoc", "--version")
	if err := cmd.Run(); err != nil {
		return ErrPandocNotFound
	}
	return nil
}

// BuildDocx builds a DOCX manuscript. If pandoc-templates/ exists in the
// project root, uses the Shunn long manuscript format via md2long.sh.
// Otherwise falls back to plain pandoc.
func BuildDocx(runner CommandRunner, projectRoot, outputDir, projectName string, chapterFiles []string) *exec.Cmd {
	metadataFile := filepath.Join(outputDir, "metadata.yaml")
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.docx", projectName))

	shunnScript := filepath.Join(projectRoot, "pandoc-templates", "bin", "md2long.sh")
	if fileExists(shunnScript) {
		args := []string{"-x", "-r", outputDir, "-o", outputFile, metadataFile}
		args = append(args, chapterFiles...)
		return runner(shunnScript, args...)
	}

	args := []string{metadataFile}
	args = append(args, chapterFiles...)
	args = append(args, "--to=docx", "-o", outputFile)
	return runner("pandoc", args...)
}

// BuildPDF builds a PDF manuscript using xelatex.
func BuildPDF(runner CommandRunner, projectRoot, outputDir, projectName string, chapterFiles []string) (*exec.Cmd, error) {
	metadataFile := filepath.Join(outputDir, "metadata.yaml")
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.pdf", projectName))

	// Write embedded lua filter to a temp file
	filterFile, err := os.CreateTemp("", "scrib-pdf-*.lua")
	if err != nil {
		return nil, fmt.Errorf("creating temp filter: %w", err)
	}
	if _, err := filterFile.Write(pdfFilter); err != nil {
		os.Remove(filterFile.Name())
		return nil, fmt.Errorf("writing temp filter: %w", err)
	}
	filterFile.Close()

	// Write LaTeX header to a temp file (page break before each chapter)
	headerFile, err := os.CreateTemp("", "scrib-pdf-*.tex")
	if err != nil {
		os.Remove(filterFile.Name())
		return nil, fmt.Errorf("creating temp header: %w", err)
	}
	if _, err := headerFile.WriteString("\\usepackage{titlesec}\n\\newcommand{\\sectionbreak}{\\clearpage}\n\\usepackage{fontspec}\n\\setmainfont{Crimson Text}\n"); err != nil {
		os.Remove(filterFile.Name())
		os.Remove(headerFile.Name())
		return nil, fmt.Errorf("writing temp header: %w", err)
	}
	headerFile.Close()

	args := []string{metadataFile}
	args = append(args, chapterFiles...)
	args = append(args,
		"--from=markdown-implicit_figures",
		fmt.Sprintf("--resource-path=%s:%s", outputDir, projectRoot),
		"--pdf-engine=xelatex",
		"--lua-filter", filterFile.Name(),
		"-V", "geometry:margin=1in",
		"-V", "fontsize=12pt",
		"-V", "linestretch=1",
		"-H", headerFile.Name(),
		"-o", outputFile,
	)
	return runner("pandoc", args...), nil
}

// BuildEPUB builds an EPUB manuscript.
func BuildEPUB(runner CommandRunner, outputDir, projectName string, chapterFiles []string) *exec.Cmd {
	metadataFile := filepath.Join(outputDir, "metadata.yaml")
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.epub", projectName))

	args := []string{metadataFile}
	args = append(args, chapterFiles...)
	args = append(args, "-o", outputFile)
	return runner("pandoc", args...)
}
