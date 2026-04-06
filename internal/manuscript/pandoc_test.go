package manuscript

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockRunner(name string, args ...string) *exec.Cmd {
	// Create a command that records what was called but doesn't execute
	return exec.Command("echo", append([]string{name}, args...)...)
}

func TestBuildDocx_PlainPandoc(t *testing.T) {
	chapterFiles := []string{"build/001-chapter-one.md", "build/002-chapter-two.md"}
	cmd := BuildDocx(mockRunner, "/project", "build", "my-novel", chapterFiles)

	args := cmd.Args
	// Should use pandoc (not Shunn) since pandoc-templates/ doesn't exist
	assert.Equal(t, "echo", args[0])
	assert.Equal(t, "pandoc", args[1])
	assert.Contains(t, args, "build/metadata.yaml")
	assert.Contains(t, args, "--to=docx")
	assert.Contains(t, args, "-o")
	assert.Contains(t, args, "build/my-novel.docx")
	assert.Contains(t, args, "build/001-chapter-one.md")
	assert.Contains(t, args, "build/002-chapter-two.md")
}

func TestBuildPDF(t *testing.T) {
	chapterFiles := []string{"build/001-chapter-one.md"}
	cmd, err := BuildPDF(mockRunner, "/project", "build", "my-novel", chapterFiles)
	assert.NoError(t, err)

	args := cmd.Args
	assert.Equal(t, "echo", args[0])
	assert.Equal(t, "pandoc", args[1])
	assert.Contains(t, args, "build/metadata.yaml")
	assert.Contains(t, args, "--pdf-engine=xelatex")
	assert.Contains(t, args, "--resource-path=build:/project")
	assert.Contains(t, args, "--lua-filter")
	assert.Contains(t, args, "-V")
	assert.Contains(t, args, "geometry:margin=1in")
	assert.Contains(t, args, "fontsize=12pt")
	assert.Contains(t, args, "linestretch=1")
	assert.Contains(t, args, "-o")
	assert.Contains(t, args, "build/my-novel.pdf")
}

func TestBuildEPUB(t *testing.T) {
	chapterFiles := []string{"build/001-chapter-one.md"}
	cmd := BuildEPUB(mockRunner, "/project", "build", "my-novel", chapterFiles)

	args := cmd.Args
	assert.Equal(t, "echo", args[0])
	assert.Equal(t, "pandoc", args[1])
	assert.Contains(t, args, "build/metadata.yaml")
	assert.Contains(t, args, "--resource-path=build:/project")
	assert.Contains(t, args, "-o")
	assert.Contains(t, args, "build/my-novel.epub")
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		wantErr  bool
	}{
		{"", FormatMD, false},
		{"md", FormatMD, false},
		{"markdown", FormatMD, false},
		{"MD", FormatMD, false},
		{"docx", FormatDocx, false},
		{"DOCX", FormatDocx, false},
		{"pdf", FormatPDF, false},
		{"epub", FormatEPUB, false},
		{"all", FormatAll, false},
		{"invalid", "", true},
	}
	for _, tt := range tests {
		f, err := ParseFormat(tt.input)
		if tt.wantErr {
			assert.Error(t, err, "input %q should produce error", tt.input)
		} else {
			assert.NoError(t, err, "input %q should not produce error", tt.input)
			assert.Equal(t, tt.expected, f)
		}
	}
}

func TestCheckPandoc_NotFound(t *testing.T) {
	notFoundRunner := func(name string, args ...string) *exec.Cmd {
		return exec.Command("nonexistent-binary-xxx")
	}
	err := CheckPandoc(notFoundRunner)
	assert.ErrorIs(t, err, ErrPandocNotFound)
}

func TestBuildDocx_ShunnTemplate(t *testing.T) {
	// Create a temp project with pandoc-templates/bin/md2long.sh
	dir := t.TempDir()
	shunnDir := dir + "/pandoc-templates/bin"
	assert.NoError(t, mkdirAll(shunnDir))
	assert.NoError(t, writeFile(shunnDir+"/md2long.sh", "#!/bin/sh\necho shunn"))
	assert.NoError(t, chmodExec(shunnDir+"/md2long.sh"))

	chapterFiles := []string{"build/001-chapter-one.md"}
	cmd := BuildDocx(mockRunner, dir, "build", "my-novel", chapterFiles)

	args := cmd.Args
	// Should use the Shunn script
	assert.Contains(t, args[1], "md2long.sh")
	assert.Contains(t, args, "-x")
	assert.Contains(t, args, "-r")
}
