# Project Features

Break down the project vision into detailed features and user stories.

## Context

You are helping define features for a new project. This is step 3 of 5.

**Previous step**: `docs/design.md` created with high-level vision.

**Your goal**: Create `docs/features.md` - detailed feature specs and user stories.

## Input

Read `docs/design.md` to understand:
- Project purpose
- Major features listed
- Success criteria

## Process

### Phase 1: Feature Breakdown

For each major feature in design.md:

1. **Clarify scope**
   "For [feature], what exactly should it do?"

2. **Identify user stories**
   "What are the key things a user will do with this?"

3. **Define acceptance criteria**
   "How do we know this feature is complete?"

### Phase 2: User Story Format

For each user story, use this format:
```
### US-001: [Title]

**As a** [user type]
**I want to** [action]
**So that** [benefit]

**Acceptance Criteria:**
- [ ] [Criterion 1]
- [ ] [Criterion 2]
- [ ] [Criterion 3]

**Notes:**
- [Any implementation notes]
```

### Phase 3: Prioritization

Ask user to prioritize:
- **MVP** (must have for v1)
- **Post-MVP** (nice to have, later)
- **Future** (maybe someday)

### Phase 4: Create Features Doc

Write `docs/features.md`:

```markdown
# [Project Name] - Features & User Stories

## Feature Overview

| ID | Feature | Priority | Stories |
|----|---------|----------|---------|
| F-001 | [Feature 1] | MVP | US-001, US-002 |
| F-002 | [Feature 2] | MVP | US-003 |
| F-003 | [Feature 3] | Post-MVP | US-004, US-005 |

---

## MVP Features

### F-001: [Feature Name]

[Feature description]

#### US-001: [User Story Title]

**As a** [user type]
**I want to** [action]
**So that** [benefit]

**Acceptance Criteria:**
- [ ] ...

---

## Post-MVP Features

### F-003: [Feature Name]
...

---

## Future Considerations

- [Ideas for future versions]
```

## Output

1. Write `docs/features.md`
2. Confirm feature count: "Created X features with Y user stories. Z are MVP."
3. End: "Features defined. Return to Flight Deck to continue to UI Spec step."

## Session End

This session ends when `docs/features.md` is complete.
