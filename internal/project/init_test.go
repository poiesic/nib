package project

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noOpRunner returns a command that always succeeds without doing anything.
func noOpRunner(name string, args ...string) *exec.Cmd {
	return exec.Command("true")
}

var noOpScaffold = func(projectDir, projectName string) ([]string, error) {
	return nil, nil
}

func noConflictOpts() InitOptions {
	return InitOptions{
		Runner:        noOpRunner,
		Stdin:         strings.NewReader(""),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
}

func TestInit_CreatesDirectoryStructure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	_, err := Init("test-novel", noConflictOpts())
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "test-novel")

	// Verify directories exist
	expectedDirs := []string{
		"scenes",
		"characters",
		"storydb",
		"appendices",
		"assets",
		"build",
	}
	for _, d := range expectedDirs {
		info, err := os.Stat(filepath.Join(projectDir, d))
		require.NoError(t, err, "directory %s should exist", d)
		assert.True(t, info.IsDir(), "%s should be a directory", d)
	}
}

func TestInit_CreatesTemplatedFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	_, err := Init("my-novel", noConflictOpts())
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "my-novel")

	// Verify templated files contain the project name
	bookYAML, err := os.ReadFile(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(bookYAML), "my-novel")

	// Verify STYLE.md exists and uses default (first-person) variant
	styleContent, err := os.ReadFile(filepath.Join(projectDir, "STYLE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(styleContent), "First-person-close or tight third person")

	// Verify TROPES.md exists and contains tropes content
	tropesContent, err := os.ReadFile(filepath.Join(projectDir, "TROPES.md"))
	require.NoError(t, err)
	assert.Contains(t, string(tropesContent), "AI Writing Tropes to Avoid")

	// Verify .gitignore exists
	gitignore, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(gitignore), "build/")
}

func TestInit_CreatesStorydbFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	_, err := Init("storydb-test", noConflictOpts())
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "storydb-test")
	expectedFiles := []string{
		"scenes.csv",
		"facts.csv",
		"scene_characters.csv",
		"locations.csv",
		"timeline.csv",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(projectDir, "storydb", f)
		info, err := os.Stat(path)
		require.NoError(t, err, "%s should exist", f)
		assert.False(t, info.IsDir())
	}
}

func TestInit_SucceedsOnExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	// Create the directory first
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "existing"), 0755))

	// R = replace all, so no further prompts
	opts := InitOptions{
		Runner:        noOpRunner,
		Stdin:         strings.NewReader("R"),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("existing", opts)
	require.NoError(t, err)

	// Verify files were created
	projectDir := filepath.Join(dir, "existing")
	_, err = os.Stat(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
}

func TestInit_SkipsFileOnSkip(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	projectDir := filepath.Join(dir, "skip-test")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Pre-create book.yaml with known content
	originalContent := "original content"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "book.yaml"), []byte(originalContent), 0644))

	// s = skip this file, then R = replace all for subsequent prompts
	opts := InitOptions{
		Runner:        noOpRunner,
		Stdin:         strings.NewReader("sR"),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("skip-test", opts)
	require.NoError(t, err)

	// Verify the pre-existing file was NOT overwritten
	content, err := os.ReadFile(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(content))

	// Verify other files were still created
	_, err = os.Stat(filepath.Join(projectDir, "TROPES.md"))
	require.NoError(t, err)
}

func TestInit_SkipAllSkipsAllFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	projectDir := filepath.Join(dir, "skipall-test")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Pre-create multiple files with known content
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "book.yaml"), []byte("old-book"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte("old-gitignore"), 0644))

	// S on first prompt should skip all without further prompts
	opts := InitOptions{
		Runner:        noOpRunner,
		Stdin:         strings.NewReader("S"),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("skipall-test", opts)
	require.NoError(t, err)

	// Verify all pre-existing files were NOT overwritten
	bookYAML, err := os.ReadFile(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "old-book", string(bookYAML))

	gitignore, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, "old-gitignore", string(gitignore))
}

