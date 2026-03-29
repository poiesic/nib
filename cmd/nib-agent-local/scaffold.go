package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"text/template"

	"github.com/poiesic/nib/cmd/nib-agent-local/templates"
	"github.com/poiesic/nib/internal/agent"
)

type scaffoldData struct {
	ProjectName string
}

func scaffold(req agent.Request) error {
	projectDir := req.Dir
	data := scaffoldData{ProjectName: req.ProjectName}

	var created []string

	// Write AGENTS.md
	destPath := filepath.Join(projectDir, "AGENTS.md")
	if err := writeTemplate(destPath, "agents.md.tmpl", data); err != nil {
		return err
	}
	created = append(created, "AGENTS.md")

	resp := agent.ScaffoldResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpProjectScaffold,
		Files:     created,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func writeTemplate(destPath, tmplName string, data scaffoldData) error {
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
