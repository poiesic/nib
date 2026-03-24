package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/poiesic/nib/internal/agent"
)

func main() {
	req, err := readRequest()
	if err != nil {
		fatal("%v", err)
	}

	switch req.Operation {
	case agent.OpComplete:
		if err := complete(req); err != nil {
			fatal("%v", err)
		}
	case agent.OpExtract:
		if err := extract(req); err != nil {
			fatal("%v", err)
		}
	case agent.OpConverse:
		if err := converse(req); err != nil {
			fatal("%v", err)
		}
	case agent.OpScaffold:
		if err := scaffold(req); err != nil {
			fatal("%v", err)
		}
	default:
		fatal("unknown operation: %s", req.Operation)
	}
}

func complete(req agent.Request) error {
	args := []string{"-p", req.Prompt, "--no-session-persistence"}
	args = appendCommon(args, req)

	cmd := exec.Command("claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("claude: %s", strings.TrimSpace(stderr.String()))
		}
		if stdout.Len() > 0 {
			return fmt.Errorf("claude: %s", strings.TrimSpace(stdout.String()))
		}
		return fmt.Errorf("claude: %w", err)
	}

	resp := agent.CompleteResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpComplete,
		Text:      stdout.String(),
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func extract(req agent.Request) error {
	args := []string{
		"-p", req.Prompt,
		"--output-format", "json",
		"--json-schema", string(req.Schema),
		"--no-session-persistence",
	}
	args = appendCommon(args, req)

	cmd := exec.Command("claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("claude: %s", strings.TrimSpace(stderr.String()))
		}
		if stdout.Len() > 0 {
			return fmt.Errorf("claude: %s", strings.TrimSpace(stdout.String()))
		}
		return fmt.Errorf("claude: %w", err)
	}

	// Claude CLI wraps structured output in an envelope
	var envelope struct {
		StructuredOutput json.RawMessage `json:"structured_output"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		return fmt.Errorf("parsing claude response: %w", err)
	}
	if envelope.StructuredOutput == nil {
		return fmt.Errorf("claude returned no structured_output")
	}

	resp := agent.ExtractResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpExtract,
		Data:      envelope.StructuredOutput,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func converse(req agent.Request) error {
	// Handle --new: delete existing session file
	if req.Session != nil && req.Session.New && req.Session.ID != "" {
		deleteSessionFile(req.Session.ID)
	}

	var args []string
	if req.Session != nil && req.Session.Resume {
		args = []string{"--resume", req.Session.ID}
	} else {
		args = []string{req.Prompt}
		if req.Session != nil && req.Session.ID != "" {
			args = append(args, "--session-id", req.Session.ID)
		}
	}
	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}
	if req.Permissions != "" {
		args = append(args, "--permission-mode", req.Permissions)
	}
	if len(req.Tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(req.Tools, ","))
	}

	cmd := exec.Command("claude", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	// Auto-resume if session already exists
	if err != nil && req.Session != nil && !req.Session.Resume && req.Session.ID != "" {
		resumeCmd := exec.Command("claude", "--resume", req.Session.ID)
		resumeCmd.Stdin = os.Stdin
		resumeCmd.Stdout = os.Stdout
		resumeCmd.Stderr = os.Stderr
		if resumeErr := resumeCmd.Run(); resumeErr == nil {
			return nil
		}
	}

	return err
}

// deleteSessionFile removes a Claude session file by ID.
func deleteSessionFile(sessionID string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	projectKey := strings.ReplaceAll(cwd, string(os.PathSeparator), "-")
	path := fmt.Sprintf("%s/.claude/projects/%s/%s.jsonl", homeDir, projectKey, sessionID)
	os.Remove(path)
}

func appendCommon(args []string, req agent.Request) []string {
	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}
	if len(req.Tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(req.Tools, ","))
	}
	return args
}

func readRequest() (agent.Request, error) {
	if path := os.Getenv(agent.RequestFileEnv); path != "" {
		return agent.ReadRequestFile(path)
	}
	var req agent.Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return agent.Request{}, fmt.Errorf("reading request: %w", err)
	}
	return req, nil
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
