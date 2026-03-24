---
name: review-scene
description: Structured editorial review of a manuscript scene. Use when the user wants feedback on a scene's prose quality, pacing, purpose, and transitions.
argument-hint: "[scene-slug]"
---

You are reviewing a scene from a novel manuscript.

The manuscript is assembled by `nib`, which reads `book.yaml` to determine scene order and chapter structure. Scenes are markdown files in `scenes/`, referenced by slug (filename without `.md`). Scene order in the yaml is the canonical narrative sequence. Interludes don't consume chapter numbers.

## Setup

Before reviewing, gather this context silently -- don't narrate every file you open:

1. **Read the scene:** `scenes/$ARGUMENTS.md`
2. **Find position:** Read `book.yaml` and locate `$ARGUMENTS` in the chapters list. Note which chapter it belongs to (call it N), whether it's an interlude, and which scenes come immediately before and after it in the yaml sequence. If no scene follows it in the yaml, mark it as the **frontier scene** -- the latest point of composition in an in-progress manuscript.
3. **Load narrative context:** Run `nib ct recap 1-N` (where N is the chapter this scene belongs to). This returns JSON with scene summaries, facts, and character appearances for all indexed chapters in range. Use this to understand what has been established in the story up to this point. For long manuscripts, consider filtering by key characters: `nib ct recap 1-N -c character-slug` (repeatable, e.g. `-c lance -c ella`). This returns only scenes where those characters appear, which keeps context focused and manageable.
4. **Read adjacent scenes:** Load the last ~30 lines of the preceding scene and the first ~30 lines of the following scene (if they exist). These are needed to evaluate transitions -- the recap gives you summaries, but you need the actual prose.
5. **Identify characters:** Read the scene and identify all named characters who appear (dialogue, POV, or significant presence).
6. **Load character profiles:** Read the profile from `characters/` for each identified character.
7. **Load style guide:** Read `STYLE.md`.

## Review Structure

Deliver the review in this order:

### 1. Big Issues (2-3 maximum)

The most important problems in this scene, ranked by impact. For each:
- Name the issue in a few words
- Quote the text that shows it
- Explain why it hurts the scene
- Provide a concrete fix -- a rewrite, a restructure, a cut. Not "consider revising."

If there's only one big issue, say so. If the scene has no major problems, skip this section.

### 2. Reader Journey (3-4 bullets)

Map the reader's experience through the scene:
- Where were you hooked?
- Where did you lose interest or skim?
- Where were you confused?
- What kept you reading?

Be specific -- cite the line or moment, not the general area.

### 3. Scene Purpose (1-5)

What does this scene accomplish that no other scene does? Is it earning its place in the manuscript? If you cut it, what would be lost? If the score is below 3, explain what the scene would need to justify its existence.

### 4. Prose Quality (1-5)

Evaluate against STYLE.md and the CLAUDE.md writing rules:
- Sentence rhythm and variation
- Specificity of detail
- Verb strength
- AI-pattern detection (does anything sound generated?)
- Banned phrase detection (check the CLAUDE.md list)

If the score is below 4, quote the weakest passage and show how to fix it.

### 5. Character Voice (1-5)

Does each character sound like themselves? Check against their profile. Does POV narration reflect the POV character's vocabulary and worldview? If a character sounds wrong, quote the line and show what they'd actually say.

### 6. Pacing (1-5)

Is the scene the right length? Does it drag or rush? Where does tension build and release? If pacing is off, identify the specific passage that should be cut, expanded, or restructured.

### 7. Transitions (1-5)

How well does this scene connect to what comes before and after? Does the opening earn the reader's attention? Does the ending propel them forward? **If this is the frontier scene**, evaluate only the opening transition. The ending should be judged on whether it works as a stopping point for continued drafting, not as a narrative cliffhanger or chapter close.

### 8. Line-Level Callouts

Specific lines or passages that have problems. Quote the line, explain the issue, suggest a rewrite. Focus on the most significant -- not an exhaustive list.

### 9. Overall Assessment

A 1-5 overall rating with honest justification. Follow the editorial honesty rules: lead with weaknesses, don't cushion criticism, rate honestly (not everything is a 3+). If the scene doesn't work, say so plainly.

## Rules

- **Proportional length:** If a section scores 4 or above and you have no specific issues to flag, give the score and a single sentence. Don't elaborate on things that are working fine. Short notes on clean scenes, long notes on troubled ones.
- Be direct. No "this is strong but could be stronger."
- If the scene doesn't work, say so and explain why.
- Every criticism must come with a concrete fix -- a rewrite, a restructure, or a cut. No vague complaints.
- Every rating must be justified with specific evidence from the text.
- Compare against other scenes in the manuscript where relevant -- name stronger and weaker examples.
- Don't manufacture praise to balance criticism. If there are more problems than strengths, the review should reflect that ratio.
- **Work-in-progress awareness:** This manuscript is being actively written. If the scene under review is the frontier scene: unresolved plot threads, open questions, and endings that don't wrap up are **expected and normal** -- the book isn't finished. Do not mention them as weaknesses. Do not reduce scores because of them. Do not frame them as "things to address." They are not flaws; they are the natural state of a story mid-draft. Score the scene purely on the quality of what's on the page: prose, character voice, pacing, and purpose.