func TestInit_AbortsOnAbort(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	projectDir := filepath.Join(dir, "abort-test")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Pre-create a file so we get prompted
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "book.yaml"), []byte("original"), 0644))

	// a = abort
	opts := InitOptions{
		Runner:        noOpRunner,
		Stdin:         strings.NewReader("a"),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("abort-test", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aborted by user")
}

func TestInit_ReplaceAllSkipsSubsequentPrompts(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	projectDir := filepath.Join(dir, "replaceall-test")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Pre-create multiple files
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "book.yaml"), []byte("old-book"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte("old-gitignore"), 0644))

	// R on first prompt should replace all without further prompts
	opts := InitOptions{
		Runner:        noOpRunner,
		Stdin:         strings.NewReader("R"),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("replaceall-test", opts)
	require.NoError(t, err)

	// Verify all files were replaced (contain the project name or template content)
	bookYAML, err := os.ReadFile(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(bookYAML), "replaceall-test")
	assert.NotEqual(t, "old-book", string(bookYAML))

	gitignore, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	require.NoError(t, err)
	assert.NotEqual(t, "old-gitignore", string(gitignore))
}

func TestInit_PromptShowsRelativePath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	projectDir := filepath.Join(dir, "prompt-test")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Pre-create a file to trigger a prompt
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "book.yaml"), []byte("old"), 0644))

	var output strings.Builder
	opts := InitOptions{
		Runner:        noOpRunner,
		Stdin:         strings.NewReader("rR"),
		Stdout:        &output,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("prompt-test", opts)
	require.NoError(t, err)

	assert.Contains(t, output.String(), "book.yaml")
}

func TestInit_ErrorOnInvalidName(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	tests := []struct {
		name string
	}{
		{"Invalid-Name"},    // uppercase
		{"has spaces"},      // spaces
		{"has_underscores"}, // underscores
		{""},                // empty
		{"-leading-dash"},   // leading dash
		{"trailing-dash-"},  // trailing dash
		{"double--dash"},    // double dash
	}
	for _, tt := range tests {
		_, err := Init(tt.name, noConflictOpts())
		assert.Error(t, err, "name %q should be invalid", tt.name)
		if err != nil {
			assert.Contains(t, err.Error(), "invalid project name")
		}
	}
}

func TestInit_ValidNames(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	validNames := []string{
		"my-novel",
		"novel",
		"the-great-story",
		"book1",
		"a",
	}
	for _, name := range validNames {
		_, err := Init(name, noConflictOpts())
		assert.NoError(t, err, "name %q should be valid", name)
	}
}

