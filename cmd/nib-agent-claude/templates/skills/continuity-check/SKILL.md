---
name: continuity-check
description: Check a scene for narrative continuity errors against the established manuscript. Use when the user wants to verify a scene doesn't contradict established facts, timeline, or character knowledge.
argument-hint: "[scene-slug]"
---

You are checking a scene for continuity errors in a novel manuscript.

The manuscript is assembled by `nib`, which reads `book.yaml` to determine scene order and chapter structure. Scenes are markdown files in `scenes/`, referenced by slug (filename without `.md`). Scene order in the yaml is the canonical narrative sequence. Interludes don't consume chapter numbers.

## Setup

Gather this context silently:

1. **Read the scene:** `scenes/$ARGUMENTS.md`
2. **Find position:** Read `book.yaml` and locate `$ARGUMENTS`. Determine its chapter (call it N) and exact position in the narrative sequence.
3. **Load narrative context:** Run `nib ct recap 1-N --detailed` (where N is the chapter this scene belongs to). The `--detailed` flag includes facts, locations, dates, times, and all character appearances (including mentioned) -- you need these for continuity checking. This is your primary reference for what has been established in the story. If the recap shows unindexed scenes, note that -- there may be gaps in your continuity data. For long manuscripts, consider filtering by key characters: `nib ct recap 1-N --detailed -c character-slug` (repeatable, e.g. `-c lance -c ella`). This returns only scenes where those characters appear, which keeps context focused and manageable.
4. **Identify characters:** Note all named characters who appear in this scene (dialogue, action, or reference).
5. **Load character profiles:** Read profiles from `characters/` for each character present.
6. **Spot-check key scenes:** If the recap reveals facts or character states that are critical to your continuity analysis, read those specific scenes in full to verify details. The recap gives you summaries -- when precision matters (exact wording, physical descriptions, spatial details), go to the source.

## Checks

### 1. Character Knowledge

Does any character react to, reference, or act on information they haven't received yet? Walk the scene sequence: by this point in the story, what does each character know and how did they learn it? Flag any knowledge that can't be traced to a prior scene.

### 2. Physical Continuity

- **Body details:** Injuries, physical descriptions, clothing established in other scenes
- **Object continuity:** Items picked up, left behind, handed off, or referenced
- **Spatial logic:** Is the character in the right place? Can they physically get from where they were in the previous scene to where they are now?

### 3. Timeline Consistency

- Do relative time references ("yesterday," "last week," "three days ago") remain accurate against earlier scenes?
- Are time-of-day details consistent with the scene sequence?

### 4. Location Consistency

- Do physical descriptions of locations match how they've been described in other scenes?
- Are building names, room descriptions, and distances consistent?

### 5. Character State

- Is anyone present who should be absent (injured, dead, at work, in another location)?
- Are relationships and attitudes consistent with what's been established? (Has something happened between these characters that should affect their interaction here?)
- Do habits and mannerisms match the character profile?

## Output

Report issues by severity:

**HARD CONTRADICTION** -- Directly conflicts with an established fact.
- What's wrong
- Where it contradicts (cite the specific scene and quote the relevant passage)
- Suggested fix

**SOFT AMBIGUITY** -- Potentially inconsistent, depends on interpretation or timing.
- What might be wrong
- Why it's ambiguous
- Whether it needs fixing or can stand

**CLEAN** -- If no issues found, say "No continuity errors found." One line. Do not enumerate everything you checked or explain why each category passed. The absence of errors does not need justification.

## What is NOT a continuity error

- Characters disagreeing about facts, numbers, or events. That's dialogue, not a bug. If two characters remember something differently, that's characterization.
- Unreliable narration. A character's internal monologue may contradict objective reality on purpose.
- Thematic echoes or deliberate parallels between scenes.
- Information the reader doesn't have yet. Withholding is not an error.

If you find yourself writing "no fix needed" or "this is intentional," don't flag it. The check exists to catch actual mistakes, not to annotate craft choices.

Order by severity. Hard contradictions first. Include a final count: N hard contradictions, N soft ambiguities.
