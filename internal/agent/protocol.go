package agent

import (
	"encoding/json"
	"fmt"
)

// Effort identifies the thinking/reasoning budget a backend should allocate
// to a request. Backends map this to whatever mechanism fits (CLI flags,
// temperature, thinking tokens). The five levels are ordered from least to
// most effort.
type Effort string

const (
	EffortLow    Effort = "low"
	EffortMedium Effort = "medium"
	EffortHigh   Effort = "high"
	EffortXHigh  Effort = "xhigh"
	EffortMax    Effort = "max"
)

// DefaultEffort is the effort used when none is specified. Keep it at or
// above High so nib produces quality output by default.
const DefaultEffort = EffortHigh

// AllEfforts returns every valid effort level in ascending order.
func AllEfforts() []Effort {
	return []Effort{EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax}
}

// BelowHigh reports whether e is lower than High. Used by the CLI to decide
// whether to warn the user about reduced quality.
func (e Effort) BelowHigh() bool {
	return e == EffortLow || e == EffortMedium
}

// ValidateEffort parses a user-provided string into an Effort. An empty
// string yields DefaultEffort. Unknown values produce an error whose message
// lists the valid options.
func ValidateEffort(s string) (Effort, error) {
	if s == "" {
		return DefaultEffort, nil
	}
	for _, e := range AllEfforts() {
		if string(e) == s {
			return e, nil
		}
	}
	return "", fmt.Errorf("invalid effort %q: must be one of low, medium, high, xhigh, max", s)
}

// Operation identifies which agent capability is being invoked.
type Operation string

const (
	OpSceneProof         Operation = "scene-proof"
	OpChapterProof       Operation = "chapter-proof"
	OpSceneCritique      Operation = "scene-critique"
	OpChapterCritique    Operation = "chapter-critique"
	OpManuscriptCritique Operation = "manuscript-critique"
	OpVoiceCheck         Operation = "voice-check"
	OpContinuityCheck    Operation = "continuity-check"
	OpContinuityAsk      Operation = "continuity-ask"
	OpContinuityIndex    Operation = "continuity-index"
	OpCharacterTalk      Operation = "character-talk"
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
	Effort        Effort          `json:"effort,omitempty"`
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
