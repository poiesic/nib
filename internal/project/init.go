package project

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"golang.org/x/term"

	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/project/templates"
	"github.com/poiesic/nib/internal/storydb"
)

const pandocTemplatesRepo = "https://github.com/prosegrinder/pandoc-templates.git"

// CommandRunner creates an exec.Cmd. Injected for testing.
type CommandRunner func(name string, args ...string) *exec.Cmd

var validProjectName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var (
	errSkipFile  = errors.New("skip file")
	errAbortInit = errors.New("init aborted by user")
)

type templateData struct {
	ProjectName string
}

// ScaffoldFunc is the function signature for agent scaffolding. Override in tests.
type ScaffoldFunc func(projectDir, projectName string) ([]string, error)

// InitOptions configures the Init function.
type InitOptions struct {
	Runner        CommandRunner // nil = exec.Command
	Stdin         io.Reader     // nil = os.Stdin
	Stdout        io.Writer     // nil = os.Stdout
	NoGit         bool          // skip git init and submodule add
	Style         string        // STYLE.md variant: first-person, third-close, third-omniscient (default: first-person)
	NoStyle       bool          // skip STYLE.md creation
	Agent         string        // agent name for scaffolding (default: claude)
	AgentScaffold ScaffoldFunc  // nil = agent.Scaffold
}

// Init creates a new scrib project directory with the full scaffold.
// Returns the resolved project name (relevant when "." or ".." is passed).
func Init(projectName string, opts ...InitOptions) (string, error) {
	var opt InitOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	run := opt.Runner
	if run == nil {
		run = exec.Command
	}
	stdin := opt.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := opt.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	// Validate and default the style
	style := opt.Style
	if !opt.NoStyle {
		if style == "" {
			style = "first-person"
		}
		if !slices.Contains(templates.ValidStyles, style) {
			return "", fmt.Errorf("invalid style %q: must be one of %s", style, strings.Join(templates.ValidStyles, ", "))
		}
	}

	var projectDir string
	if projectName == "." || projectName == ".." {
		absDir, err := filepath.Abs(projectName)
		if err != nil {
			return "", fmt.Errorf("resolving %s: %w", projectName, err)
		}
		projectName = filepath.Base(absDir)
		projectDir = absDir
	} else {
		projectDir = filepath.Join(".", projectName)
	}

	if !validProjectName.MatchString(projectName) {
		return "", fmt.Errorf("invalid project name %q: must be lowercase alphanumeric with hyphens", projectName)
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("creating project directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{
		"scenes",
		"characters",
		"storydb",
		"appendices",
		"assets",
		"build",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectDir, dir), 0755); err != nil {
			return "", fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	data := templateData{ProjectName: projectName}
	var replaceAll, skipAll bool

	// Process templated files (universal only — agent-specific files come from scaffold)
	templatedFiles := map[string]string{
		"tropes.md.tmpl": "TROPES.md",
		"book.yaml.tmpl": "book.yaml",
		"gitignore.tmpl": ".gitignore",
	}
	if !opt.NoStyle {
		templatedFiles[fmt.Sprintf("style-%s.md.tmpl", style)] = "STYLE.md"
	}
	for tmplName, destName := range templatedFiles {
		destPath := filepath.Join(projectDir, destName)
		err := promptOnConflict(projectDir, destPath, stdin, stdout, &replaceAll, &skipAll)
		if errors.Is(err, errSkipFile) {
			continue
		}
		if err != nil {
			return "", err
		}
		if err := writeTemplate(projectDir, tmplName, destName, data); err != nil {
			return "", fmt.Errorf("writing %s: %w", destName, err)
		}
	}

	// Create storydb CSV files
	db, err := storydb.Open(projectDir)
	if err != nil {
		return "", fmt.Errorf("creating storydb: %w", err)
	}
	db.Close()

	// Ask the agent backend to write its scaffolding files
	scaffoldFn := opt.AgentScaffold
	if scaffoldFn == nil {
		scaffoldFn = agent.Scaffold
	}
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolving project dir: %w", err)
	}
	scaffoldedFiles, err := scaffoldFn(absProjectDir, projectName)
	if err != nil {
		return "", fmt.Errorf("agent scaffolding: %w", err)
	}
	for _, f := range scaffoldedFiles {
		fmt.Fprintf(stdout, "  %s (agent)\n", f)
	}

	// Initialize git repo and add pandoc-templates submodule
	if !opt.NoGit {
		if err := initGitRepo(projectDir, run); err != nil {
			return "", fmt.Errorf("initializing git: %w", err)
		}
		// Persist agent choice in git config
		agentName := opt.Agent
		if agentName == "" {
			agentName = "claude"
		}
		setAgent := run("git", "-C", absProjectDir, "config", "nib.agent", agentName)
		setAgent.Stdout = os.Stdout
		setAgent.Stderr = os.Stderr
		if err := setAgent.Run(); err != nil {
			fmt.Fprintf(stdout, "Warning: could not set nib.agent in git config: %v\n", err)
		}
	}

	return projectName, nil
}

// readKey reads a single keypress from stdin. When stdin is a terminal,
// raw mode is used so no Enter key is required.
func readKey(stdin io.Reader, stdout io.Writer) (byte, error) {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		oldState, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			return 0, err
		}
		defer term.Restore(int(f.Fd()), oldState)
	}
	buf := make([]byte, 1)
	_, err := stdin.Read(buf)
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(stdout, "%c\r\n", buf[0])
	return buf[0], nil
}

func promptOnConflict(projectDir, path string, stdin io.Reader, stdout io.Writer, replaceAll, skipAll *bool) error {
	if *replaceAll {
		return nil
	}
	if *skipAll {
		return errSkipFile
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	relPath, _ := filepath.Rel(projectDir, path)
	if relPath == "" {
		relPath = path
	}
	fmt.Fprintf(stdout, "File %q already exists. (r)eplace (R)eplace all (s)kip (S)kip all (a)bort: ", relPath)
	key, err := readKey(stdin, stdout)
	if err != nil {
		if err == io.EOF {
			return errAbortInit
		}
		return err
	}
	switch key {
	case 'r':
		return nil
	case 'R':
		*replaceAll = true
		return nil
	case 's':
		return errSkipFile
	case 'S':
		*skipAll = true
		return errSkipFile
	case 'a':
		return errAbortInit
	default:
		return errAbortInit
	}
}

func initGitRepo(projectDir string, run CommandRunner) error {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	// Skip git init if .git/ already exists
	if _, err := os.Stat(filepath.Join(absDir, ".git")); errors.Is(err, os.ErrNotExist) {
		gitInit := run("git", "init", absDir)
		gitInit.Stdout = os.Stdout
		gitInit.Stderr = os.Stderr
		if err := gitInit.Run(); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	}

	// Skip submodule add if pandoc-templates/ already exists
	if _, err := os.Stat(filepath.Join(absDir, "pandoc-templates")); errors.Is(err, os.ErrNotExist) {
		submoduleAdd := run("git", "-C", absDir, "submodule", "add", pandocTemplatesRepo, "pandoc-templates")
		submoduleAdd.Stdout = os.Stdout
		submoduleAdd.Stderr = os.Stderr
		if err := submoduleAdd.Run(); err != nil {
			return fmt.Errorf("adding pandoc-templates submodule: %w", err)
		}
	}

	return nil
}

func writeTemplate(projectDir, tmplName, destName string, data templateData) error {
	content, err := templates.FS.ReadFile(tmplName)
	if err != nil {
		return err
	}
	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return err
	}
	destPath := filepath.Join(projectDir, destName)
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}
