package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// availableTools returns tool definitions for the requested tool names.
func availableTools(names []string) []toolDef {
	all := map[string]toolDef{
		"Read": {
			Type: "function",
			Function: toolFunctionDef{
				Name:        "Read",
				Description: "Read a file from the project. Returns the file contents.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"required": ["path"],
					"properties": {
						"path": {"type": "string", "description": "File path relative to project root"}
					}
				}`),
			},
		},
		"Edit": {
			Type: "function",
			Function: toolFunctionDef{
				Name:        "Edit",
				Description: "Replace text in a file. The old_string must match exactly.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"required": ["path", "old_string", "new_string"],
					"properties": {
						"path":       {"type": "string", "description": "File path relative to project root"},
						"old_string": {"type": "string", "description": "Exact text to find"},
						"new_string": {"type": "string", "description": "Replacement text"}
					}
				}`),
			},
		},
		"Bash": {
			Type: "function",
			Function: toolFunctionDef{
				Name:        "Bash",
				Description: "Execute a shell command and return its output.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"required": ["command"],
					"properties": {
						"command": {"type": "string", "description": "Shell command to execute"}
					}
				}`),
			},
		},
	}

	var defs []toolDef
	for _, name := range names {
		if d, ok := all[name]; ok {
			defs = append(defs, d)
		}
	}
	return defs
}

// executeTool runs a tool call and returns the result text.
func executeTool(tc toolCall) string {
	switch tc.Function.Name {
	case "Read":
		return execRead(tc.Function.Arguments)
	case "Edit":
		return execEdit(tc.Function.Arguments)
	case "Bash":
		return execBash(tc.Function.Arguments)
	default:
		return fmt.Sprintf("unknown tool: %s", tc.Function.Name)
	}
}

func execRead(argsJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error parsing arguments: %v", err)
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return fmt.Sprintf("error reading file: %v", err)
	}
	return string(data)
}

func execEdit(argsJSON string) string {
	var args struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error parsing arguments: %v", err)
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return fmt.Sprintf("error reading file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, args.OldString) {
		return fmt.Sprintf("old_string not found in %s", args.Path)
	}
	newContent := strings.Replace(content, args.OldString, args.NewString, 1)
	if err := os.WriteFile(args.Path, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("error writing file: %v", err)
	}
	return "OK"
}

func execBash(argsJSON string) string {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error parsing arguments: %v", err)
	}
	cmd := exec.Command("bash", "-c", args.Command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s\nerror: %v", string(out), err)
	}
	return string(out)
}

// runWithTools executes a chat completion with tool calling in a loop.
// The model can call tools multiple times until it produces a text response.
// maxTurns limits the number of tool-call rounds to prevent infinite loops.
func runWithTools(cfg config, messages []chatMessage, tools []toolDef, maxTurns int) (string, []chatMessage, error) {
	for turn := 0; turn < maxTurns; turn++ {
		choice, err := chatComplete(cfg, messages, nil, tools)
		if err != nil {
			return "", messages, err
		}

		// If no tool calls, we have the final response
		if len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, messages, nil
		}

		// Append the assistant message with tool calls (content may be empty)
		messages = append(messages, chatMessage{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		})

		// Execute each tool call and append results
		for _, tc := range choice.Message.ToolCalls {
			result := executeTool(tc)
			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}

	return "", messages, fmt.Errorf("tool loop exceeded %d turns", maxTurns)
}
