# nib

A tool to handle the mechanics of your novel so you can focus on writing.

## Talk to your characters

Launch an interactive session where the AI role-plays as any character at any point in your story. Grounded in their profile, personality, and knowledge of events through the scene you specify.

```
nib profile talk elena 12.3
```

## Ask your manuscript anything

Plain-English questions answered from your indexed continuity data, with citations back to specific scenes.

```
nib continuity ask "When did Marcus first learn about the letter?"
```

## Verified continuity data

AI-powered extraction of facts, characters, locations, and timeline from your prose. Every record passes through single-keypress review — hallucinated data never enters your database silently.

```
nib continuity index 1-5
```

## One notation, every command

`3.2` means chapter 3, scene 2. Ranges like `1-5` and `1.1-3.2` work everywhere — index, recap, critique, proof, build, and talk.

## Build to any format

Assemble scenes into a complete manuscript and export to Markdown, DOCX, PDF, or EPUB via Pandoc. Supports Shunn standard manuscript format for submissions.

```
nib manuscript build docx
```

## Principles

- **Local-first.** Your manuscript never leaves your machine.
- **Not a content generator.** Reads, organizes, and checks. AI is an inference engine for structured extraction and interactive creative sessions, not for writing.
- **Git-friendly.** All source-of-truth files are plaintext: YAML, JSONL, Markdown.
- **Agent-neutral.** AI operations go through a backend protocol. Ships with Claude and local model backends. Choose per-project.

## Getting Started

```
nib init my-novel
```

This scaffolds a complete project:

```
my-novel/
  scenes/                   Scene prose files (pure markdown)
  characters/               Character profile YAML files
  storydb/                  Indexed continuity data (CSV)
  appendices/               Supporting documents
  assets/                   Images and other media
  build/                    Generated output (gitignored)
  pandoc-templates/         Manuscript formatting templates
  STYLE.md                  Voice and prose style guide
  TROPES.md                 AI writing tropes to avoid
  book.yaml                 Front matter + chapter/scene ordering
```

Style variants are available for first-person, third-close, and third-omniscient narration (`nib init --style third-close`). Run `nib styles` to see the full list.

## Dependencies

Nib depends on [binder](https://github.com/poiesic/binder) for parsing `book.yaml` and assembling scene files into per-chapter markdown. Binder is imported as a Go library, not a subprocess.

External tools: [Pandoc](https://pandoc.org/) for manuscript builds, `git` for project init, and a configured AI agent backend for continuity, critique, proof, and character sessions.

## Commands

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

`profile talk` launches an interactive AI session where the AI role-plays as a character at a specific point in the story. It uses the character's YAML profile and a character-filtered continuity recap to ground the conversation. The scene argument uses dotted notation (e.g. `37.2`) to set where in the story the character "is" — they know everything up to that point and nothing after. Sessions persist by default; use `--resume` to continue or `--new` to start fresh.

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
nib manuscript search <range> <query>  # search scenes with a plain-English query
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

## AI Backends

Nib delegates all AI operations to external agent backends — standalone executables that communicate via JSON over stdin/stdout. The active backend is resolved from: `NIB_AGENT` env var > `git config nib.agent` > default (`claude`).

**`nib-agent-claude`** — wraps Claude Code CLI. Uses Claude's native tool use, structured output, and session management.

**`nib-agent-local`** — targets OpenAI-compatible local inference servers (LM Studio, ollama, vLLM). Configurable via git config or env vars:

```
nib.agents.local.endpoint    # default: http://localhost:1234/v1
nib.agents.local.model       # default: qwen3-30b-a3b
nib.agents.local.no-think    # default: true (suppress reasoning tokens)
```

The agent protocol is MIT-licensed so third-party backends can be built without AGPL obligations. See `docs/agent-protocol.md` for the specification.

## Building

```
task build                            # build nib + agent binaries
task test                             # run all tests (verbose)
task all                              # fmt, vet, test, build
```

Releases are automated via [GoReleaser](https://goreleaser.com/). Push a version tag and GitHub Actions cross-compiles for macOS (ARM), Linux (ARM + x86), and Windows (x86), then creates a GitHub release:

```
git tag v1.0.0 && git push origin v1.0.0
```

Tags with pre-release identifiers (e.g. `v1.0.0-rc1`) are automatically marked as pre-releases.

## License

Nib is dual-licensed:

- **AGPL-3.0** — the nib CLI, bundled agent backends, and all other code not listed below. See [LICENSE](LICENSE).
- **MIT** — the agent protocol specification (`docs/agent-protocol.md`, `docs/agent-schemas.md`) and the `internal/agent/` package. See [internal/agent/LICENSE](internal/agent/LICENSE).

The intent: if you build a third-party agent backend for nib, you can use the protocol docs and the `internal/agent` integration code under MIT terms without the AGPL applying to your backend.
