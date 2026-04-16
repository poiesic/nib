package main

import (
	"encoding/json"
	"fmt"
	"os"
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
		err = sceneCritique(req)
	case agent.OpChapterCritique:
		err = chapterCritique(req)
	case agent.OpManuscriptCritique:
		err = manuscriptCritique(req)
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
	prompt := fmt.Sprintf("Proofread the following scene files for mechanical errors only: %s\n\n"+
		"Fix grammar, punctuation, spelling, missing words, duplicated words, and formatting errors. "+
		"Do NOT tighten prose, improve word choice, remove filter words, or restructure sentences. "+
		"Edit the files directly, then print a brief summary of what you fixed.",
		strings.Join(req.Paths, " "))
	return completePipe(prompt, proofSystemPrompt, req.Effort, []string{"Read", "Edit"}, req.Operation)
}

func sceneCritique(req agent.Request) error {
	prompt := fmt.Sprintf("Review the following scene files: %s", strings.Join(req.Paths, " "))
	return converseSession(req.Effort, prompt, critiqueSystemPrompt, nil)
}

func chapterCritique(req agent.Request) error {
	prompt := fmt.Sprintf("Review the following chapter (all scenes in order): %s", strings.Join(req.Paths, " "))
	return converseSession(req.Effort, prompt, critiqueSystemPrompt, nil)
}

func manuscriptCritique(req agent.Request) error {
	prompt := fmt.Sprintf(
		"Review the complete novel manuscript in the single file at %s as one unified work. "+
			"Do NOT review it chapter-by-chapter and stitch the pieces together. "+
			"Read the whole file, then evaluate the book as a single object: overall arc, "+
			"macro-pacing across chapters, thematic through-lines, character arcs across the full work, "+
			"and structural problems that only reveal themselves at book scale.",
		strings.Join(req.Paths, " "))
	return converseSession(req.Effort, prompt, critiqueSystemPrompt, nil)
}

func voiceCheck(req agent.Request) error {
	prompt := fmt.Sprintf("Check voice consistency for character %q across these scenes: %s\n\n"+
		"Read the character profile from characters/%s.yaml first, then read each scene. "+
		"Report any places where the character's dialogue or POV narration doesn't match their established voice.",
		req.CharacterSlug, strings.Join(req.Paths, " "), req.CharacterSlug)
	return completePipe(prompt, systemPrompt, req.Effort, []string{"Read"}, req.Operation)
}

func continuityCheck(req agent.Request) error {
	prompt := fmt.Sprintf("Check for continuity errors in these scenes: %s\n\n"+
		"Use `nib ct recap` to understand what has been established, then read the scenes. "+
		"Report contradictions in facts, timelines, character knowledge, or physical details.",
		strings.Join(req.Paths, " "))
	return completePipe(prompt, systemPrompt, req.Effort, []string{"Read", "Bash"}, req.Operation)
}

func continuityAsk(req agent.Request) error {
	prompt := buildAskPrompt(req.Question, req.Range)
	return completePipe(prompt, systemPrompt, req.Effort, []string{"Read", "Bash"}, req.Operation)
}

func continuityIndex(req agent.Request) error {
	cfg := loadConfig(req.Effort)
	cfg.Temperature = 0.2

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt + "\n\nReturn ONLY valid JSON. No markdown, no explanation, no code fences."},
		{Role: "user", Content: req.Context},
	}

	tools := availableTools([]string{"Read"})

	var text string
	var err error
	if len(tools) > 0 {
		text, _, err = runWithTools(cfg, messages, tools, 10)
	} else {
		respFmt := &responseFormat{
			Type:       "json_schema",
			JSONSchema: req.Schema,
		}
		choice, cerr := chatComplete(cfg, messages, respFmt, nil)
		if cerr != nil {
			return cerr
		}
		text = choice.Message.Content
	}
	if err != nil {
		return err
	}

	var data json.RawMessage
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return fmt.Errorf("model returned invalid JSON: %w\nraw: %s", err, text)
	}

	resp := agent.IndexResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpContinuityIndex,
		Data:      data,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func characterTalk(req agent.Request) error {
	return converseSession(req.Effort, req.Context, systemPrompt, req.Session)
}

// completePipe runs a non-interactive completion and writes a CompleteResponse to stdout.
func completePipe(prompt, sysPrompt string, effort agent.Effort, toolNames []string, op agent.Operation) error {
	cfg := loadConfig(effort)

	messages := []chatMessage{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: prompt},
	}

	tools := availableTools(toolNames)

	var text string
	var err error
	if len(tools) > 0 {
		text, _, err = runWithTools(cfg, messages, tools, 10)
	} else {
		choice, cerr := chatComplete(cfg, messages, nil, nil)
		if cerr != nil {
			return cerr
		}
		text = choice.Message.Content
	}
	if err != nil {
		return err
	}

	resp := agent.CompleteResponse{
		Type:      agent.RespSuccess,
		Operation: op,
		Text:      text,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
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