func TestInit_DotInitializesCurrentDir(t *testing.T) {
	dir := t.TempDir()
	// Create a directory with a valid project name and chdir into it
	projectDir := filepath.Join(dir, "my-novel")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	require.NoError(t, os.Chdir(projectDir))

	resolved, err := Init(".", noConflictOpts())
	require.NoError(t, err)
	assert.Equal(t, "my-novel", resolved)

	// Files should be in the current directory, not in a subdirectory
	_, err = os.Stat(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
}

func TestInit_DotDotInitializesParentDir(t *testing.T) {
	dir := t.TempDir()
	// Create parent with valid name and a child to chdir into
	parentDir := filepath.Join(dir, "my-novel")
	childDir := filepath.Join(parentDir, "subdir")
	require.NoError(t, os.MkdirAll(childDir, 0755))
	require.NoError(t, os.Chdir(childDir))

	resolved, err := Init("..", noConflictOpts())
	require.NoError(t, err)
	assert.Equal(t, "my-novel", resolved)

	// Files should be in the parent directory
	_, err = os.Stat(filepath.Join(parentDir, "book.yaml"))
	require.NoError(t, err)
}

func TestInit_DotRejectsInvalidDirName(t *testing.T) {
	dir := t.TempDir()
	// Create a directory with an invalid project name
	badDir := filepath.Join(dir, "Bad_Name")
	require.NoError(t, os.MkdirAll(badDir, 0755))
	require.NoError(t, os.Chdir(badDir))

	_, err := Init(".", noConflictOpts())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid project name")
}

func TestInit_NoGitSkipsGitCommands(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	var commands [][]string
	captureRunner := func(name string, args ...string) *exec.Cmd {
		commands = append(commands, append([]string{name}, args...))
		return exec.Command("true")
	}

	opts := InitOptions{
		Runner:        captureRunner,
		Stdin:         strings.NewReader(""),
		Stdout:        io.Discard,
		NoGit:         true,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("nogit-test", opts)
	require.NoError(t, err)

	assert.Empty(t, commands, "should not run any git commands when NoGit is true")

	// Verify project was still scaffolded
	projectDir := filepath.Join(dir, "nogit-test")
	_, err = os.Stat(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
}

func TestInit_RunsGitInitAndSubmodule(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	var commands [][]string
	captureRunner := func(name string, args ...string) *exec.Cmd {
		commands = append(commands, append([]string{name}, args...))
		return exec.Command("true")
	}

	opts := InitOptions{
		Runner:        captureRunner,
		Stdin:         strings.NewReader(""),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("git-test", opts)
	require.NoError(t, err)

	require.Len(t, commands, 3, "should run git init, git submodule add, and git config")

	// First command: git init <dir>
	assert.Equal(t, "git", commands[0][0])
	assert.Equal(t, "init", commands[0][1])
	assert.Contains(t, commands[0][2], "git-test")

	// Second command: git -C <dir> submodule add <repo> pandoc-templates
	assert.Equal(t, "git", commands[1][0])
	assert.Equal(t, "-C", commands[1][1])
	assert.Contains(t, commands[1][2], "git-test")
	assert.Equal(t, "submodule", commands[1][3])
	assert.Equal(t, "add", commands[1][4])
	assert.Contains(t, commands[1][5], "pandoc-templates")
	assert.Equal(t, "pandoc-templates", commands[1][6])

	// Third command: git -C <dir> config nib.agent claude
	assert.Equal(t, "git", commands[2][0])
	assert.Equal(t, "-C", commands[2][1])
	assert.Equal(t, "config", commands[2][3])
	assert.Equal(t, "nib.agent", commands[2][4])
	assert.Equal(t, "claude", commands[2][5])
}

func TestInit_SkipsGitInitIfGitExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	// Pre-create .git dir and pandoc-templates
	projectDir := filepath.Join(dir, "git-exists")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".git"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "pandoc-templates"), 0755))

	var commands [][]string
	captureRunner := func(name string, args ...string) *exec.Cmd {
		commands = append(commands, append([]string{name}, args...))
		return exec.Command("true")
	}

	opts := InitOptions{
		Runner:        captureRunner,
		Stdin:         strings.NewReader("R"),
		Stdout:        io.Discard,
		AgentScaffold: noOpScaffold,
	}
	_, err := Init("git-exists", opts)
	require.NoError(t, err)

	// Should skip git init and submodule add, but still run git config for agent
	require.Len(t, commands, 1)
	assert.Equal(t, "config", commands[0][3])
	assert.Equal(t, "nib.agent", commands[0][4])
}

func TestInit_StyleFirstPerson(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	opts := noConflictOpts()
	opts.Style = "first-person"
	_, err := Init("fp-test", opts)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "fp-test", "STYLE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "First-person-close or tight third person")
}

func TestInit_StyleThirdClose(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	opts := noConflictOpts()
	opts.Style = "third-close"
	_, err := Init("tc-test", opts)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "tc-test", "STYLE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "Tight third person throughout")
}

func TestInit_StyleThirdOmniscient(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	opts := noConflictOpts()
	opts.Style = "third-omniscient"
	_, err := Init("to-test", opts)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "to-test", "STYLE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "Third-person omniscient")
}

func TestInit_StyleDefaultsToFirstPerson(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	opts := noConflictOpts()
	// Style left empty — should default to first-person
	_, err := Init("default-test", opts)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "default-test", "STYLE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "First-person-close or tight third person")
}

func TestInit_NoStyleSkipsStyleMd(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	opts := noConflictOpts()
	opts.NoStyle = true
	_, err := Init("nostyle-test", opts)
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "nostyle-test")

	// STYLE.md should not exist
	_, err = os.Stat(filepath.Join(projectDir, "STYLE.md"))
	assert.True(t, os.IsNotExist(err), "STYLE.md should not be created with --no-style")

	// Other files should still exist
	_, err = os.Stat(filepath.Join(projectDir, "book.yaml"))
	require.NoError(t, err)
}

func TestInit_StyleInvalid(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	opts := noConflictOpts()
	opts.Style = "stream-of-consciousness"
	_, err := Init("bad-style", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid style")
	assert.Contains(t, err.Error(), "stream-of-consciousness")
}
