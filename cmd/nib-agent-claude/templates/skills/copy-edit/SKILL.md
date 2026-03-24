---
name: copy-edit
description: Line-level copy editing for grammar, punctuation, prose tightening, and style compliance. Use when the user wants a mechanical editing pass on a scene.
argument-hint: "[scene-slug]"
---

You are copy editing a scene from a novel manuscript. This is a line-level pass focused on mechanics, clarity, and prose quality -- not structural or developmental feedback. Don't comment on whether the scene works or what it accomplishes. Just make the prose clean.

Scenes are markdown files in `scenes/`, referenced by slug (filename without `.md`).

## Setup

1. **Read the scene:** `scenes/$ARGUMENTS.md`
2. **Load style guide:** Read `STYLE.md`.
3. **Load character profiles:** Read profiles from `characters/` for characters who appear, to inform vocabulary-appropriate word choice in dialogue.

## Editing Pass

Work through the scene in order. Group findings by category:

### 1. Grammar & Punctuation

- **Dialogue punctuation:** Commas and periods inside closing quotes. Comma before attribution ("said"), period when no attribution follows. Question marks and exclamation points replace commas/periods.
- **Comma splices:** Two independent clauses joined by a comma without a conjunction. Fix with a period, semicolon, em dash, or conjunction -- whichever fits the rhythm.
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

### 3. Banned Patterns

Flag any instance of these (from CLAUDE.md):
- "something [temperature] moved through [body part]" for emotions
- "it was like [thing] and not" for descriptions
- "it was [like a thing or action] in the way someone [trying hard to be the thing or do the action] would be"
- "it wasn't [a thing]. [he or she] took it as one"
- Three consecutive paragraphs starting with transition words (But, And, So, Then, Still, Yet, etc.)
- Three-part lists unless specifically needed

### 4. Prose Tightening

- **Dead weight:** very, really, just, quite, somewhat, rather, slightly, a bit -- cut unless they're doing character work in dialogue
- **Redundancy:** nodded his head, shrugged his shoulders, blinked his eyes, sat down, stood up
- **Weak verbs + adverbs:** walked slowly -> trudged, shuffled, dragged. Said loudly -> shouted, snapped. Look for the stronger verb.
- **Repetition:** Same point made twice in consecutive sentences. Same word used twice in a paragraph (unless intentional).
- **Filter words in close POV:** "he saw," "she noticed," "he felt," "she heard," "he realized" -- in close third, just show what they saw/noticed/felt/heard. The POV is already established.
- **Throat-clearing:** Sentences that set up what the next sentence actually says. Cut the setup.

### 5. Word Choice

- **Repeated words:** Same significant word (not articles/pronouns) appearing multiple times in close proximity
- **Generic nouns:** "the building," "the room," "the car" when a specific name or detail exists and has been established
- **Cliches:** Flag them, suggest alternatives

## Process

Edit the scene file directly. Do not show a preview or ask for permission -- make the corrections in place. The author can review the diff in git.

After editing, print a brief summary: how many issues by category (e.g. "3 comma fixes, 1 filter word, 2 tightening edits"). No need to list every change -- the diff speaks for itself.

## Rules

- Edit the file directly. Do not present changes and wait for approval.
- Don't touch intentional style. Fragments, sentence fragments used for rhythm, unconventional punctuation that's clearly a choice -- leave them unless they're actually broken.
- Don't embellish. Don't add metaphors, imagery, or personality that isn't already in the prose. Tighten, don't decorate.
- Don't restructure paragraphs or reorder sentences. That's developmental editing, not copy editing.
- If the scene is clean, say so. Don't invent problems.
