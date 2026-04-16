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
- **stdin** receives a JSON request object (pipe mode) or is passed through to the terminal (interactive mode).
- **stdout** receives the JSON response (pipe mode) or is passed through to the terminal (interactive mode).
- **stderr** is always passed through to the terminal. Use it for diagnostic messages.
- **Exit code 0** means success. Non-zero means failure.

### Pipe vs Interactive Mode

**Pipe operations** (`scene-proof`, `chapter-proof`, `voice-check`, `continuity-check`, `continuity-ask`, `continuity-index`, `manuscript-search`, `project-scaffold`) send the request as JSON on stdin and expect a JSON response on stdout.

**Interactive operations** (`scene-critique`, `chapter-critique`, `manuscript-critique`, `character-talk`) need the terminal for user interaction. The request is written to a temporary file and its path is passed via the `NIB_AGENT_REQUEST_FILE` environment variable. The backend reads the request from this file (and deletes it), then uses stdin/stdout/stderr for the interactive session. No JSON response is expected.

## Request Format

Every operation sends a single JSON object:

```json
{
  "operation": "scene-proof",
  "dir": "/absolute/path/to/project",
  "paths": ["scenes/foo.md", "scenes/bar.md"],
  "character_slug": "lance-thurgood",
  "question": "Who drives Bo to Elko?",
  "range": "1-5",
  "context": "...",
  "schema": { ... },
  "session": { "id": "...", "resume": false, "new": false },
  "project_name": "my-novel",
  "effort": "high"
}
```

All fields except `operation` are optional. Which fields are present depends on the operation.

### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `operation` | string | **Required.** The operation to perform. |
| `dir` | string | Absolute path to the project root. Working directory is also set to this. |
| `paths` | string[] | Scene/chapter file paths (relative to project root). |
| `character_slug` | string | Character identifier (e.g. `lance-thurgood`). |
| `question` | string | Plain-English question about the manuscript. |
| `range` | string | Optional chapter/scene range to scope a query (e.g. `1-5`). |
| `context` | string | Pre-assembled prompt content (e.g. character profile + recap for `character-talk`, or indexing prompt for `continuity-index`). |
| `schema` | object | JSON Schema for structured output (`continuity-index` only). |
| `session` | object | Session management options for interactive operations. |
| `project_name` | string | Project name for template substitution (`project-scaffold` only). |
| `effort` | string | Reasoning effort for the request: `low`, `medium`, `high`, `xhigh`, or `max`. Backends MUST treat an empty value as `high` (nib's default). Backends map the level to whatever mechanism fits (CLI flag, thinking tokens, sampling temperature). Levels above `high` are for models that expose larger reasoning budgets; backends that have no additional capacity should treat `xhigh`/`max` as equivalent to `high`. |

## Response Format

All pipe operations return a JSON object on stdout with a `type` field that is either `"success"` or `"error"`.

### Success

Success responses include `type`, `operation`, and operation-specific fields:

```json
{
  "type": "success",
  "operation": "scene-proof",
  "text": "3 comma fixes, 1 apostrophe."
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

## Operations

### `scene-proof`

Mechanical proofreading of scene files. Fix grammar, punctuation, spelling, typos, and formatting. Do not make taste decisions or tighten prose. Edit files directly as a side effect.

**Request fields:** `paths`, `dir`
**Response type:** text (summary of fixes)
**Used by:** `nib ma proof` (with dotted scene refs)

---

### `chapter-proof`

Same as `scene-proof` but at chapter scope. The `paths` field contains all scenes in the chapter.

**Request fields:** `paths`, `dir`
**Response type:** text (summary of fixes)
**Used by:** `nib ma proof` (with whole-chapter refs)

---

### `scene-critique`

Interactive editorial review of a scene. The backend takes over the terminal for a conversation about prose quality, pacing, voice, and purpose.

**Request fields:** `paths`, `dir`
**Response type:** interactive (no JSON response)
**Used by:** `nib ma critique` (with dotted scene refs)

---

### `chapter-critique`

Interactive editorial review of a chapter. The `paths` field contains all scenes in the chapter in narrative order.

**Request fields:** `paths`, `dir`
**Response type:** interactive (no JSON response)
**Used by:** `nib ma critique` (with whole-chapter refs)

---

### `manuscript-critique`

Interactive editorial review of the complete manuscript as a single unified work. The `paths` field contains exactly one path: the absolute path to a pre-assembled single-file markdown copy of the whole book (nib writes this to `build/manuscript-full.md` before dispatching).

Backends MUST NOT review the manuscript chapter-by-chapter and concatenate the results. The one-file payload is specifically designed to make the "let me review chapter 1, then chapter 2, ..." failure mode impossible. Read the file as one object and evaluate at book scale: overall arc, act structure, macro-pacing, thematic through-lines, character arcs across the full work, and structural problems that only reveal themselves at book scale.

**Request fields:** `paths` (length 1), `dir`
**Response type:** interactive (no JSON response)
**Used by:** `nib ma critique` (no range argument)

---

### `voice-check`

Analyze character voice consistency across sampled scenes. Read the character's profile and check that their dialogue and POV narration match their established voice.

**Request fields:** `character_slug`, `paths`, `dir`
**Response type:** text (analysis)
**Used by:** `nib ma voice`

---

### `continuity-check`

Detect continuity errors in the specified scenes. Check for contradictions in facts, timelines, character knowledge, and physical details.

**Request fields:** `paths`, `dir`
**Response type:** text (findings)
**Used by:** `nib ct check`

---

### `continuity-ask`

Answer a research question about the manuscript. The backend should use available tools (file reads, nib CLI commands) to find evidence before answering.

**Request fields:** `question`, `range` (optional), `dir`
**Response type:** text (answer)
**Used by:** `nib ct ask`

---

### `continuity-index`

Extract structured continuity data from a scene. The `context` field contains the assembled prompt and the `schema` field contains the JSON Schema the response must conform to.

**Request fields:** `context`, `schema`, `dir`

**Success response:**
```json
{
  "type": "success",
  "operation": "continuity-index",
  "data": { ... }
}
```

The `data` field contains structured data conforming to the provided schema. The backend is responsible for enforcing schema compliance — how it does so is implementation-specific (JSON mode, guided generation, post-validation, etc.).

**Used by:** `nib ct index`

---

### `character-talk`

Interactive in-character interview. The `context` field contains a pre-assembled prompt with the character's profile and story recap through a specific scene. The backend uses this as the initial message for a conversation.

**Request fields:** `context`, `session`, `dir`
**Response type:** interactive (no JSON response)

**Session options:**

| Field    | Type   | Meaning |
|----------|--------|---------|
| `id`     | string | Session identifier. Use for persistence/resumption. |
| `resume` | bool   | Resume an existing session instead of starting new. When true, `context` is empty. |
| `new`    | bool   | Delete any existing session with this ID and start fresh. |

Session persistence is backend-specific.

**Used by:** `nib pr talk`

---

### `manuscript-search`

Natural-language search across a set of scene files. Return matching lines with file and line-number references.

**Request fields:** `question` (the query), `paths`, `dir`
**Response type:** text (list of matches or "No matches found.")
**Used by:** `nib ma search`

---

### `project-scaffold`

Write agent-specific project files during `nib init`. Non-interactive.

**Request fields:** `dir`, `project_name`

**Success response:**
```json
{
  "type": "success",
  "operation": "project-scaffold",
  "files": [
    "CLAUDE.md",
    ".claude/skills/copy-edit/SKILL.md"
  ]
}
```

The `files` array lists relative paths of files the backend created. The backend writes files directly to `dir`.

**Used by:** `nib init`

## Error Handling

Backends have two ways to report errors:

1. **Structured (preferred):** Return `{"type": "error", "error": "message"}` on stdout with exit code 0. Nib surfaces the message directly.
2. **Process-level:** Write to stderr and exit non-zero. Nib captures stderr and includes it in the error message.

For interactive operations, only process-level errors apply since the backend owns stdout.

## Design Principles

The protocol defines **what** to do, not **how**. Each operation is a domain concept (proof, critique, voice-check) rather than a generic AI primitive (complete, extract). This means:

- **Backends own prompt construction.** Nib sends structured data (file paths, character slugs, questions). The backend decides how to prompt its model.
- **Backends own execution strategy.** Whether to use tools, streaming, multiple passes, or guided generation is the backend's decision.
- **Backends own most configuration.** Temperature, model selection, and tool permissions are internal to the backend. The one exception is `effort`, which is user-facing (exposed via `--effort` on nib commands) and therefore part of the request — backends must honor it.
- **Swapping backends changes implementation, not behavior.** Every backend implements the same operations with the same semantic contract.

## Minimal Example

A backend that handles proof operations (useful for testing):

```bash
#!/usr/bin/env bash
# nib-agent-echo

request=$(cat)
operation=$(echo "$request" | jq -r .operation)

case "$operation" in
  scene-proof|chapter-proof)
    printf '{"type":"success","operation":"%s","text":"No issues found."}\n' "$operation"
    ;;
  project-scaffold)
    echo '{"type":"success","operation":"project-scaffold","files":[]}'
    ;;
  scene-critique|chapter-critique|character-talk)
    echo "Interactive operations not supported" >&2
    exit 1
    ;;
  *)
    printf '{"type":"error","error":"unknown operation: %s"}\n' "$operation"
    ;;
esac
```

Make it executable, put it on PATH as `nib-agent-echo`, and test with `NIB_AGENT=echo nib ma proof 1.1`.

## Reference Implementations

- `cmd/nib-agent-claude/` — Claude Code CLI backend (skills, session management)
- `cmd/nib-agent-local/` — Local model backend (OpenAI-compatible API, tool loop)
