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
      "enum": ["complete", "extract", "converse", "scaffold"]
    },
    "prompt": {
      "type": "string",
      "description": "Prompt text for the model"
    },
    "effort": {
      "type": "string",
      "enum": ["low", "medium", "high"],
      "description": "Suggested effort level"
    },
    "tools": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Tools the model should have access to (e.g. Read, Edit, Bash)"
    },
    "schema": {
      "type": "object",
      "description": "JSON Schema for structured output (extract only)"
    },
    "session": {
      "$ref": "#/$defs/session_options"
    },
    "dir": {
      "type": "string",
      "description": "Absolute path to the project root"
    },
    "permissions": {
      "type": "string",
      "description": "Permission mode hint (e.g. acceptEdits)"
    },
    "project_name": {
      "type": "string",
      "description": "Project name for template substitution (scaffold only)"
    }
  },
  "$defs": {
    "session_options": {
      "type": "object",
      "properties": {
        "id":     {"type": "string", "description": "Session identifier (UUID)"},
        "resume": {"type": "boolean", "description": "Resume existing session"},
        "new":    {"type": "boolean", "description": "Delete existing session and start fresh"}
      }
    }
  }
}
```

## Responses

All pipe operations (`complete`, `extract`, `scaffold`) return one of two shapes: success or error. The `converse` operation does not return JSON.

### Error Response

Returned by any operation on failure.

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

### `complete` — Success Response

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "operation", "text"],
  "properties": {
    "type":      {"const": "success"},
    "operation": {"const": "complete"},
    "text":      {"type": "string", "description": "The model's text response"}
  }
}
```

### `extract` — Success Response

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "operation", "data"],
  "properties": {
    "type":      {"const": "success"},
    "operation": {"const": "extract"},
    "data":      {"type": "object", "description": "Structured data conforming to the request schema"}
  }
}
```

### `scaffold` — Success Response

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "operation", "files"],
  "properties": {
    "type":      {"const": "success"},
    "operation": {"const": "scaffold"},
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
        "operation": {"const": "complete"},
        "text":      {"type": "string"}
      }
    },
    {
      "type": "object",
      "required": ["type", "operation", "data"],
      "properties": {
        "type":      {"const": "success"},
        "operation": {"const": "extract"},
        "data":      {"type": "object"}
      }
    },
    {
      "type": "object",
      "required": ["type", "operation", "files"],
      "properties": {
        "type":      {"const": "success"},
        "operation": {"const": "scaffold"},
        "files":     {"type": "array", "items": {"type": "string"}}
      }
    }
  ]
}
```
