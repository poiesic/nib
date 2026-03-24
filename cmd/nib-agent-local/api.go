package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// chatMessage represents a message in the OpenAI chat format.
// Supports text messages, assistant tool calls, and tool results.
type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// toolCall is a function call requested by the model.
type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

// functionCall contains the function name and arguments.
type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// toolDef defines a tool available to the model.
type toolDef struct {
	Type     string          `json:"type"`
	Function toolFunctionDef `json:"function"`
}

// toolFunctionDef describes a function the model can call.
type toolFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// chatRequest is the OpenAI-compatible chat completions request.
type chatRequest struct {
	Model              string              `json:"model"`
	Messages           []chatMessage       `json:"messages"`
	Temperature        float64             `json:"temperature"`
	TopP               float64             `json:"top_p,omitempty"`
	MaxTokens          int                 `json:"max_tokens,omitempty"`
	RepeatPenalty      float64             `json:"repeat_penalty,omitempty"`
	Stream             bool                `json:"stream"`
	ResponseFormat     *responseFormat     `json:"response_format,omitempty"`
	Tools              []toolDef           `json:"tools,omitempty"`
	ChatTemplateKwargs *chatTemplateKwargs `json:"chat_template_kwargs,omitempty"`
}

// chatTemplateKwargs controls model-specific template behavior.
type chatTemplateKwargs struct {
	EnableThinking bool `json:"enable_thinking"`
}

// responseFormat requests structured JSON output.
type responseFormat struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

// chatResponse is the OpenAI-compatible chat completions response.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// chatChoice is a single choice in the response.
type chatChoice struct {
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// streamChunk is a single chunk from a streaming response.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// chatComplete sends a non-streaming chat completion request and returns the full response.
func chatComplete(cfg config, messages []chatMessage, respFmt *responseFormat, tools []toolDef) (*chatChoice, error) {
	req := chatRequest{
		Model:          cfg.Model,
		Messages:       messages,
		Temperature:    cfg.Temperature,
		TopP:           cfg.TopP,
		MaxTokens:      cfg.MaxTokens,
		RepeatPenalty:  cfg.RepeatPenalty,
		Stream:         false,
		ResponseFormat: respFmt,
		Tools:          tools,
	}
	if cfg.NoThink {
		req.ChatTemplateKwargs = &chatTemplateKwargs{EnableThinking: false}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := cfg.Endpoint + "/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &chatResp.Choices[0], nil
}
