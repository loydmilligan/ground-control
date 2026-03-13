# Project Discovery

Guide the user through defining their project vision and create the design document.

## Context

You are helping initialize a new project. This is step 2 of 5 in the new project workflow.

**Previous step**: Project directory created and adopted into Flight Deck.

**Your goal**: Create `docs/design.md` - a high-level design document.

## Input

The user may provide:
- A reference file from another project (check workflow.json context)
- Verbal description of what they want to build
- Problem they're trying to solve

## Process

### Phase 1: Understanding (Conversational)

Ask these questions one at a time, building on answers:

1. **What is this project?**
   "In one sentence, what will this project do?"

2. **What problem does it solve?**
   "What's painful or missing today that this fixes?"

3. **Who is it for?**
   "Who will use this? Just you, your team, or external users?"

4. **Does it build on existing work?**
   "Is there a reference file or existing project this extends?"
   - If yes, read and analyze it
   - Identify what's being reused vs what's new

### Phase 2: Clarification

Based on answers, ask follow-up questions:
- Scope boundaries (what's NOT included)
- Key constraints (tech, time, resources)
- Success criteria (how do we know it works)

### Phase 3: Synthesis

Summarize your understanding:
```
Here's what I understand:

**Project**: [name]
**Purpose**: [one sentence]
**Problem**: [what it solves]
**Audience**: [who uses it]
**Builds on**: [reference project, if any]

Key features:
1. ...
2. ...
3. ...

Is this accurate?
```

### Phase 4: Create Design Doc

Once confirmed, create `docs/design.md`:

```markdown
# [Project Name] - Design Document

## Overview
[One paragraph describing the project]

## Problem Statement
[What problem this solves and why it matters]

## Solution Approach
[High-level description of how it works]

## Major Features
1. **[Feature 1]**: [Description]
2. **[Feature 2]**: [Description]
3. **[Feature 3]**: [Description]

## Non-Goals (Out of Scope)
- [What this project will NOT do]

## Technical Constraints
- [Any known constraints]

## Success Criteria
- [How we know it's working]

## Reference
- Built on: [reference project, if any]
- Related: [related projects/docs]
```

## Output

1. Create `docs/` directory if needed
2. Write `docs/design.md`
3. Confirm: "Design document created. When ready, return to Flight Deck to continue to Features step."

## Session End

This session ends when `docs/design.md` is written and confirmed.
The user should exit and return to FD to trigger the next step.
