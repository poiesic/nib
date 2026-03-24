# Nib Agent Protocol

Nib delegates all AI operations to external **agent backends** — standalone executables that communicate with nib via JSON over stdin/stdout. This document specifies the protocol so you can implement a backend for any AI provider.

## Discovery

Nib finds the active backend by name:

1. `NIB_AGENT` environment variable (highest priority)
2. `git config --get nib.agent` (per-project)
3. Default: `claude`

The name maps to an executable: `nib-agent-<name>`. For example, agent name `ollama` maps to `nib-agent-ollama` on `PATH`.

## Invocation

Nib runs the agent binary as a subprocess:

- **Working directory** is set to the project root.
- **stdin** receives a JSON request object.
- **stdout** receives the JSON response (for pipe operations) or is passed through to the terminal (for interactive operations).
- **stderr** is passed through to the terminal. Use it for diagnostic messages.
- **Exit code 0** means success. Non-zero means failure.

## Request Format

Every operation sends a single JSON object on stdin:

```json
{
  "operation": "complete",
  "prompt": "...",
  "effort": "medium",
  "tools": ["Read", "Bash"],
  "schema": { ... },
  "session": { "id": "...", "resume": false, "new": false },
  "dir": "/absolute/path/to/project",
  "permissions": "acceptEdits",
  "project_name": "my-novel"
}
```

All fields except `operation` are optional. Which fields are present depends on the operation.

## Response Format

All pipe operations (`complete`, `extract`, `scaffold`) return a JSON object on stdout with a `type` field that is either `"success"` or `"error"`.

### Success

Success responses include `type`, `operation`, and operation-specific fields:

```json
{
  "type": "success",
  "operation": "complete",
  "text": "..."
}
```

Nib validates that `type` is `"success"` and `operation` matches the requested operation. Missing or mismatched fields produce a clear error for the backend author.

### Error

Error responses include `type` and `error`:

```json
{
  "type": "error",
  "error": "model not available"
}
```

Backends should prefer structured errors over stderr when possible. Nib surfaces the `error` field directly to the user.

### Response Schemas

Each operation's success response can be validated with JSON Schema using `oneOf`:

```json
{
  "oneOf": [
    {
      "type": "object",
      "required": ["type", "operation", ...],
      "properties": {
        "type": {"const": "success"},
        "operation": {"const": "complete"},
        ...
      }
    },
    {
      "type": "object",
      "required": ["type", "error"],
      "properties": {
        "type": {"const": "error"},
        "error": {"type": "string"}
      }
    }
  ]
}
```

## Operations

### `complete`

Send a prompt, get text back. Non-interactive.

**Request fields:**
- `prompt` — the prompt text
- `effort` — suggested effort level (`low`, `medium`, `high`)
- `tools` — tools the model should have access to (e.g. `["Read", "Bash"]`)
- `dir` — project root (working directory is also set to this)

**Success response:**
```json
{
  "type": "success",
  "operation": "complete",
  "text": "The model's response text."
}
```

**Used by:** `nib ct ask`, `nib ma proof`, `nib ct check`, `nib ma voice`

---

### `extract`

Send a prompt with a JSON schema, get structured data back. Non-interactive.

**Request fields:**
- `prompt` — the prompt text
- `schema` — a JSON Schema object describing the expected response structure
- `effort` — suggested effort level
- `tools` — tools the model should have access to
- `dir` — project root

**Success response:**
```json
{
  "type": "success",
  "operation": "extract",
  "data": { ... }
}
```

The `data` field contains the structured response conforming to the provided schema. The backend is responsible for enforcing schema compliance — how it does so is implementation-specific (JSON mode, guided generation, post-validation, etc.).

**Used by:** `nib ct index`

---

### `converse`

Launch an interactive session. The user talks to the model directly.

**Request fields:**
- `prompt` — initial message (empty string if resuming)
- `effort` — suggested effort level
- `tools` — tools the model should have access to (omitted = unrestricted)
- `session` — session management options (see below)
- `permissions` — permission mode hint (e.g. `acceptEdits`)
- `dir` — project root

**Response:** None. The backend takes over stdout/stderr for the interactive session. Exit code indicates success or failure. The typed response format does not apply to `converse`.

**TTY handling:** Since stdin carries the JSON request, the backend must reopen the terminal for interactive input:

```go
// Go
tty, _ := os.Open("/dev/tty")  // Windows: "CON"
```

```python
# Python
tty = open("/dev/tty", "r")  # Windows: open("CON", "r")
```

```bash
# Shell
exec 3</dev/tty  # read user input from fd 3
```

Read the full JSON request from stdin first, then switch to the TTY for user interaction.

**Session options:**

| Field    | Type   | Meaning |
|----------|--------|---------|
| `id`     | string | Session identifier (UUID). Use for persistence/resumption. |
| `resume` | bool   | Resume an existing session instead of starting new. |
| `new`    | bool   | Delete any existing session with this ID and start fresh. |

Session persistence is backend-specific. The `id` is a deterministic UUID v5 generated by nib — the same inputs always produce the same session ID.

**Used by:** `nib ma critique`, `nib pr talk`

---

### `scaffold`

Write agent-specific project files during `nib init`. Non-interactive.

**Request fields:**
- `dir` — absolute path to the project directory
- `project_name` — the project name (for template substitution)

**Success response:**
```json
{
  "type": "success",
  "operation": "scaffold",
  "files": [
    "CLAUDE.md",
    ".claude/skills/copy-edit/SKILL.md"
  ]
}
```

The `files` array lists relative paths of files the backend created. The backend writes files directly to `dir` — nib does not process the response further.

**Used by:** `nib init`

## Error Handling

Backends have two ways to report errors:

1. **Structured (preferred):** Return `{"type": "error", "error": "message"}` on stdout with exit code 0. Nib surfaces the message directly.
2. **Process-level:** Write to stderr and exit non-zero. Nib captures stderr and includes it in the error message.

For `converse`, only process-level errors apply since the backend owns stdout.

## Effort Levels

The `effort` field is a hint, not a directive. Map it to whatever makes sense for your provider:

| Effort   | Intent |
|----------|--------|
| `low`    | Fast, cheap. Mechanical tasks like copy-editing. |
| `medium` | Balanced. Structured extraction. |
| `high`   | Thorough. Analysis, critique, complex queries. |

## Tools

The `tools` array names capabilities the model should have access to. Common values:

- `Read` — read files from the project
- `Edit` — modify files in the project
- `Bash` — execute shell commands

How you implement tool access depends on your provider. Some models support native tool use; others may need the tools described in the prompt.

## Minimal Example

A backend that echoes the prompt (useful for testing):

```bash
#!/usr/bin/env bash
# nib-agent-echo

request=$(cat)
operation=$(echo "$request" | jq -r .operation)

case "$operation" in
  complete)
    prompt=$(echo "$request" | jq -r .prompt)
    printf '{"type":"success","operation":"complete","text":"Echo: %s"}\n' "$prompt"
    ;;
  extract)
    echo '{"type":"success","operation":"extract","data":{}}'
    ;;
  converse)
    echo "Converse not supported" >&2
    exit 1
    ;;
  scaffold)
    echo '{"type":"success","operation":"scaffold","files":[]}'
    ;;
  *)
    printf '{"type":"error","error":"unknown operation: %s"}\n' "$operation"
    ;;
esac
```

Make it executable, put it on PATH as `nib-agent-echo`, and test with `NIB_AGENT=echo nib ct ask "hello"`.

## Reference Implementation

See `cmd/nib-agent-claude/` in the nib repository for a complete implementation targeting Claude Code CLI.
