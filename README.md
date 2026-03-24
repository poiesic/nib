# Nib Overview

Nib is a novel-writing CLI tool. It handles project scaffolding, manuscript assembly, build automation, scene management, character profiles, and AI-assisted continuity tracking. It is editorial infrastructure -- it does not generate prose.

## Guiding Principles

- **Local-first.** Manuscript never leaves the author's machine.
- **Not a content generator.** Reads, organizes, and checks. AI is used as an inference engine for structured extraction (continuity indexing) and interactive creative sessions, not for writing.
- **Git-friendly.** All source-of-truth files are plaintext: YAML, JSONL, Markdown.
- **Agent-neutral.** AI operations go through a backend protocol. Ship with Claude and local model backends. Authors choose per-project.

## Dependencies

Nib depends on [binder](https://github.com/poiesic/binder). Binder provides the core logic for parsing `book.yaml` and assembling scene files into per-chapter markdown. Nib imports binder as a Go library, not as a subprocess.

External tools: `pandoc` (for docx/pdf/epub builds), `git` (for project init), and a configured AI agent backend (for continuity indexing, critique, proof, and character talk sessions).

## Command Surface

Aliases are shown in parentheses.

### Project Setup

```
nib init <project-name>               # scaffold a new novel project
      --style <variant>                  # STYLE.md variant (first-person, third-close, third-omniscient)
      --no-style                         # skip STYLE.md creation
      --no-git                           # skip git repo initialization
      --agent <name>                     # AI backend (default: claude)

nib styles                             # list available style variants
nib version                            # show version, commit, and build date
```

### Chapters (ch)

```
nib chapter add [--name] [--interlude] [--at]   # insert chapter into book.yaml
nib chapter list                                 # list chapters with scene counts
nib chapter name <index> <name>                  # set a chapter name
nib chapter clear-name <index>                   # remove a chapter name
nib chapter move <from> <to>                     # move chapter to a new position
nib chapter remove <index>                       # remove chapter (files stay on disk)
```

### Character Profiles (pr)

```
nib profile add <slug>                 # create a new character profile YAML
nib profile list                       # list all character profiles
nib profile edit <slug>                # open profile in your editor
nib profile talk <slug> <scene>        # role-play as a character (see below)
      --resume                           # resume an existing talk session
      --new                              # delete existing session and start fresh
nib profile remove <slug>              # remove a character profile
```

**`profile talk`** launches an interactive AI session where the AI role-plays as a character at a specific point in the story. It uses the character's YAML profile and a character-filtered continuity recap to ground the conversation. Useful for testing dialogue voice, exploring character motivation, or pressure-testing plot decisions. The scene argument uses dotted notation (e.g. `37.2`) to set where in the story the character "is" -- they know everything up to that point and nothing after. Sessions persist by default; use `--resume` to continue or `--new` to start fresh.

### Scenes (sc)

```
nib scene add <ch> <slug> [--at]       # add scene to chapter, create .md file
nib scene list [--chapter <n>]         # list scenes grouped by chapter
nib scene remove <ch> <slug>           # remove from book.yaml (file stays)
nib scene edit [slug]                  # open in $NIB_EDITOR / $VISUAL / $EDITOR
nib scene rename <old> <new>           # rename a scene slug (updates file + book.yaml)
nib scene move <from> [to]             # move via dotted notation (e.g. 3.1 4.2)
nib scene focus [chapter[.scene]]      # set or show current working scene
nib scene unfocus                      # clear focus
```

Scene commands that accept a slug will fall back to the currently focused scene if no slug is given.

### Manuscript (ma)

```
nib manuscript build [format]          # assemble + pandoc (md/docx/pdf/epub/all)
      --scene-headings                   # include scene filenames as headings
nib manuscript status                  # word count, scene stats, unassigned scenes
nib manuscript toc                     # chapter/scene structure in dotted notation
nib manuscript critique <range>        # AI-powered structured scene/chapter review
nib manuscript proof <range>           # AI-powered copy-editing (edits in place)
nib manuscript voice <char> [char...]  # check character voice consistency
      --thorough                         # sample 60% of scenes instead of 30%
```

Range arguments use flexible notation: `1-3` (chapters 1-3), `1.1-2.3` (scene-level range), `1,3,5` (specific chapters), `2.1,2.3` (specific scenes).

### Continuity (ct)

```
nib continuity check <range>           # check scenes for continuity errors
nib continuity recap <range>           # JSON recap of chapters from indexed data
      --character <slug> [-c ...]        # filter to scenes involving specific characters
      --detailed                         # include facts, locations, dates, times
nib continuity index [range]           # extract structured data from scenes via AI
      --force                            # re-index even if scene file hasn't changed
      --verbose                          # print prompt and raw AI response
nib continuity characters [range]      # list characters from indexed data (JSON)
      --all                              # include mentioned characters (default: pov+present)
nib continuity chapters <char> [char]  # list scenes where characters appear (dotted notation)
      --or                               # scenes where ANY character appears (default: AND)
nib continuity ask "<question>"        # ask a plain-English question about the novel
      --range <range>                    # limit search to a chapter/scene range
nib continuity reset [--yes]           # clear all indexed continuity data
```

## Scaffolded Project Layout

`nib init my-novel` creates:

```
my-novel/
  scenes/                   Scene prose files (pure markdown, no frontmatter)
  characters/               Character profile YAML files (one per character)
  storydb/                  CSV tables (scenes, facts, locations, etc.)
  appendices/               Supporting documents
  assets/                   Images and other media
  build/                    Generated output (gitignored)
  pandoc-templates/         Git submodule for manuscript formatting
  STYLE.md                  Voice and prose style guide
  TROPES.md                 AI writing tropes to avoid
  book.yaml                 Front matter + chapter/scene ordering (two-doc YAML)
  .gitignore
```

Agent-specific files are created by the backend's scaffold operation. For example, the Claude backend adds `CLAUDE.md`, `TOOLS.md`, and `.claude/skills/`. The local backend adds `PROJECT.md`.

## Data Model

### book.yaml

Two-document YAML stream. First document is front matter (title, author, contact info). Second document defines the chapter/scene sequence. This is the single source of truth for manuscript ordering.

### Scene Files (scenes/*.md)

Pure prose. No frontmatter, no metadata. Named by slug convention: `{pov-character}-{action}` for regular scenes, `{document-type}-{context}` for interludes.

### Character Profiles (characters/*.yaml)

Human-authored. One file per character with personality, relationships, background. Used by `profile talk` to ground AI character sessions.

### StoryDB (storydb/*.csv)

Machine-curated, human-approved relational data. Five tables:

| File                     | Content                                                      |
| ------------------------ | ------------------------------------------------------------ |
| `scenes.csv`             | Per-scene metadata (POV, location, time, summary)            |
| `facts.csv`              | Established narrative details (events, descriptions, states) |
| `scene_characters.csv`   | Character appearances per scene                              |
| `locations.csv`          | Canonical location details                                   |
| `timeline.csv`           | Temporal sequence of events                                  |

Populated by `nib continuity index`, which uses AI to extract structured data and presents each record for interactive accept/reject/edit review.

### Project State (\.nib/state.json)

Gitignored. Stores transient state like the current focus (working scene).

## Build Pipeline

```
book.yaml
  |
  v
binder.AssembleMarkdown     -- scenes -> numbered chapter .md files + metadata.yaml
  |
  v
pandoc                      -- chapter files -> docx / pdf / epub
  |
  v
build/my-novel.{docx,pdf,epub}
```

DOCX builds detect `pandoc-templates/bin/md2long.sh` for Shunn manuscript format, falling back to plain pandoc if not present. PDF uses xelatex with 12pt, double-spaced, 1-inch margins.

## AI Integration

### Agent Protocol

Nib delegates all AI operations to external **agent backends** — standalone executables that communicate via JSON over stdin/stdout. This decouples nib from any specific AI provider.

The active backend is resolved from: `NIB_AGENT` env var > `git config nib.agent` > default (`claude`). The name maps to an executable: `nib-agent-<name>` on PATH.

Four operations:

| Operation  | Mode        | Purpose                                      |
| ---------- | ----------- | -------------------------------------------- |
| `complete` | Pipe        | Send prompt, get text back                   |
| `extract`  | Pipe        | Send prompt + JSON schema, get structured data |
| `converse` | Interactive | Launch a TTY session with the model          |
| `scaffold` | Pipe        | Write agent-specific project files           |

All pipe responses use typed envelopes (`{"type": "success", "operation": "...", ...}` or `{"type": "error", "error": "..."}`).

See `docs/agent-protocol.md` and `docs/agent-schemas.md` for the full specification.

### Bundled Backends

**`nib-agent-claude`** — wraps Claude Code CLI. Supports all four operations. Uses Claude's native tool use, structured output, and session management.

**`nib-agent-local`** — targets OpenAI-compatible local inference servers (LM Studio, ollama, vLLM). Supports all four operations including tool calling via the OpenAI function calling API. Configurable via git config or env vars:

```
nib.agents.local.endpoint    # default: http://localhost:1234/v1
nib.agents.local.model       # default: qwen3-30b-a3b
nib.agents.local.no-think    # default: true (suppress reasoning tokens)
```

Tested with Qwen3-30B, Qwen3-27B, and Nemotron-3-Nano.

### AI-Powered Features

| Feature                  | Command        | What It Does                                     |
| ------------------------ | -------------- | ------------------------------------------------ |
| Continuity indexing      | `ct index`     | Extracts facts, characters, locations from scenes |
| Continuity checking      | `ct check`     | Detects contradictions across scenes              |
| Continuity Q&A           | `ct ask`       | Answers plain-English questions about the novel   |
| Manuscript critique      | `ma critique`  | Structured multi-category scene/chapter review    |
| Manuscript proof         | `ma proof`     | Line-level copy-editing (edits in place)          |
| Voice consistency        | `ma voice`     | Checks character voice across sampled scenes      |
| Character talk           | `pr talk`      | Interactive role-play as a character              |

### Bundled Skills (Claude backend)

Five skills are scaffolded into Claude-backed projects:

| Skill              | Purpose                                              |
| ------------------ | ---------------------------------------------------- |
| `review-scene`     | Structured scene review with 1-5 ratings             |
| `review-chapter`   | Structured chapter review with 1-5 ratings           |
| `copy-edit`        | Line-level mechanical editing (edits in place)        |
| `continuity-check` | Cross-scene contradiction detection                  |
| `voice-check`      | Per-character voice consistency analysis              |

## Building and Testing

```
task build              # build nib + both agent binaries
task test               # verbose
task test:short         # quiet
task all                # fmt, vet, test, build
task clean              # remove build artifacts
task release VERSION=v1.0.0   # tag, cross-compile (4 platforms), create GitHub release
```

Tests use `testify` for assertions. Pandoc and git commands are tested via injected `CommandRunner` functions that capture arguments without invoking real binaries. Filesystem tests use `t.TempDir()` for isolation. Agent operations use injectable function fields in options structs.
