package agent

import "encoding/json"

// Operation identifies which agent capability is being invoked.
type Operation string

const (
	OpComplete Operation = "complete"
	OpExtract  Operation = "extract"
	OpConverse Operation = "converse"
	OpScaffold Operation = "scaffold"
)

// Request is the JSON payload sent to an agent backend on stdin.
type Request struct {
	Operation   Operation       `json:"operation"`
	Prompt      string          `json:"prompt"`
	Effort      string          `json:"effort,omitempty"`
	Tools       []string        `json:"tools,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Session     *SessionOptions `json:"session,omitempty"`
	Dir         string          `json:"dir,omitempty"`
	Permissions string          `json:"permissions,omitempty"`
	ProjectName string          `json:"project_name,omitempty"`
}

// SessionOptions controls session behavior for converse operations.
type SessionOptions struct {
	ID     string `json:"id,omitempty"`
	Resume bool   `json:"resume,omitempty"`
	New    bool   `json:"new,omitempty"`
}

// ResponseType identifies success or error.
type ResponseType string

const (
	RespSuccess ResponseType = "success"
	RespError   ResponseType = "error"
)

// ResponseEnvelope is the common wrapper for all agent responses.
// Parse this first to check Type, then unmarshal into the operation-specific type.
type ResponseEnvelope struct {
	Type      ResponseType `json:"type"`
	Operation Operation    `json:"operation,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// CompleteResponse is returned by a successful complete operation.
type CompleteResponse struct {
	Type      ResponseType `json:"type"`
	Operation Operation    `json:"operation"`
	Text      string       `json:"text"`
}

// ExtractResponse is returned by a successful extract operation.
type ExtractResponse struct {
	Type      ResponseType    `json:"type"`
	Operation Operation       `json:"operation"`
	Data      json.RawMessage `json:"data"`
}

// ScaffoldResponse is returned by a successful scaffold operation.
type ScaffoldResponse struct {
	Type      ResponseType `json:"type"`
	Operation Operation    `json:"operation"`
	Files     []string     `json:"files"`
}
