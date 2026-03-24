# Nib - Design Brainstorm

A single Go CLI tool for managing novel manuscripts. Handles project scaffolding,
manuscript assembly, character management, continuity tracking, and build
pipelines. Not a content generator -- editorial infrastructure.

## Guiding Principles

- **Local-first.** Manuscript never leaves the author's machine.
- **Not a content generator.** Reads, organizes, and checks. Doesn't write prose.
- **Claude as inference engine.** Uses Claude Code for intelligent operations
  (fact extraction, continuity checking) rather than embedded local LLMs.
- **Git-friendly.** All source-of-truth files are plaintext (YAML, CSV, Markdown).
- **Convention over configuration.** Consistent project structure means tools
  and skills work without project-specific customization.

## Project Structure

`nib init <project-name>` scaffolds this layout:

```
my-novel/
├── .claude/
│   ├── settings.local.json
│   └── skills/
│       ├── draft-scene/SKILL.md
│       ├── review-scene/SKILL.md
│       ├── continuity-check/SKILL.md
│       ├── copy-edit/SKILL.md
│       └── voice-check/SKILL.md
├── CLAUDE.md
├── STYLE.md
├── book.yaml
├── manuscript/
├── characters/
├── storydb/
│   ├── scenes.csv
│   ├── facts.csv
│   ├── locations.csv
│   ├── scene_characters.csv
│   └── timeline.csv
├── appendices/
├── assets/
├── build/
└── .gitignore
```

## Data Model

### Hybrid YAML + CSV

Two formats, each used where it fits:

- **CSV** for relational data you query across (scenes, facts, timeline,
  character appearances). Queryable with SQL via csvq.
- **YAML** for hierarchical data you read individually (character profiles,
  book spec, project config).

### Character Profiles (characters/*.yaml)

Human-authored, human-maintained. Rich hierarchical data: personality,
relationships, background, family, habits. One file per character.

### Book Spec (book.yaml)

Front matter (title, author, contact info) and chapter/scene ordering.
Multi-document YAML: first document is front matter, second is the book
structure with chapters and scene references.

### Scene Files (manuscript/*.md)

Prose only. No frontmatter, no metadata. Scrib controls file naming using
a consistent slug convention:

- Regular scenes: `{pov-character}-{action/context}` or
  `{pov-character}-{other-character}-{context}`
- Interludes: `{document-type}-{context}`

Examples: `lance-bo-lunch`, `ella-school-volunteer`,
`memo-walsh-safety-protocols`

### StoryDB (storydb/*.csv)

Machine-curated, human-approved relational data. Populated by `scrib
continuity index` which uses Claude to extract structured data from scenes
and proposes changes for author approval.

**scenes.csv** -- Per-scene metadata (POV, location, time, event refs).

**facts.csv** -- Everything established in the narrative. A fact is any
established detail -- events, physical descriptions, relationships, states.
No distinction between "events" and "facts"; a fact has a time if it's
temporal, doesn't if it's descriptive.

**locations.csv** -- Canonical location details.

**scene_characters.csv** -- Which characters appear in which scenes.

**timeline.csv** -- Canonical temporal sequence. A single authored file
that represents when things happen. The timeline is authored, not derived,
because:
- Roughing in the first version needs to be fast
- Easy to see what comes before and after
- Fixing continuity issues often means changing the timeline directly,
  which should be a single-row edit not a multi-file operation

#### Example Queries (via csvq)

```sql
-- What does Lance know by chapter 12?
SELECT f.summary, s.scene, s.time
FROM facts f
JOIN scenes s ON f.scene = s.scene
JOIN scene_characters sc ON s.scene = sc.scene
WHERE sc.character = 'lance-thurgood'
AND s.time <= '2025-03-20'
ORDER BY s.time;

-- Which scenes have characters without profiles?
SELECT DISTINCT sc.character
FROM scene_characters sc
WHERE sc.character NOT IN (
  SELECT slug FROM characters
);
```

## Command Surface

Commands are organized by domain:

### Project

```
nib init <project-name>       # scaffold a new novel project
```

### Manuscript

```
nib manuscript status         # word count, page estimate, unassigned scenes
nib manuscript build [format] # assemble + pandoc (docx, pdf, epub, all)
nib manuscript wordcount      # word count breakdown
nib manuscript watch          # rebuild on changes
```

### Scene

```
nib scene add --pov <char> --summary "description"
                                # create scene file with consistent naming
nib scene add --interlude --type <type> --summary "description"
                                # create interlude scene
nib scene list                # list scenes in narrative order
nib scene move <slug> --after <slug>
                                # reorder scenes in book.yaml
nib scene remove <slug>       # remove from book.yaml (doesn't delete file)
nib scene draft <slug>        # spawn interactive Claude Code with /draft-scene
nib scene review <slug>       # spawn interactive Claude Code with /review-scene
nib scene edit <slug>         # spawn interactive Claude Code with /copy-edit
```

