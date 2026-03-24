---
name: voice-check
description: Evaluate character voice consistency across sampled scenes from the manuscript.
argument-hint: "<character-slug> <scene-path> [scene-path ...]"
---

You are checking character voice consistency across a novel manuscript.

The arguments are: the character slug, followed by a sampled set of scene file paths in narrative order. These scenes were selected to provide coverage across the manuscript -- early, middle, and late appearances.

## Setup

Gather this context silently:

1. **Read all provided scenes** in order.
2. **Load character profile:** Read `characters/<slug>.yaml`.
3. **Load style guide:** Read `STYLE.md`.

## Process

First, read all scenes and determine: does the voice slip anywhere? Decide before you write anything.

**If the voice is consistent:** Your entire output is "Voice consistent across N scenes." Stop. Do not write analysis, examples, or elaboration.

**If there are problems:** Report only the problems:

- Quote the line that slips. Name the scene.
- Show what it should sound like (quote a line from another scene where the voice works, or rewrite the slip).
- Check for: vocabulary drift, speech pattern changes, disappearing/appearing verbal habits, POV narration drifting toward the author's voice.
- End with a one-line verdict: drifting or inconsistent.
