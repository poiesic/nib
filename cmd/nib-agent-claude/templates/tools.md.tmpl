# Scrib Tools Reference

## Build Commands

| Command                          | Description                                        |
| -------------------------------- | -------------------------------------------------- |
| `nib manuscript build`         | Build manuscript as Markdown (default)             |
| `nib manuscript build docx`    | Build manuscript as DOCX (requires pandoc)         |
| `nib manuscript build pdf`     | Build manuscript as PDF (requires pandoc + xelatex)|
| `nib manuscript build epub`    | Build manuscript as EPUB (requires pandoc)         |
| `nib manuscript build all`     | Build all formats                                  |
| `nib manuscript status`        | Show word count, scene count, and project stats    |

## Continuity Tracking

### Indexing

`nib ct index [range]` extracts structured data from scenes (characters, facts, locations, timeline) using Claude and stores it in storydb.

    nib ct index              # index all unindexed scenes
    nib ct index 3            # index scenes in chapter 3
    nib ct index 1-5          # index chapters 1 through 5
    nib ct index -f 2.1       # force re-index of chapter 2 scene 1

### Characters

`nib ct characters [range]` lists all characters from indexed data as sorted, deduplicated JSON. Default shows pov and present only.

    nib ct characters              # all characters across the manuscript
    nib ct characters 3            # characters in chapter 3
    nib ct characters 1-5          # characters in chapters 1 through 5
    nib ct characters 2.1          # characters in chapter 2 scene 1
    nib ct characters 1,3          # characters in chapters 1 and 3
    nib ct characters --all 3      # include mentioned characters

### Finding Scenes by Character

`nib ct chapters <character> [character...]` finds scenes where characters appear (pov or present) and returns dotted `chapter.scene` notation as JSON.

Default is AND -- scenes where all listed characters appear together. Use `--or` to get per-character results as a JSON object keyed by character slug.

    nib ct chapters lance              # scenes where lance is pov or present
    nib ct chapters lance bo           # scenes where both appear together
    nib ct chapters --or lance bo      # scenes for each, grouped by character

### Recap

`nib ct recap <range>` outputs a JSON recap of indexed chapter data. Useful for providing context to Claude when drafting or reviewing.

    nib ct recap 3                       # recap chapter 3
    nib ct recap 1-5                     # recap chapters 1 through 5
    nib ct recap 1,3,5                   # recap chapters 1, 3, and 5
    nib ct recap 1-5 -c lance-thurgood   # only scenes with lance
    nib ct recap 1-5 --detailed          # include all indexed data

### Reset

`nib ct reset` clears all indexed continuity data. Prompts for confirmation unless `--yes` is passed.

    nib ct reset            # prompt before clearing
    nib ct reset --yes      # skip confirmation

## Range Notation

Many commands accept a range argument. Supported formats:

| Format    | Meaning                                      |
| --------- | -------------------------------------------- |
| `3`       | All scenes in chapter 3                      |
| `3.2`     | Chapter 3, scene 2                           |
| `1-5`     | All scenes in chapters 1 through 5           |
| `1.1-3.2` | Chapter 1 scene 1 through chapter 3 scene 2  |
| `1,3,5`   | All scenes in chapters 1, 3, and 5           |
| `1.1,2.3` | Specific scenes by dotted notation           |
