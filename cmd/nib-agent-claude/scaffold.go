package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/poiesic/nib/cmd/nib-agent-claude/templates"
	"github.com/poiesic/nib/internal/agent"
)

type scaffoldData struct {
	ProjectName string
}

func scaffold(req agent.Request) error {
	projectDir := req.Dir
	data := scaffoldData{ProjectName: req.ProjectName}

	var created []string

	// Process templated files
	templatedFiles := map[string]string{
		"claude.md.tmpl": "CLAUDE.md",
		"tools.md.tmpl":  "TOOLS.md",
	}
	for tmplName, destName := range templatedFiles {
		destPath := filepath.Join(projectDir, destName)
		if err := writeTemplate(projectDir, tmplName, destPath, data); err != nil {
			return err
		}
		created = append(created, destName)
	}

	// Copy skill files
	skillFiles, err := copySkills(projectDir)
	if err != nil {
		return err
	}
	created = append(created, skillFiles...)

	resp := agent.ScaffoldResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpScaffold,
		Files:     created,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func writeTemplate(projectDir, tmplName, destPath string, data scaffoldData) error {
	content, err := templates.FS.ReadFile(tmplName)
	if err != nil {
		return err
	}
	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

func copySkills(projectDir string) ([]string, error) {
	var created []string
	err := fs.WalkDir(templates.FS, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destPath := filepath.Join(projectDir, ".claude", path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		content, err := templates.FS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return err
		}
		relPath, _ := filepath.Rel(projectDir, destPath)
		created = append(created, relPath)
		return nil
	})
	return created, err
}
