# Nib Cheatsheet

## Project Setup

```bash
nib init my-novel                      # scaffold a new project (default: claude backend)
nib init --style=third-close my-novel  # use tight third person style guide
nib init --agent=ollama my-novel       # use a different AI backend
nib styles                             # list available style variants
nib version                            # show version and build info
```

## Chapters

```bash
nib ch add                    # append a new chapter
nib ch add --name "The End"   # append a named chapter
nib ch add --interlude        # append an interlude
nib ch add --at 3             # insert at position 3
nib ch list                   # list all chapters
nib ch name 5 "The End"       # set chapter 5's name
nib ch clear-name 5           # remove chapter 5's name
nib ch move 5 2               # move chapter 5 to position 2
nib ch remove 3               # remove chapter 3
```

## Scenes

```bash
nib sc add 1 opening          # add scene "opening" to chapter 1
nib sc add 1 cafe --at 2      # insert at position 2 in chapter 1
nib sc list                   # list all scenes by chapter
nib sc list --chapter 3       # list scenes in chapter 3
nib sc edit                   # open focused scene in $EDITOR
nib sc edit my-scene           # open specific scene
nib sc rename old-name new-name
nib sc move 3.1 4.2           # move scene from 3.1 to 4.2
nib sc focus 3.2              # set focus to chapter 3, scene 2
nib sc focus                  # show current focus
nib sc unfocus                # clear focus
nib sc remove 1 opening       # remove scene from chapter
```

## Dotted Notation

Used across all commands that take a range:

```
3.2         single scene (chapter 3, scene 2)
3           all scenes in chapter 3
1-5         chapters 1 through 5
1.1-3.2     scene-level range
1,3,5       specific chapters
1.1,2.3     specific scenes
```

## Character Profiles

```bash
nib pr add lance-thurgood     # create profile with scaffold
nib pr list                   # list all profiles (tab-separated)
nib pr edit lance-thurgood    # open profile in $EDITOR
nib pr remove lance-thurgood  # delete profile
```

## Talk to Characters

```bash
nib pr talk lance-thurgood 37.2            # interview Lance at scene 37.2
nib pr talk --resume lance-thurgood 37.2   # resume previous conversation
nib pr talk --new lance-thurgood 37.2      # start fresh (delete old session)
```

## Manuscript

```bash
nib ma toc                    # table of contents with dotted notation
nib ma status                 # word count, chapters, scenes, pages
nib ma build                  # assemble to markdown
nib ma build docx             # build Word document
nib ma build pdf              # build PDF
nib ma build epub             # build EPUB
```

## Critique & Proof

```bash
nib ma critique 3.2           # interactive critique of a scene
nib ma critique 1-5           # critique chapters 1-5 (one session per chapter)
nib ma proof 3.2              # copy-edit a scene (edits in place)
nib ma proof 1-3              # copy-edit a range
nib ma voice lance-thurgood   # check voice consistency (~30% of scenes)
nib ma vo --thorough lance    # deeper check (~60% of scenes)
```

## Continuity

```bash
nib ct index 3.2              # index a single scene
nib ct index 1-5              # index chapters 1-5
nib ct index 1-5 --force      # re-index even if unchanged
nib ct check 3.2              # check scene for continuity errors
nib ct check 1-5              # check a range
nib ct recap 1-5              # JSON recap of chapters 1-5
nib ct recap 1-5 --detailed   # include facts, locations, dates
nib ct recap 1-5 -c lance     # filter to scenes with lance
nib ct characters             # list all indexed characters
nib ct characters 1-5         # characters in a range
nib ct chapters lance bo      # scenes where both appear (AND)
nib ct chapters --or lance bo # scenes per character (OR)
nib ct ask "question"         # ask about the novel in plain English
nib ct ask "question" --range 1-10  # scope to a range
nib ct reset                  # clear all indexed data
```

## Build & Release

```bash
task build                              # build nib + agent binary
task test                               # run all tests (verbose)
task test:short                         # run all tests (quiet)
task all                                # fmt, vet, test, build
task release VERSION=v1.0.0             # tag, cross-compile, GitHub release
```