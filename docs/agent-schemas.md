# Nib Agent Protocol — JSON Schemas

Companion to [agent-protocol.md](agent-protocol.md). These schemas can be used to validate agent backend requests and responses.

## Request

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["operation"],
  "properties": {
    "operation": {
      "type": "string",
      "enum": [
        "scene-proof", "chapter-proof",
        "scene-critique", "chapter-critique",
        "voice-check",
        "continuity-check", "continuity-ask", "continuity-index",
        "character-talk",
        "project-scaffold"
      ]
    },
    "dir": {
      "type": "string",
      "description": "Absolute path to the project root"
    },
    "paths": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Scene/chapter file paths relative to project root"
    },
    "character_slug": {
      "type": "string",
      "description": "Character identifier (e.g. lance-thurgood)"
    },
    "question": {
      "type": "string",
      "description": "Plain-English question about the manuscript"
    },
    "range": {
      "type": "string",
      "description": "Optional chapter/scene range to scope a query (e.g. 1-5)"
    },
    "context": {
      "type": "string",
      "description": "Pre-assembled prompt content (character-talk, continuity-index)"
    },
    "schema": {
      "type": "object",
      "description": "JSON Schema for structured output (continuity-index only)"
    },
    "session": {
      "$ref": "#/$defs/session_options"
    },
    "project_name": {
      "type": "string",
      "description": "Project name for template substitution (project-scaffold only)"
    }
  },
  "$defs": {
    "session_options": {
      "type": "object",
      "properties": {
        "id":     {"type": "string", "description": "Session identifier"},
        "resume": {"type": "boolean", "description": "Resume existing session"},
        "new":    {"type": "boolean", "description": "Delete existing session and start fresh"}
      }
    }
  }
}
```

## Responses

Pipe operations return one of two shapes: success or error. Interactive operations (`scene-critique`, `chapter-critique`, `manuscript-critique`, `character-talk`) do not return JSON.

### Error Response

Returned by any pipe operation on failure.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "error"],
  "properties": {
    "type":  {"const": "error"},
    "error": {"type": "string", "description": "Human-readable error message"}
  }
}
```

### Text Response

Returned by `scene-proof`, `chapter-proof`, `voice-check`, `continuity-check`, `continuity-ask`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "operation", "text"],
  "properties": {
    "type":      {"const": "success"},
    "operation": {"type": "string"},
    "text":      {"type": "string", "description": "The operation's text output"}
  }
}
```

### Index Response

Returned by `continuity-index`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "operation", "data"],
  "properties": {
    "type":      {"const": "success"},
    "operation": {"const": "continuity-index"},
    "data":      {"type": "object", "description": "Structured data conforming to the request schema"}
  }
}
```

### Scaffold Response

Returned by `project-scaffold`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "operation", "files"],
  "properties": {
    "type":      {"const": "success"},
    "operation": {"const": "project-scaffold"},
    "files": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Relative paths of files created in the project directory"
    }
  }
}
```

## Combined Response Schema (oneOf)

Use this to validate any pipe operation response in a single pass:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "oneOf": [
    {
      "type": "object",
      "required": ["type", "error"],
      "properties": {
        "type":  {"const": "error"},
        "error": {"type": "string"}
      }
    },
    {
      "type": "object",
      "required": ["type", "operation", "text"],
      "properties": {
        "type":      {"const": "success"},
        "operation": {"type": "string"},
        "text":      {"type": "string"}
      }
    },
    {
      "type": "object",
      "required": ["type", "operation", "data"],
      "properties": {
        "type":      {"const": "success"},
        "operation": {"const": "continuity-index"},
        "data":      {"type": "object"}
      }
    },
    {
      "type": "object",
      "required": ["type", "operation", "files"],
      "properties": {
        "type":      {"const": "success"},
        "operation": {"const": "project-scaffold"},
        "files":     {"type": "array", "items": {"type": "string"}}
      }
    }
  ]
}
```
