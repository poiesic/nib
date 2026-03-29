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
	case agent.OpSceneProof:
		err = proof(req)
	case agent.OpChapterProof:
		err = proof(req)
	case agent.OpSceneCritique:
		err = critique(req, "review-scene")
	case agent.OpChapterCritique:
		err = critique(req, "review-chapter")
	case agent.OpVoiceCheck:
		err = voiceCheck(req)
	case agent.OpContinuityCheck:
		err = continuityCheck(req)
	case agent.OpContinuityAsk:
		err = continuityAsk(req)
	case agent.OpContinuityIndex:
		err = continuityIndex(req)
	case agent.OpCharacterTalk:
		err = characterTalk(req)
	case agent.OpProjectScaffold:
		err = scaffold(req)
	default:
		fatal("unknown operation: %s", req.Operation)
	}
	if err != nil {
		fatal("%v", err)
	}
}

func proof(req agent.Request) error {
	prompt := fmt.Sprintf("/copy-edit %s", strings.Join(req.Paths, " "))
	return runPipe(prompt, "medium", []string{"Read", "Edit"}, req.Operation)
}

func critique(req agent.Request, skill string) error {
	prompt := fmt.Sprintf("/%s %s", skill, strings.Join(req.Paths, " "))
	return runInteractive(prompt, "high", nil)
}

func voiceCheck(req agent.Request) error {
	prompt := fmt.Sprintf("/voice-check %s %s", req.CharacterSlug, strings.Join(req.Paths, " "))
	return runPipe(prompt, "high", []string{"Read"}, req.Operation)
}

func continuityCheck(req agent.Request) error {
	prompt := fmt.Sprintf("/continuity-check %s", strings.Join(req.Paths, " "))
	return runPipe(prompt, "high", []string{"Read", "Bash"}, req.Operation)
}

func continuityAsk(req agent.Request) error {
	prompt := buildAskPrompt(req.Question, req.Range)
	return runPipe(prompt, "high", []string{"Read", "Bash"}, req.Operation)
}

func continuityIndex(req agent.Request) error {
	args := []string{
		"-p", req.Context,
		"--output-format", "json",
		"--json-schema", string(req.Schema),
		"--no-session-persistence",
		"--effort", "medium",
		"--allowedTools", "Read",
	}

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

	resp := agent.IndexResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpContinuityIndex,
		Data:      envelope.StructuredOutput,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func characterTalk(req agent.Request) error {
	if req.Session != nil && req.Session.New && req.Session.ID != "" {
		deleteSessionFile(req.Session.ID)
	}

	if req.Session != nil && req.Session.Resume {
		return runInteractive("", "", req.Session)
	}
	return runInteractive(req.Context, "", req.Session)
}

// runPipe invokes claude in non-interactive pipe mode and writes a CompleteResponse to stdout.
func runPipe(prompt, effort string, tools []string, op agent.Operation) error {
	args := []string{"-p", prompt, "--no-session-persistence"}
	if effort != "" {
		args = append(args, "--effort", effort)
	}
	if len(tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(tools, ","))
	}

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
		Operation: op,
		Text:      stdout.String(),
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

// runInteractive invokes claude with TTY passthrough for interactive sessions.
func runInteractive(prompt, effort string, session *agent.SessionOptions) error {
	var args []string
	if session != nil && session.Resume {
		args = []string{"--resume", session.ID}
	} else {
		args = []string{prompt}
		if session != nil && session.ID != "" {
			args = append(args, "--session-id", session.ID)
		}
	}
	if effort != "" {
		args = append(args, "--effort", effort)
	}

	cmd := exec.Command("claude", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	// Auto-resume if session already exists
	if err != nil && session != nil && !session.Resume && session.ID != "" {
		resumeCmd := exec.Command("claude", "--resume", session.ID)
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

func buildAskPrompt(question, rangeExpr string) string {
	var b strings.Builder

	b.WriteString("You are a research assistant for a novel manuscript managed by `nib`.\n\n")

	b.WriteString("## How to find information\n\n")
	b.WriteString("You have access to these tools for locating information in the manuscript:\n\n")

	b.WriteString("### Nib commands (run via Bash)\n")
	b.WriteString("- `nib ct recap <range>` -- JSON summaries of scenes in a chapter range (e.g. `1-5`, `3`, `1,3,5`)\n")
	b.WriteString("- `nib ct recap <range> --detailed` -- includes facts, locations, dates, times, and all character appearances\n")
	b.WriteString("- `nib ct recap <range> -c <character-slug>` -- filter recap to scenes involving a character (repeatable)\n")
	b.WriteString("- `nib ct characters` -- list all known character slugs from indexed data\n")
	b.WriteString("- `nib ct characters <range>` -- characters in a specific chapter range\n")
	b.WriteString("- `nib ct chapters <character> [character...]` -- find scenes where characters appear together (AND)\n")
	b.WriteString("- `nib ct chapters --or <character> [character...]` -- find scenes per character (OR)\n")
	b.WriteString("- `nib ma status` -- manuscript statistics (chapters, scenes, word count)\n\n")

	b.WriteString("### File access (via Read)\n")
	b.WriteString("- `scenes/<slug>.md` -- scene prose files\n")
	b.WriteString("- `characters/<slug>.yaml` -- character profiles\n")
	b.WriteString("- `book.yaml` -- chapter structure and scene ordering\n")
	b.WriteString("- `storydb/` -- CSV files with indexed continuity data\n\n")

	b.WriteString("## Strategy\n\n")
	b.WriteString("1. Start with the nib commands to locate relevant scenes and data.\n")
	b.WriteString("2. Read specific scene files only when you need the actual prose (exact wording, dialogue, descriptions).\n")
	b.WriteString("3. Use character profiles when the question involves character traits, background, or relationships.\n")
	b.WriteString("4. Synthesize a clear, direct answer. Cite specific scenes (by slug or chapter.scene number) when relevant.\n\n")

	b.WriteString("## Rules\n\n")
	b.WriteString("- Answer the question directly. No preamble, no restating the question.\n")
	b.WriteString("- Cite evidence from the manuscript. Quote relevant passages when they strengthen the answer.\n")
	b.WriteString("- If the indexed data doesn't cover the answer, say so -- don't guess.\n")
	b.WriteString("- Keep the answer concise but complete.\n\n")

	if rangeExpr != "" {
		fmt.Fprintf(&b, "## Scope\n\nLimit your search to chapters/scenes in range: %s\n\n", rangeExpr)
	}

	fmt.Fprintf(&b, "## Question\n\n%s\n", question)

	return b.String()
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
