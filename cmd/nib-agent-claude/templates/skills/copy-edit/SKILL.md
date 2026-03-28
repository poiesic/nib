---
name: copy-edit
description: Mechanical proofreading for grammar, punctuation, typos, and formatting errors. Fixes only objectively wrong issues -- no taste decisions.
argument-hint: "[scene-slug]"
---

You are proofreading a scene from a novel manuscript. This is a purely mechanical pass -- fix things that are objectively wrong. Do not make taste decisions, tighten prose, improve word choice, or restructure sentences. Those belong in a critique pass, not here.

If the prose is correct, leave it alone. Do not invent problems.

Scenes are markdown files in `scenes/`, referenced by slug (filename without `.md`).

## Setup

1. **Read the scene:** `scenes/$ARGUMENTS.md`
2. **Load style guide:** Read `STYLE.md` for formatting rules.

## What to Fix

### 1. Grammar & Punctuation

- **Dialogue punctuation:** Commas and periods inside closing quotes. Comma before attribution ("said"), period when no attribution follows. Question marks and exclamation points replace commas/periods.
- **Comma splices:** Two independent clauses joined by a comma without a conjunction. Fix with a period, semicolon, em dash, or conjunction -- match the surrounding rhythm, don't redesign the sentence.
- **Missing commas:** After introductory phrases, around nonrestrictive clauses, in compound sentences before the conjunction.
- **Misplaced commas:** Before restrictive clauses, between subject and verb, after coordinating conjunctions.
- **Semicolons:** Only between independent clauses or in complex lists. Not as a fancy comma.
- **Apostrophes:** Contractions (it's vs. its, they're vs. their). Possessives (character's vs. characters').
- **Tense consistency:** The manuscript is past tense. Flag any unintentional present-tense slips.
- **Subject-verb agreement.**
- **Dangling modifiers:** Participial phrases that attach to the wrong noun.
- **Pronoun ambiguity:** When "he" or "she" could refer to more than one character in the scene, flag it. This is especially common in scenes with two characters of the same gender.

### 2. Formatting

- **Em dashes:** Must be `--` (double hyphen), not Unicode `---` or single `-`. This is a hard rule -- scrib/pandoc converts `--` during build.
- **Consistent spacing:** No double spaces. Clean paragraph breaks.

### 3. Typos & Errors

- **Misspelled words.**
- **Missing words** that make a sentence ungrammatical.
- **Duplicated words:** "the the," "was was," etc.
- **Redundant body parts:** "nodded his head," "shrugged her shoulders," "blinked his eyes" -- the body part is implied by the verb.

### 4. Banned Patterns

Flag any instance of these (from CLAUDE.md):
- "something [temperature] moved through [body part]" for emotions
- "it was like [thing] and not" for descriptions
- "it was [like a thing or action] in the way someone [trying hard to be the thing or do the action] would be"
- "it wasn't [a thing]. [he or she] took it as one"
- Three consecutive paragraphs starting with transition words (But, And, So, Then, Still, Yet, etc.)
- Three-part lists unless specifically needed

## Process

Edit the scene file directly. Do not show a preview or ask for permission -- make the corrections in place. The author can review the diff in git.

After editing, print a brief summary: how many issues by category (e.g. "3 comma fixes, 1 apostrophe, 2 typos"). No need to list every change -- the diff speaks for itself.

## What NOT to Do

This is the most important section. Violating these rules means the proof pass is broken.

- **Do not tighten prose.** Do not cut words or phrases because they could be shorter. "She suppressed the urge" and "She bit back the urge" are both correct -- choosing between them is a taste decision.
- **Do not remove filter words.** "She noticed," "he felt," "she heard" may or may not be the author's intent. Leave them.
- **Do not replace weak verbs with strong verbs.** "walked slowly" is not wrong. Changing it to "trudged" is a taste decision.
- **Do not cut sentences** you consider throat-clearing, dead weight, or redundant. That is developmental editing.
- **Do not restructure sentences** for rhythm, clarity, or impact. If a sentence is grammatically correct, leave it.
- **Do not improve word choice.** "Story" and "cover" may both work in context -- picking between them is taste.
- **Do not flag or remove clichés.** That is a critique concern.
- **Do not add metaphors, imagery, or personality** that isn't already in the prose.
- **Do not reorder sentences or restructure paragraphs.**
- **Do not touch intentional style.** Fragments for rhythm, unconventional punctuation that's clearly a choice -- leave them unless they're actually broken.
