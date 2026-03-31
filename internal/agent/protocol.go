package agent

import "encoding/json"

// Operation identifies which agent capability is being invoked.
type Operation string

const (
	OpSceneProof      Operation = "scene-proof"
	OpChapterProof    Operation = "chapter-proof"
	OpSceneCritique   Operation = "scene-critique"
	OpChapterCritique Operation = "chapter-critique"
	OpVoiceCheck      Operation = "voice-check"
	OpContinuityCheck Operation = "continuity-check"
	OpContinuityAsk   Operation = "continuity-ask"
	OpContinuityIndex Operation = "continuity-index"
	OpCharacterTalk   Operation = "character-talk"
	OpProjectScaffold    Operation = "project-scaffold"
	OpManuscriptSearch   Operation = "manuscript-search"
)

// Request is the JSON payload sent to an agent backend on stdin.
type Request struct {
	Operation     Operation       `json:"operation"`
	Dir           string          `json:"dir,omitempty"`
	Paths         []string        `json:"paths,omitempty"`
	CharacterSlug string          `json:"character_slug,omitempty"`
	Question      string          `json:"question,omitempty"`
	Range         string          `json:"range,omitempty"`
	Context       string          `json:"context,omitempty"`
	Schema        json.RawMessage `json:"schema,omitempty"`
	Session       *SessionOptions `json:"session,omitempty"`
	ProjectName   string          `json:"project_name,omitempty"`
}

// SessionOptions controls session behavior for interactive operations.
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

// CompleteResponse is returned by operations that produce text output.
type CompleteResponse struct {
	Type      ResponseType `json:"type"`
	Operation Operation    `json:"operation"`
	Text      string       `json:"text"`
}

// IndexResponse is returned by a successful continuity-index operation.
type IndexResponse struct {
	Type      ResponseType    `json:"type"`
	Operation Operation       `json:"operation"`
	Data      json.RawMessage `json:"data"`
}

// ScaffoldResponse is returned by a successful project-scaffold operation.
type ScaffoldResponse struct {
	Type      ResponseType `json:"type"`
	Operation Operation    `json:"operation"`
	Files     []string     `json:"files"`
}
