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

// CharacterTalkOptions configures a character-talk operation.
type CharacterTalkOptions struct {
	Session *SessionOptions
	Context string
}

// SceneProof runs mechanical proofreading on the specified scene files.
func SceneProof(paths []string, dir string) (string, error) {
	return completeOp(Request{
		Operation: OpSceneProof,
		Paths:     paths,
		Dir:       dir,
	})
}

// ChapterProof runs mechanical proofreading on the specified chapter files.
func ChapterProof(paths []string, dir string) (string, error) {
	return completeOp(Request{
		Operation: OpChapterProof,
		Paths:     paths,
		Dir:       dir,
	})
}

// VoiceCheck checks character voice consistency across sampled scenes.
func VoiceCheck(slug string, paths []string, dir string) (string, error) {
	return completeOp(Request{
		Operation:     OpVoiceCheck,
		CharacterSlug: slug,
		Paths:         paths,
		Dir:           dir,
	})
}

// ContinuityCheck runs continuity error detection on the specified scenes.
func ContinuityCheck(paths []string, dir string) (string, error) {
	return completeOp(Request{
		Operation: OpContinuityCheck,
		Paths:     paths,
		Dir:       dir,
	})
}

// ContinuityIndex extracts structured continuity data from a scene.
func ContinuityIndex(prompt string, schema json.RawMessage, dir string) (json.RawMessage, error) {
	req := Request{
		Operation: OpContinuityIndex,
		Context:   prompt,
		Schema:    schema,
		Dir:       dir,
	}
	stdout, err := dispatch(req)
	if err != nil {
		return nil, err
	}
	if err := validateResponse(stdout, OpContinuityIndex); err != nil {
		return nil, err
	}
	var resp IndexResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("parsing agent response: %w", err)
	}
	return resp.Data, nil
}

// ContinuityAsk sends a research question about the manuscript.
func ContinuityAsk(question, rangeExpr, dir string) (string, error) {
	return completeOp(Request{
		Operation: OpContinuityAsk,
		Question:  question,
		Range:     rangeExpr,
		Dir:       dir,
	})
}

// SceneCritique launches an interactive editorial review of a scene.
func SceneCritique(paths []string, dir string) error {
	return interactiveOp(Request{
		Operation: OpSceneCritique,
		Paths:     paths,
		Dir:       dir,
	})
}

// ChapterCritique launches an interactive editorial review of a chapter.
func ChapterCritique(paths []string, dir string) error {
	return interactiveOp(Request{
		Operation: OpChapterCritique,
		Paths:     paths,
		Dir:       dir,
	})
}

// CharacterTalk launches an interactive in-character interview session.
func CharacterTalk(opts CharacterTalkOptions, dir string) error {
	return interactiveOp(Request{
		Operation: OpCharacterTalk,
		Context:   opts.Context,
		Session:   opts.Session,
		Dir:       dir,
	})
}

// ProjectScaffold asks the agent backend to write its project scaffolding files.
func ProjectScaffold(projectDir string, projectName string) ([]string, error) {
	req := Request{
		Operation:   OpProjectScaffold,
		Dir:         projectDir,
		ProjectName: projectName,
	}
	stdout, err := dispatch(req)
	if err != nil {
		return nil, err
	}
	if err := validateResponse(stdout, OpProjectScaffold); err != nil {
		return nil, err
	}
	var resp ScaffoldResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("parsing agent response: %w", err)
	}
	return resp.Files, nil
}

// completeOp dispatches a pipe-mode operation and returns the text response.
func completeOp(req Request) (string, error) {
	stdout, err := dispatch(req)
	if err != nil {
		return "", err
	}
	if err := validateResponse(stdout, req.Operation); err != nil {
		return "", err
	}
	var resp CompleteResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return "", fmt.Errorf("parsing agent response: %w", err)
	}
	return resp.Text, nil
}

// interactiveOp dispatches an interactive operation that takes over the TTY.
func interactiveOp(req Request) error {
	return dispatchInteractive(req)
}

// validateResponse checks the response envelope for errors and operation consistency.
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
