package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	cfg := loadConfig(req.Effort)

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Prompt},
	}

	tools := availableTools(req.Tools)

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
		err = nil
	}
	if err != nil {
		return err
	}

	resp := agent.CompleteResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpComplete,
		Text:      text,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func extract(req agent.Request) error {
	cfg := loadConfig(req.Effort)

	// Lower temperature for structured output
	cfg.Temperature = 0.2

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt + "\n\nReturn ONLY valid JSON. No markdown, no explanation, no code fences."},
		{Role: "user", Content: req.Prompt},
	}

	tools := availableTools(req.Tools)

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
		err = nil
	}
	if err != nil {
		return err
	}

	// Validate the response is valid JSON
	var data json.RawMessage
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return fmt.Errorf("model returned invalid JSON: %w\nraw: %s", err, text)
	}

	resp := agent.ExtractResponse{
		Type:      agent.RespSuccess,
		Operation: agent.OpExtract,
		Data:      data,
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
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
