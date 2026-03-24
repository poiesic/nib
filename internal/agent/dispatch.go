package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

const defaultAgent = "claude"

// agentName resolves the active agent from git config, env var override, or default.
func agentName() string {
	if v := os.Getenv("NIB_AGENT"); v != "" {
		return v
	}
	cmd := exec.Command("git", "config", "--get", "nib.agent")
	out, err := cmd.Output()
	if err == nil {
		name := string(bytes.TrimSpace(out))
		if name != "" {
			return name
		}
	}
	return defaultAgent
}

// binaryName returns the expected executable name for a given agent.
func binaryName(agent string) string {
	return "nib-agent-" + agent
}

// Complete sends a prompt to the agent and returns a text response.
func Complete(prompt string, effort string, tools []string, dir string) (string, error) {
	req := Request{
		Operation: OpComplete,
		Prompt:    prompt,
		Effort:    effort,
		Tools:     tools,
		Dir:       dir,
	}
	stdout, err := dispatch(req)
	if err != nil {
		return "", err
	}
	if err := validateResponse(stdout, OpComplete); err != nil {
		return "", err
	}
	var resp CompleteResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return "", fmt.Errorf("parsing agent response: %w", err)
	}
	return resp.Text, nil
}

// Extract sends a prompt with a JSON schema and returns structured data.
func Extract(prompt string, schema json.RawMessage, effort string, tools []string, dir string) (json.RawMessage, error) {
	req := Request{
		Operation: OpExtract,
		Prompt:    prompt,
		Effort:    effort,
		Tools:     tools,
		Schema:    schema,
		Dir:       dir,
	}
	stdout, err := dispatch(req)
	if err != nil {
		return nil, err
	}
	if err := validateResponse(stdout, OpExtract); err != nil {
		return nil, err
	}
	var resp ExtractResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("parsing agent response: %w", err)
	}
	return resp.Data, nil
}

// ConverseOptions configures a converse operation.
type ConverseOptions struct {
	Effort      string
	Session     *SessionOptions
	Permissions string   // e.g. "acceptEdits", "default"
	Tools       []string // allowed tools; nil = unrestricted
}

// Converse launches an interactive session. The agent reads the request
// from stdin, then takes over the TTY for interactive use.
func Converse(prompt string, opts ConverseOptions, dir string) error {
	req := Request{
		Operation:   OpConverse,
		Prompt:      prompt,
		Effort:      opts.Effort,
		Tools:       opts.Tools,
		Session:     opts.Session,
		Dir:         dir,
		Permissions: opts.Permissions,
	}
	return dispatchInteractive(req)
}

// Scaffold asks the agent backend to write its project scaffolding files.
// Returns the list of relative file paths created.
func Scaffold(projectDir string, projectName string) ([]string, error) {
	req := Request{
		Operation:   OpScaffold,
		Dir:         projectDir,
		ProjectName: projectName,
	}
	stdout, err := dispatch(req)
	if err != nil {
		return nil, err
	}
	if err := validateResponse(stdout, OpScaffold); err != nil {
		return nil, err
	}
	var resp ScaffoldResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("parsing agent response: %w", err)
	}
	return resp.Files, nil
}

// validateResponse checks the response envelope for errors and type/operation consistency.
func validateResponse(data []byte, expectedOp Operation) error {
	var env ResponseEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("invalid agent response: %w", err)
	}
	if env.Type == "" {
		return fmt.Errorf("invalid agent response: missing \"type\" field")
	}
	if env.Type == RespError {
		return fmt.Errorf("agent error: %s", env.Error)
	}
	if env.Type != RespSuccess {
		return fmt.Errorf("invalid agent response: unknown type %q", env.Type)
	}
	if env.Operation != expectedOp {
		return fmt.Errorf("invalid agent response: expected operation %q, got %q", expectedOp, env.Operation)
	}
	return nil
}

// dispatch executes the agent binary in pipe mode and returns stdout.
func dispatch(req Request) ([]byte, error) {
	name := agentName()
	bin := binaryName(name)

	binPath, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("agent %q not found: %s not on PATH", name, bin)
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	cmd := exec.Command(binPath)
	cmd.Dir = req.Dir
	cmd.Stdin = bytes.NewReader(reqJSON)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		if stderrBuf.Len() > 0 {
			return nil, fmt.Errorf("agent %q failed: %s", name, stderrBuf.String())
		}
		return nil, fmt.Errorf("agent %q failed: %w", name, err)
	}

	return stdoutBuf.Bytes(), nil
}

// dispatchInteractive executes the agent binary with the request passed via
// a temporary file, allowing the terminal stdin to flow through to the child.
func dispatchInteractive(req Request) error {
	name := agentName()
	bin := binaryName(name)

	binPath, err := exec.LookPath(bin)
	if err != nil {
		return fmt.Errorf("agent %q not found: %s not on PATH", name, bin)
	}

	reqFile, err := WriteRequestFile(req)
	if err != nil {
		return err
	}
	defer os.Remove(reqFile) // clean up if agent didn't

	cmd := exec.Command(binPath)
	cmd.Dir = req.Dir
	cmd.Env = append(os.Environ(), RequestFileEnv+"="+reqFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
