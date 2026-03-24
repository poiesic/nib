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

### Package Structure

**`cmd/nib/main.go`** -- Thin CLI wrapper using `urfave/cli/v3`. Defines three commands: `init`, `manuscript build`, `manuscript status`. All logic lives in `internal/`.

**`internal/config`** -- Project root detection. `FindProjectRoot` walks up directories looking for `book.yaml`. Used by both `build` and `status`.

**`internal/project`** -- `nib init` scaffolding. `Init()` validates the project name (lowercase alphanumeric + hyphens), creates the directory tree, renders Go `text/template` files, and copies embedded skill files.

**`internal/project/templates`** -- `embed.FS` containing `.tmpl` files and `skills/` directory. Templates use `{{.ProjectName}}` for substitution. Skills are static SKILL.md files copied as-is into `.claude/skills/`.

**`internal/manuscript`** -- Build and status logic:
- `build.go` -- `Build()` orchestrates: find project root, assemble via binder, invoke pandoc. Supports docx/pdf/epub/all formats.
- `pandoc.go` -- Pandoc command construction. Uses injected `CommandRunner` for testability. DOCX build detects `pandoc-templates/bin/md2long.sh` for Shunn manuscript format, falling back to plain pandoc.
- `status.go` -- `GetStatus()` computes scene/chapter/interlude counts, word count (pure Go, no `wc`), estimated pages, and finds unassigned scenes (files in `manuscript/` not referenced in `book.yaml`).

### Testing Patterns

- Pandoc tests use an injected `CommandRunner` that wraps calls in `echo` to capture arguments without invoking pandoc.
- Project init tests use `t.TempDir()` and `os.Chdir()` for filesystem isolation.
- Test helpers in `internal/manuscript/helpers_test.go` provide `mkdirAll`, `writeFile`, `chmodExec`.
