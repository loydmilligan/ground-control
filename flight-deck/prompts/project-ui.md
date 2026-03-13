# Project UI Spec

Define the user interface and user experience for the project.

## Context

You are helping define the UI/UX. This is step 4 of 5.

**Previous steps**:
- `docs/design.md` - project vision
- `docs/features.md` - features and user stories

**Your goal**: Create `docs/ui-spec.md` - UI/UX specification.

## Input

Read existing docs to understand:
- What the project does
- Who uses it
- What features exist

## Process

### Phase 1: UI Type

Determine the interface type:
- **Web App** - Browser-based SPA/MPA
- **CLI** - Command-line tool
- **TUI** - Terminal UI (like this project)
- **Mobile** - iOS/Android app
- **Desktop** - Native desktop app
- **API Only** - No UI, just endpoints

Ask: "What type of interface will this have?"

### Phase 2: Screen/View Inventory

For each major feature, identify:
- What screens/views are needed
- What's the user flow between them

"Walk me through what a user sees when they [do X feature]"

### Phase 3: Component Identification

For web/mobile apps:
- List reusable components
- Identify patterns (forms, lists, cards, etc.)

For CLI/TUI:
- List commands/subcommands
- Define output formats

### Phase 4: Create UI Spec

Write `docs/ui-spec.md`:

```markdown
# [Project Name] - UI Specification

## Interface Type
[Web App / CLI / TUI / etc.]

## Design Principles
- [Principle 1: e.g., "Simple over feature-rich"]
- [Principle 2: e.g., "Keyboard-first navigation"]

## Screen Inventory

| Screen | Purpose | Key Elements |
|--------|---------|--------------|
| Home | Landing/dashboard | [elements] |
| [Screen 2] | [purpose] | [elements] |

---

## Screen Details

### Home Screen

**Purpose**: [What user does here]

**Layout**:
```
┌─────────────────────────────┐
│  Header / Navigation        │
├─────────────────────────────┤
│                             │
│  Main Content Area          │
│                             │
├─────────────────────────────┤
│  Footer / Actions           │
└─────────────────────────────┘
```

**Elements**:
- [Element 1]: [description]
- [Element 2]: [description]

**User Actions**:
- [Action 1] → [Result]
- [Action 2] → [Result]

---

## Component Library

| Component | Usage | Variants |
|-----------|-------|----------|
| Button | Primary actions | primary, secondary, danger |
| Card | Content container | default, compact |

---

## User Flows

### Flow: [Primary User Journey]
1. User lands on [screen]
2. User clicks [element]
3. System shows [screen]
4. User completes [action]

---

## CLI Commands (if applicable)

| Command | Description | Example |
|---------|-------------|---------|
| `cmd action` | Does X | `cmd action --flag` |

---

## Responsive Behavior (if web/mobile)
- Mobile: [behavior]
- Tablet: [behavior]
- Desktop: [behavior]
```

## Output

1. Write `docs/ui-spec.md`
2. Summarize: "UI spec complete. X screens, Y components defined."
3. End: "Return to Flight Deck to continue to Scaffold step."

## Session End

This session ends when `docs/ui-spec.md` is complete.
