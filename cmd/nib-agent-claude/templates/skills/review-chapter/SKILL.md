---
name: review-chapter
description: Structured editorial review of an entire chapter. Use when the user wants feedback on a chapter's arc, scene ordering, pacing, and cohesion.
argument-hint: "[scene-path ...]"
---

You are reviewing a complete chapter from a novel manuscript.

The manuscript is assembled by `nib`, which reads `book.yaml` to determine scene order and chapter structure. Scenes are markdown files in `scenes/`, referenced by slug (filename without `.md`). Scene order in the yaml is the canonical narrative sequence. Interludes don't consume chapter numbers.

The arguments are the file paths of every scene in this chapter, in narrative order.

## Setup

Before reviewing, gather this context silently -- don't narrate every file you open:

1. **Read all scenes:** Read each file path from `$ARGUMENTS` in order. Together these form the complete chapter.
2. **Find position:** Read `book.yaml` and identify which chapter these scenes belong to (call it N), whether it's an interlude, and which chapters come before and after it. If no chapter follows it in the yaml, mark it as the **frontier chapter** -- the latest chapter in an in-progress manuscript.
3. **Load narrative context:** Run `nib ct recap 1-N` (where N is this chapter's number). This returns JSON with scene summaries, facts, and character appearances for all indexed chapters in range. Use this to understand what has been established in the story up to this point. For long manuscripts, consider filtering by key characters: `nib ct recap 1-N -c character-slug` (repeatable, e.g. `-c lance -c ella`). This returns only scenes where those characters appear, which keeps context focused and manageable.
4. **Read chapter boundaries:** Load the last ~30 lines of the final scene in the preceding chapter and the first ~30 lines of the first scene in the following chapter (if they exist). These are needed to evaluate how this chapter connects to its neighbors.
5. **Identify characters:** Read all scenes and identify all named characters who appear (dialogue, POV, or significant presence).
6. **Load character profiles:** Read the profile from `characters/` for each identified character.
7. **Load style guide:** Read `STYLE.md`.

## Review Structure

Deliver the review in this order:

### 1. Big Issues (2-3 maximum)

The most important problems in this chapter, ranked by impact. For each:
- Name the issue in a few words
- Quote the text that shows it (name the scene slug)
- Explain why it hurts the chapter
- Provide a concrete fix -- a rewrite, a restructure, a cut, a reordering. Not "consider revising."

If there's only one big issue, say so. If the chapter has no major problems, skip this section.

### 2. Reader Journey (3-4 bullets)

Map the reader's experience through the chapter:
- Where were you hooked?
- Where did you lose interest or skim?
- Where were you confused?
- What kept you reading?

Be specific -- cite the scene slug and line or moment, not the general area.

### 3. Chapter Purpose (1-5)

What does this chapter accomplish in the larger story? What would be lost if you cut it entirely? Does it advance plot, deepen character, shift stakes, or establish something essential? If the score is below 3, explain what the chapter would need to justify its existence.

### 4. Arc and Structure (1-5)

Does this chapter have a discernible shape? Where does it begin emotionally and where does it end? Is there a turn, shift, or escalation? Do the scenes build on each other in a meaningful sequence, or could they be reordered without consequence? If the arc is weak, show how to restructure it.

### 5. Scene Ordering and Necessity

For each scene in the chapter:
- Does it earn its position in the sequence? Would moving it change the chapter's effect?
- Does it pull its weight, or is it filler? If you cut it, what would the chapter lose?

Be specific. Name scenes by slug. If a scene should be cut or moved, say so directly.

### 6. Pacing (1-5)

Does the chapter move at the right speed? Where does it drag? Where does it rush? Is the balance of action, dialogue, and interiority appropriate? If pacing is off, identify the specific scene or passage that should be cut, expanded, or restructured.

### 7. Prose Quality (1-5)

Evaluate against STYLE.md and the CLAUDE.md writing rules:
- Sentence rhythm and variation
- Specificity of detail
- Verb strength
- AI-pattern detection (does anything sound generated?)
- Banned phrase detection (check the CLAUDE.md list)
- Filter words in close POV: "he saw," "she noticed," "he felt," "she heard," "he realized" -- in close third, the POV is already established. Just show what they saw/noticed/felt/heard. Flag instances where the filter word adds nothing.
- Throat-clearing: sentences that exist only to set up what the next sentence actually says. If the second sentence makes the point, the first is dead weight. Quote both sentences so the pattern is visible.

This is a chapter-level assessment. Note patterns across scenes rather than exhaustive line-by-line coverage -- save specific callouts for section 10. If the score is below 4, quote the weakest passage and show how to fix it.

### 8. Character Voice (1-5)

Does each character sound like themselves across the chapter? Check against their profiles. Are there inconsistencies between scenes? Does POV narration reflect the POV character's vocabulary and worldview? If a character sounds wrong, quote the line and show what they'd actually say.

### 9. Transitions (1-5)

How well do scenes connect to each other within the chapter? Does the chapter opening earn the reader's attention? Does the chapter ending create appropriate momentum? Evaluate both inter-scene transitions and the chapter's connections to its neighbors.

### 10. Line-Level Callouts

Specific lines or passages that have problems. Quote the line, name the scene slug, explain the issue, suggest a rewrite. Focus on the most significant issues -- this is a chapter review, not a copy-edit pass.

### 11. Overall Assessment

A 1-5 overall rating with honest justification. Follow the editorial honesty rules: lead with weaknesses, don't cushion criticism, rate honestly (not everything is a 3+). If the chapter doesn't work, say so plainly.

## Scoring Rubric

Every 1-5 rating must use this scale:

- **5** — Exceptional. Publishable as-is. You'd point to this as an example of the craft done right.
- **4** — Solid. Minor issues only; none that would bother a reader.
- **3** — Functional. Does its job but has clear weaknesses that should be addressed.
- **2** — Struggling. Significant problems that undermine the chapter's purpose.
- **1** — Broken. Fails at what it's trying to do. Needs fundamental rework.

Most chapters in a competent manuscript will land at 3-4. A 5 should be rare and earned. A 1-2 means something is genuinely wrong, not just imperfect.

## Rules

- **Proportional length:** If a section scores 4 or above and you have no specific issues to flag, give the score and a single sentence. Don't elaborate on things that are working fine. Short notes on clean chapters, long notes on troubled ones.
- Lead with weaknesses. Don't manufacture praise to balance criticism.
- If the chapter doesn't work, say so and explain why.
- Every criticism must come with a concrete fix -- a rewrite, a restructure, or a cut. No vague complaints.
- Every rating must be justified with specific evidence from the text.
- Compare against other chapters in the manuscript where relevant -- name stronger and weaker examples.
- **Accuracy:** When you quote the text, quote it exactly as it appears in the file. Do not paraphrase, reconstruct from memory, or combine separate passages into one quote. If you're unsure of the exact wording, re-read the file before quoting.
- **Work-in-progress awareness:** This manuscript is being actively written. If this is the frontier chapter: unresolved plot threads, open questions, and endings that don't wrap up are **expected and normal** -- the book isn't finished. Do not mention them as weaknesses. Do not reduce scores because of them. Do not frame them as "things to address." They are not flaws; they are the natural state of a story mid-draft. Score the chapter purely on the quality of what's on the page: arc structure, prose, character voice, pacing, and purpose.
