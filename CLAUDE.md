# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What Is Nib

Nib is a novel-writing CLI tool that consolidates manuscript assembly, build automation, and project scaffolding into a single Go binary. It replaces a previous workflow of separate `binder` + `Taskfile` tools.

## Build and Test Commands

Uses Task Runner (`task` command). Run a single test with `go test -v -run TestName ./internal/package/`.

| Command | Description |
|---------|-------------|
| `task build` | Build binary to `build/nib` |
| `task test` | Run all tests (verbose) |
| `task test:short` | Run all tests (quiet) |
| `task all` | fmt, vet, test, build |
| `task clean` | Remove build artifacts |

## Architecture

### Dependency on Binder

Nib depends on `github.com/poiesic/binder`, a sibling repo at `../binder`. The `go.mod` uses a `replace` directive for local development. Binder provides `LoadBook`, `AssembleMarkdown`, and `OutputFiles` -- the core logic for parsing `book.yaml` and assembling scene files into per-chapter markdown.

### Agent Protocol

Nib delegates AI operations to swappable agent backends via a domain-operation protocol. The protocol defines **what** to do; each backend decides **how**.

**`internal/agent/protocol.go`** -- Request/response types and operation constants. Operations are domain-specific (not generic "complete" or "converse"):

| Operation | Mode | Description |
|-----------|------|-------------|
| `scene-proof` | pipe (text) | Mechanical proofreading of scene files |
| `chapter-proof` | pipe (text) | Mechanical proofreading at chapter scope |
| `scene-critique` | interactive | Editorial review of a scene |
| `chapter-critique` | interactive | Editorial review of a chapter |
| `manuscript-critique` | interactive | Editorial review of the whole manuscript |
| `voice-check` | pipe (text) | Character voice consistency analysis |
| `continuity-check` | pipe (text) | Continuity error detection |
| `continuity-ask` | pipe (text) | Research question about the manuscript |
| `continuity-index` | pipe (JSON) | Structured data extraction for indexing |
| `character-talk` | interactive | In-character interview session |
| `project-scaffold` | pipe (files) | Agent-specific project scaffolding |

**`internal/agent/dispatch.go`** -- One exported function per operation (e.g. `agent.SceneProof(paths, dir)`). Callers pass structured domain data, not prompts. Pipe-mode ops return text/JSON; interactive ops take over the TTY.

**Agent backends** (`cmd/nib-agent-claude/`, `cmd/nib-agent-local/`) -- Each backend implements all operations. The Claude backend uses Claude Code skills and CLI flags. The local backend uses inline system prompts and an OpenAI-compatible API. Effort levels, tool permissions, and prompt construction are internal to each backend.

**Request dispatch:** Pipe-mode requests are JSON on stdin; interactive requests use a temp file (`NIB_AGENT_REQUEST_FILE` env var) so stdin flows to the agent's TTY. Agent selection: `NIB_AGENT` env > `git config nib.agent` > default `"claude"`. Binary name: `nib-agent-{name}`.

### Package Structure

**`cmd/nib/main.go`** -- Thin CLI wrapper using `urfave/cli/v3`. All logic lives in `internal/`.

**`internal/config`** -- Project root detection. `FindProjectRoot` walks up directories looking for `book.yaml`. Used by both `build` and `status`.

**`internal/project`** -- `nib init` scaffolding. `Init()` validates the project name (lowercase alphanumeric + hyphens), creates the directory tree, renders Go `text/template` files, and calls `agent.ProjectScaffold()` for agent-specific files.

**`internal/project/templates`** -- `embed.FS` containing `.tmpl` files and `skills/` directory. Templates use `{{.ProjectName}}` for substitution. Skills are static SKILL.md files copied as-is into `.claude/skills/`.

**`internal/manuscript`** -- Build, status, review, and voice logic:
- `build.go` -- `Build()` orchestrates: find project root, assemble via binder, invoke pandoc. Supports docx/pdf/epub/all formats.
- `pandoc.go` -- Pandoc command construction. Uses injected `CommandRunner` for testability. DOCX build detects `pandoc-templates/bin/md2long.sh` for Shunn manuscript format, falling back to plain pandoc.
- `status.go` -- `GetStatus()` computes scene/chapter/interlude counts, word count (pure Go, no `wc`), estimated pages, and finds unassigned scenes (files in `manuscript/` not referenced in `book.yaml`).
- `review.go` -- `Proof()` and `Critique()` dispatch to agent operations. Callers resolve paths and pass structured data; no prompt construction.
- `voice.go` -- `Voice()` samples scenes per character and dispatches `agent.VoiceCheck()`.

**`internal/continuity`** -- Continuity checking, indexing, and research queries. `Check()`, `Ask()`, and `Index()` dispatch to the corresponding agent operations.

**`internal/character`** -- Character profile management and `Talk()` for in-character interviews. `buildTalkPrompt()` assembles profile + recap context; the agent receives it via the `Context` field.

### Testing Patterns

- Pandoc tests use an injected `CommandRunner` that wraps calls in `echo` to capture arguments without invoking pandoc.
- Continuity index tests use an injected `ExtractFunc` to mock agent extraction without calling a real backend.
- Project init tests use `t.TempDir()` and `os.Chdir()` for filesystem isolation.
- Test helpers in `internal/manuscript/helpers_test.go` provide `mkdirAll`, `writeFile`, `chmodExec`.
