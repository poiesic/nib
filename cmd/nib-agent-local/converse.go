package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/poiesic/nib/internal/agent"
)

// converseSession runs an interactive chat session with the local model.
func converseSession(effort, initialPrompt, sysPrompt string, session *agent.SessionOptions) error {
	cfg := loadConfig(effort)

	// Load or start conversation history
	var messages []chatMessage
	sessionPath := sessionFilePathFromOpts(session)

	if session != nil && session.New {
		os.Remove(sessionPath)
	}

	if session != nil && session.Resume {
		loaded, err := loadSession(sessionPath)
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}
		messages = loaded
	}

	// Initialize with system prompt if starting fresh
	if len(messages) == 0 {
		messages = []chatMessage{
			{Role: "system", Content: sysPrompt},
		}
		// Add the initial prompt as the first user message
		if initialPrompt != "" {
			messages = append(messages, chatMessage{Role: "user", Content: initialPrompt})

			// Get and display the first response
			text, err := chatStream(cfg, messages, os.Stdout)
			if err != nil {
				return err
			}
			messages = append(messages, chatMessage{Role: "assistant", Content: text})
			fmt.Fprintln(os.Stdout)
		}
	}

	// Interactive loop
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Fprint(os.Stdout, "\n> ")
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Fprint(os.Stdout, "> ")
			continue
		}
		if input == "/quit" || input == "/exit" {
			break
		}

		messages = append(messages, chatMessage{Role: "user", Content: input})

		fmt.Fprintln(os.Stdout)
		text, err := chatStream(cfg, messages, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			// Remove the failed user message
			messages = messages[:len(messages)-1]
			fmt.Fprint(os.Stdout, "\n> ")
			continue
		}
		messages = append(messages, chatMessage{Role: "assistant", Content: text})
		fmt.Fprint(os.Stdout, "\n\n> ")
	}

	// Save session
	if session != nil && session.ID != "" {
		if err := saveSession(sessionPath, messages); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save session: %v\n", err)
		}
	}

	return nil
}

// chatStream sends a streaming chat completion and writes tokens to w as they arrive.
// Returns the full response text.
func chatStream(cfg config, messages []chatMessage, w io.Writer) (string, error) {
	req := chatRequest{
		Model:         cfg.Model,
		Messages:      messages,
		Temperature:   cfg.Temperature,
		TopP:          cfg.TopP,
		MaxTokens:     cfg.MaxTokens,
		RepeatPenalty: cfg.RepeatPenalty,
		Stream:        true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	url := cfg.Endpoint + "/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("calling %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			token := chunk.Choices[0].Delta.Content
			full.WriteString(token)
			fmt.Fprint(w, token)
		}
	}

	return full.String(), nil
}

// sessionFilePathFromOpts returns the path for storing conversation history.
func sessionFilePathFromOpts(session *agent.SessionOptions) string {
	if session == nil || session.ID == "" {
		return ""
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(homeDir, ".nib", "sessions")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, session.ID+".json")
}

func loadSession(path string) ([]chatMessage, error) {
	if path == "" {
		return nil, fmt.Errorf("no session ID")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var messages []chatMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func saveSession(path string, messages []chatMessage) error {
	if path == "" {
		return nil
	}
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