### Chapter

```
nib chapter add [--interlude] # add a chapter to book.yaml
nib chapter list              # list chapters
nib chapter remove            # remove a chapter
```

### Character

```
scrib character add <slug>      # create a character profile
scrib character show <slug>     # display a profile
scrib character list            # list all characters
scrib character status          # profiles, scene counts, missing profiles
```

### Continuity

```
nib continuity index <slug>   # extract facts/metadata from a scene via Claude
nib continuity check <slug>   # check scene for contradictions (warnings)
nib continuity status         # indexed vs unindexed, fact count, warnings
nib continuity facts          # query facts (--character, --location, --scene)
nib continuity timeline       # display timeline
nib continuity timeline --format html
                                # generate static HTML timeline visualization
nib continuity timeline --format yaml
                                # export as single YAML file
nib continuity timeline --format csv
                                # export as CSV
```

## Claude Code Integration

### Spawning Interactive Sessions

Scrib spawns Claude Code as an interactive subprocess for creative work:

```go
cmd := exec.Command("claude", "/draft-scene lance-bo-lunch")
cmd.Dir = projectDir
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Run()
```

The user runs `nib scene draft lance-bo-lunch` and lands in an interactive
Claude Code session with the skill already invoked. When they exit Claude
Code, they're back in their shell.

### Skills as Templates

Claude skills are bundled as templates within scrib. They reference
conventional paths (`book.yaml`, `manuscript/`, `characters/`, `storydb/`)
that are consistent across all nib projects. No project-specific editing
after scaffolding.

Skills can evolve to call nib for context gathering:

```markdown
1. Run `nib scene context <slug>` to get scene metadata,
   surrounding scenes, and relevant character profiles
2. Run `nib continuity status` to check for outstanding warnings
```

### Indexing Pipeline

`nib continuity index <slug>`:

1. Reads the scene from `manuscript/<slug>.md`
2. Reads relevant character profiles and existing storydb data
3. Calls Claude to extract structured data (facts, characters present,
   locations, temporal position)
4. Proposes new/updated CSV rows for author approval
5. Writes approved changes to the storydb CSV files
6. Changes are committed via git like any other source file

## Contradictions

Contradictions are **warnings**, not errors. Scrib reports them but doesn't
block work. The author fixes them when ready.

```
nib continuity check lance-bo-lunch

  WARNING: Bo is at the Wellness Center at 2pm on March 14
  (facts.csv row 47) but bo-ahr-handoff has him in Building 7
  at the same time (facts.csv row 52)

  2 warnings
```

Warnings persist until the underlying contradiction is resolved -- either
by fixing the prose or updating the storydb data. No explicit dismiss or
acknowledge workflow.

## Status Commands

Each domain has its own status view:

```
nib manuscript status

  Scenes: 82
  Chapters: 20 + 10 interludes
  Word count: 91,420
  Est. pages: 366 (250 words/page)
  Unassigned scenes: 3 (in manuscript/ but not in book.yaml)

nib continuity status

  Indexed: 64/82 scenes (18 pending)
  Facts: 312
  Locations: 14
  Warnings: 7 unresolved
  Last indexed: lance-bo-elko (2 hours ago)

scrib character status

  Profiles: 18
  Scenes by character:
    Lance Thurgood    22 scenes
    Ella Mazur        16 scenes
    Mark Thompson     11 scenes
    ...
  Characters in scenes but no profile: 2
    - security-guard
    - heather
```

## Build Pipeline

Scrib absorbs binder's assembly logic as an imported Go package (not a
subprocess). The build pipeline:

```
book.yaml (scene sequence)
    │
    ▼
nib manuscript build
    │
    ├── assemble scenes into numbered chapter .md files
    ├── generate build/metadata.yaml (pandoc front matter)
    │
    ▼
pandoc + templates
    │
    ▼
build/my-novel.docx / .pdf / .epub
```

## Exports

Static file generation for views that benefit from visual presentation:

```
nib continuity timeline --format html    # vertical timeline, color-coded by POV
scrib character show <slug> --format html  # character dossier page
```

Single self-contained HTML files (inline CSS, no dependencies) dropped
into `build/`. Read-only exports -- YAML/CSV remain the source of truth.

## Open Questions

- CSV schema details: exact columns for each storydb table
- csvq integration: import as Go library vs shell out to binary
- Character profile YAML schema: standardize across projects or keep flexible
- How does `nib init` handle pandoc template setup (submodule, copy, download)
- Series support: multiple books in the same world sharing a storydb
- Should `nib continuity index` run automatically after `nib scene draft`
  exits, or always be manual
