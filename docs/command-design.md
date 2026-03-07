# Ground Control Command Surface Design

## Overview

This document defines the complete command surface for Ground Control, focusing on the agentic orchestration pipeline that takes tasks from creation through verified completion.

## Philosophy

Ground Control treats task execution as a **quality-gated pipeline**, not fire-and-forget. Each task passes through specialized agents with feedback loops that catch issues early. The system is designed for:

1. **Verification at every stage** — No stage completes without passing its gate
2. **Iterative feedback** — Agents can send work back for revision
3. **Human escalation** — After N failures, humans decide next steps
4. **Parallel execution** — Independent tasks run concurrently with isolation
5. **Audit trail** — Every action logged for AI-Matt training

---

## Command Surface

### Core Commands

| Command | Purpose |
|---------|---------|
| `gc tasks` | List tasks with filtering |
| `gc dump` | Add brain dump entry |
| `gc create` | Interactive task creation |
| `gc process` | Process brain dump → tasks |
| `gc taskmaster` | Run Taskmaster review |
| `gc orc` | Orchestrate task execution |
| `gc complete` | Manual task completion |
| `gc standup` | Daily standup ritual |
| `gc self-learn` | Review issues and propose improvements |
| `gc tui` | Interactive TUI |

### Command Details

---

### `gc tasks`

List tasks with dynamic indexes and filtering.

```bash
gc tasks                    # Active tasks with indexes (hides completed)
gc tasks --all              # Include completed
gc tasks --state=blocked    # Filter by state
gc tasks --tag=cli          # Filter by tag
gc tasks --project=gc       # Filter by project
gc tasks --json             # JSON output for scripting
```

**Index Assignment**:
- Indexes are 1-based, assigned at display time
- Only non-completed tasks get indexes by default
- Indexes are ephemeral (recalculated each run)
- Use `--all` to include completed tasks in indexing

**Output Format**:
```
Tasks (3 active, 2 blocked, 1 pending)

  #  State    Imp   Title
  1  active   high  Implement authentication flow
  2  active   med   Add user settings page
  3  blocked  high  Deploy to staging
  4  pending  med   Write API documentation
  5  pending  low   Refactor logger module

Use 'gc orc 1 3' to orchestrate tasks or 'gc tui' for interactive mode.
```

---

### `gc dump`

Quick brain dump capture.

```bash
gc dump "implement rate limiting for API"
gc dump --urgent "fix production auth bug"
gc dump  # Opens $EDITOR for longer entry
```

**Behavior**:
- Appends to `data/brain-dump.json`
- Triggers background `gc process` if TUI is running
- Supports `--urgent` flag for priority hint

---

### `gc create`

Interactive task creation with guided prompts.

```bash
gc create                   # Full interactive mode
gc create --quick           # Minimal prompts (title + type)
gc create --from-template   # Use task template
```

**Interactive Flow**:
1. Title (required)
2. Description (required)
3. Type: coding | research | ai-planning | human-input | simple
4. Complexity: 1-5 scale
5. Importance: high | medium | low
6. Tags (comma-separated)
7. Project (optional, with autocomplete)
8. Blocked by (optional, shows indexed tasks)
9. Verification type + details

**Output**: Created task with ID, ready for `gc orc`

---

### `gc process`

Convert brain dump entries to tasks.

```bash
gc process                  # Process all unprocessed entries
gc process --dry-run        # Show what would be created
gc process --interactive    # Approve each conversion
```

**Processing Rules**:
1. Analyze entry content for task type signals
2. Extract urgency hints
3. Identify related existing tasks
4. Generate task with appropriate context
5. Log conversion in activity log

**Runs automatically** in background when TUI is active (every 60s).

---

### `gc taskmaster`

Run Taskmaster for state review and recommendations.

```bash
gc taskmaster               # Full review
gc taskmaster --quick       # Priority recommendations only
gc taskmaster --impact      # Post-change impact review
```

**Review Outputs**:
- Priority recommendations
- Blocked task analysis
- Suggested task merges/splits
- Dependency graph issues
- Stale task warnings

---

### `gc orc` (Orchestrate)

The core command. Execute tasks through the agentic pipeline.

```bash
gc orc 1                    # Single task by index
gc orc 1 3 5                # Multiple tasks by index
gc orc task_17412...        # By task ID
gc orc --all-pending        # All pending tasks
gc orc --continue           # Resume paused orchestration
```

**Flags**:
- `--dry-run`: Show execution plan without running
- `--max-parallel=N`: Override config max_parallel_agents
- `--skip-review`: Skip code review stage (not recommended)
- `--skip-test`: Skip test stage (not recommended)
- `--verbose`: Show agent output in real-time

---

## The Orchestration Pipeline

### Context Bundle (Created at Task Time)

Before a task enters the pipeline, Taskmaster (or a Context Manager agent) builds a **context bundle** when the task is created. This front-loads the context gathering so orchestration can start immediately.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     CONTEXT BUNDLE (at task creation)                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Built by: Taskmaster / Context Manager                                 │
│  Stored in: task.context_bundle (or separate file)                      │
│                                                                         │
│  Contents:                                                              │
│  ├── requirements.md        # Clear requirements from task + discussion │
│  ├── relevant_code/         # Code snippets that will be modified       │
│  │   ├── file1.go:45-120    # Specific line ranges                     │
│  │   └── file2.go:1-50                                                  │
│  ├── project_context.md     # From CLAUDE.md, README, etc.             │
│  ├── patterns.md            # Project patterns/conventions              │
│  ├── decisions.md           # Relevant past decisions from activity log │
│  ├── conversations.md       # Key conversation snippets that led here   │
│  └── test_hints.md          # Suggested test scenarios                  │
│                                                                         │
│  When created:                                                          │
│  - During gc create (interactive)                                       │
│  - During gc process (from brain dump)                                  │
│  - When Taskmaster assigns/reviews task                                 │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Context Manager Responsibilities**:
1. Analyze task description for affected files
2. Extract relevant code snippets (not entire files)
3. Pull project-specific documentation
4. Find related decisions from activity log
5. Include conversation context if task came from discussion
6. Generate test hints based on requirements

### Pipeline Stages (Simplified)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ORCHESTRATION PIPELINE (Simplified)                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Prerequisites: Context bundle already built at task creation           │
│                                                                         │
│  ┌──────────────┐                                                       │
│  │   SANITY     │  Verify task ready: deps met, context complete       │
│  │   CHECK      │  Gate: Can we start? If not → blocked                │
│  └──────┬───────┘                                                       │
│         │ pass                                                          │
│         ▼                                                               │
│  ┌──────────────┐                                                       │
│  │    CODER     │  Implement using context bundle                      │
│  │              │  Output: code changes + implementation notes          │
│  └──────┬───────┘                                                       │
│         │                                                               │
│         ▼                                                               │
│  ┌──────────────┐                                                       │
│  │    CODE      │  Single review pass                                  │
│  │   REVIEWER   │  Gate: approved | needs_revision | escalate          │
│  └──────┬───────┘                                                       │
│         │ approved (or escalate after 3 revision rounds)               │
│         ▼                                                               │
│  ┌──────────────┐                                                       │
│  │   TESTER     │  Run tests, write new tests if needed                │
│  │              │  Gate: tests pass | fix needed | escalate            │
│  └──────┬───────┘                                                       │
│         │ tests pass (or escalate after 3 fix rounds)                  │
│         ▼                                                               │
│  ┌──────────────┐                                                       │
│  │    REPO      │  Combined: update docs + commit + deploy             │
│  │   UPDATE     │  Gate: clean commit, hooks pass, deployed (if req)   │
│  └──────┬───────┘                                                       │
│         │                                                               │
│         ▼                                                               │
│     TASK COMPLETED                                                      │
│         │                                                               │
│         ▼ (after all tasks in session complete)                         │
│  ┌──────────────┐                                                       │
│  │   SESSION    │  Taskmaster reviews entire work session              │
│  │   REVIEW     │  Validates issues, identifies patterns               │
│  └──────┬───────┘  Logs to learning-log.json                           │
│         │                                                               │
│         ▼                                                               │
│     SESSION COMPLETE                                                    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Stage Definitions

#### 1. Sanity Check

**Purpose**: Verify task is ready for execution.

**Checks**:
- Requirements are clear and unambiguous
- All blocked_by tasks are completed
- Required files/dependencies exist
- No conflicting changes in progress (parallel execution)

**Gate**: All checks pass, or escalate to human.

**Failure Handling**: If sanity check fails, task goes to `blocked` state with clear reason.

#### 2. Coder

**Purpose**: Implement the task requirements.

**Inputs**:
- Context bundle (built at task creation time):
  - `requirements.md`
  - `relevant_code/` snippets
  - `project_context.md`
  - `patterns.md`
  - `decisions.md`
  - `conversations.md` (if applicable)
  - `test_hints.md`

**Process**:
1. Read context bundle
2. Plan approach based on patterns and decisions
3. Implement changes
4. Run basic validation (compile, lint)
5. Self-review against requirements

**Outputs**:
- Code changes (tracked in git worktree)
- `implementation_notes.md`: What was done and why

**Handoff**: Changes ready for Code Reviewer.

#### 3. Code Reviewer

**Purpose**: Single review pass for quality and standards.

**Inputs**:
- Code changes from Coder
- Context bundle (especially patterns.md, decisions.md)
- `implementation_notes.md`

**Review Criteria**:
- Follows project conventions
- No obvious bugs or edge cases missed
- Appropriate error handling
- No security issues
- Clean, readable code
- Matches requirements

**Outputs**:
- `review_feedback.md`: Specific, actionable feedback
- Decision: `approved` | `needs_revision` | `escalate`

**Feedback Loop** (if `needs_revision`):
- Send feedback to Coder for revision
- Max 3 revision rounds total
- After 3 failures: Escalate to human

#### 4. Tester

**Purpose**: Verify implementation works correctly.

**Inputs**:
- Code changes
- `test_hints.md` from context bundle
- Existing test suite

**Process**:
1. Run existing tests
2. Write new tests for changed functionality (using test hints)
3. Run full test suite
4. Check coverage (if configured)

**Outputs**:
- New test files (if any)
- `test_results.md`: Pass/fail summary
- Decision: `tests_pass` | `needs_fix` | `escalate`

**Feedback Loop** (if `needs_fix`):
- Send failures to Coder for fixes
- Max 3 fix rounds total
- After 3 failures: Escalate to human

#### 5. Repo Update (Combined Stage)

**Purpose**: Update docs, commit, and deploy in one pass.

**Inputs**:
- All code changes from previous stages
- `implementation_notes.md`
- Commit message guidelines
- Deployment config (if applicable)

**Process**:
1. **Docs**: Update affected documentation (inline comments, README, CHANGELOG)
2. **Commit**: Stage files, generate commit message, run hooks, create commit
3. **Deploy**: If `task.deploy_target` set, push and trigger deployment

**Outputs**:
- Documentation changes
- Git commit with hash
- Deployment status (if applicable)

**Gate**: Commit succeeds, hooks pass, deployment successful (if required).

---

### Context Bundle Creation

The context bundle is built **before** orchestration, during task creation. This ensures agents have everything they need upfront.

#### When Context is Built

| Trigger | Who Builds Context | Notes |
|---------|-------------------|-------|
| `gc create` | Context Manager | Interactive, can ask clarifying questions |
| `gc process` | Taskmaster → Context Manager | Taskmaster creates task, then calls Context Manager |
| Task assignment | Taskmaster → Context Manager | Reviews task, enriches context |
| Task review | Taskmaster | May add decisions/conversations to existing bundle |

#### Taskmaster → Context Manager Flow

**Important**: Taskmaster always invokes Context Manager as part of task creation:

```
┌────────────────────────────────────────────────────────────────┐
│                 TASKMASTER TASK CREATION FLOW                  │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  1. Taskmaster receives task request (from process, create)    │
│                                                                │
│  2. Taskmaster creates task skeleton:                          │
│     - ID, title, description                                   │
│     - Type, complexity, importance                             │
│     - Requirements, constraints                                │
│     - Verification criteria                                    │
│                                                                │
│  3. Taskmaster CALLS Context Manager:                          │
│     "Build context bundle for task_123"                        │
│                                                                │
│  4. Context Manager builds bundle:                             │
│     - Finds relevant code snippets                             │
│     - Extracts project patterns                                │
│     - Pulls related decisions from activity log                │
│     - Captures conversation context (if any)                   │
│     - Generates test hints                                     │
│                                                                │
│  5. Context Manager returns bundle → attached to task          │
│                                                                │
│  6. Task is now ready for orchestration                        │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

This ensures every task has rich context before it ever enters the pipeline.

#### Context Bundle Contents

```
task_<id>/context/
├── requirements.md       # What needs to be done (from task + discussion)
├── relevant_code/        # Code snippets (NOT full files)
│   ├── auth.go:45-120   # Just the relevant function
│   └── types.go:1-30    # Just the relevant types
├── project_context.md    # Extracted from CLAUDE.md, README
├── patterns.md           # How this project does things
├── decisions.md          # Past decisions that affect this task
├── conversations.md      # Key discussion snippets (if from chat)
└── test_hints.md         # Suggested test scenarios
```

#### Building decisions.md

The Context Manager queries `activity-log.json` for relevant decisions:

```json
{
  "type": "decision_made",
  "summary": "Use Go + Bubble Tea for CLI",
  "reasoning": "Better TUI support, compiled binary, good Claude Code integration",
  "decision_factors": ["performance", "developer experience", "ecosystem"]
}
```

These get summarized into `decisions.md` so the Coder understands **why** things are the way they are.

#### Building conversations.md

When a task comes from a discussion (brain dump that evolved through conversation), key exchanges are captured:

```markdown
## Relevant Conversation

**User**: "I want rate limiting but not too aggressive"
**Response**: "Suggested 100 req/min for authenticated, 20 for anonymous"
**User**: "Yes, that sounds right"

→ Requirement: 100 req/min auth, 20 req/min anon
```

This preserves intent that might not be explicit in the task description.

---

### Feedback Loop Mechanics

```
┌─────────────────────────────────────────────────────┐
│              FEEDBACK LOOPS (max 3 rounds)          │
├─────────────────────────────────────────────────────┤
│                                                     │
│  REVIEW LOOP:                                       │
│  ┌────────┐  feedback  ┌──────────┐                │
│  │ Coder  │◄───────────│ Reviewer │                │
│  └────────┘            └──────────┘                │
│       │                     ▲                       │
│       └─────────────────────┘                       │
│             revision                                │
│                                                     │
│  TEST LOOP:                                         │
│  ┌────────┐  failures  ┌──────────┐                │
│  │ Coder  │◄───────────│  Tester  │                │
│  └────────┘            └──────────┘                │
│       │                     ▲                       │
│       └─────────────────────┘                       │
│              fixes                                  │
│                                                     │
│  After 3 rounds → ESCALATION                        │
│  ┌─────────────────────────────────┐               │
│  │  Human Decision Required        │               │
│  │  • Provide guidance, retry      │               │
│  │  • Take over manually           │               │
│  │  • Descope task                 │               │
│  │  • Abandon task                 │               │
│  └─────────────────────────────────┘               │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**Escalation Details**:
- Task state changes to `waiting`
- Human receives summary of all attempts
- Full feedback history preserved
- Human decision logged for AI-Matt training
- `gc orc --continue` resumes after human input

---

### Parallel Execution

When multiple tasks are orchestrated simultaneously:

```
┌─────────────────────────────────────────────────────┐
│           PARALLEL EXECUTION MODEL                  │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Task 1 ──► [worktree-1] ──► Pipeline ──► Merge    │
│                                              │      │
│  Task 3 ──► [worktree-3] ──► Pipeline ──► Merge    │
│                                              │      │
│  Task 5 ──► [worktree-5] ──► Pipeline ──► Merge    │
│                                              │      │
│                                              ▼      │
│                                         [main]     │
│                                                     │
│  Conflict Detection:                                │
│  - Same file modified by multiple tasks             │
│  - Semantic conflicts (same function changed)       │
│  - Merge conflicts at integration                   │
│                                                     │
│  Resolution:                                        │
│  - Pause conflicting tasks                          │
│  - Human decides merge order                        │
│  - Or: serialize conflicting tasks                  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**Git Worktree Strategy**:
1. Create worktree per task: `git worktree add .worktrees/task_123 -b task/task_123`
2. Pipeline runs in isolated worktree
3. On completion, merge back to main
4. Clean up worktree

**Conflict Handling**:
- Pre-execution: Analyze `files_to_modify.txt` for overlap
- During execution: Monitor for conflicts
- Post-execution: Merge with conflict detection
- On conflict: Pause, notify human

**Config**:
```json
{
  "max_parallel_agents": 3,
  "conflict_strategy": "pause_and_ask",  // or "serialize"
  "worktree_base": ".worktrees"
}
```

---

### TUI Integration

The TUI provides interactive access to all commands.

#### Views

1. **Task List** (default)
   - Arrow keys: Navigate
   - Space: Toggle select (multi-select mode)
   - Enter: View details
   - `o`: Orchestrate selected
   - `c`: Create new task
   - `s`: Change state
   - `f`: Filter menu
   - `?`: Help

2. **Task Detail**
   - Full task information
   - Outputs and verification status
   - Activity history
   - `o`: Orchestrate this task
   - `e`: Edit task
   - `b`: Back to list

3. **Orchestration View**
   - Live pipeline progress
   - Current stage indicator
   - Agent output stream
   - Feedback loop counter
   - `p`: Pause orchestration
   - `Esc`: Background (continue in bg)

4. **Brain Dump Processing**
   - Shows unprocessed entries
   - Live conversion preview
   - Approve/reject each

#### Multi-Select Mode

```
┌─ Tasks ──────────────────────────────────────────────┐
│                                                      │
│  [x] 1  Implement user authentication     high       │
│  [ ] 2  Add settings page                 medium     │
│  [x] 3  Create API rate limiter           high       │
│  [ ] 4  Write documentation               low        │
│  [x] 5  Refactor database layer           medium     │
│                                                      │
│  ─────────────────────────────────────────────────   │
│  3 selected │ Press 'o' to orchestrate               │
│                                                      │
└──────────────────────────────────────────────────────┘
```

---

### Configuration

Config lives in `data/config.json`:

```json
{
  "max_parallel_agents": 3,
  "feedback_loop_max_iterations": 3,
  "auto_process_interval_seconds": 60,
  "conflict_strategy": "pause_and_ask",
  "worktree_base": ".worktrees",

  "pipeline": {
    "skip_review": false,
    "skip_test": false,
    "skip_docs": false,
    "require_human_approval_before_commit": false
  },

  "claude_code": {
    "model": "claude-sonnet-4-20250514",
    "timeout_minutes": 30
  },

  "defaults": {
    "verification_type": "test_pass",
    "after_completion": "taskmaster_review",
    "autonomy_level": "checkpoints"
  }
}
```

**Project-Level Overrides**:

In `data/projects.json`, each project can override:

```json
{
  "id": "proj_gc",
  "name": "Ground Control",
  "config_overrides": {
    "pipeline": {
      "require_human_approval_before_commit": true
    }
  }
}
```

---

## User Journeys

### Journey 1: Quick Idea → Execution

```
1. gc dump "add rate limiting to API"
   └─► Entry added to brain-dump.json

2. gc process
   └─► Task created: "Implement API rate limiting"
   └─► Type: coding, Complexity: 3, Importance: medium

3. gc orc 1
   └─► Context gathered
   └─► Coder implements
   └─► Reviewer approves (1 round of feedback)
   └─► Tests pass
   └─► Docs updated
   └─► Committed

4. Task complete, taskmaster review triggered
```

### Journey 2: Complex Feature via TUI

```
1. gc tui
   └─► See all tasks

2. Space to select tasks 1, 3, 5
   └─► All related to "authentication feature"

3. Press 'o' to orchestrate
   └─► Dependency analysis: 1 blocks 3 blocks 5
   └─► Tasks serialize automatically

4. Watch pipeline progress
   └─► Task 1: Passes all stages
   └─► Task 3: Fails review (2 rounds), passes on 3rd
   └─► Task 5: Completes

5. All three committed, feature complete
```

### Journey 3: Escalation Flow

```
1. gc orc 1
   └─► Context gathered
   └─► Coder implements

2. Review feedback loop:
   └─► Round 1: "Missing error handling" → Coder fixes
   └─► Round 2: "Edge case X not handled" → Coder fixes
   └─► Round 3: "Still missing Y" → Coder fixes
   └─► Round 4: ESCALATION

3. Task state → waiting
   └─► Human notified with full history

4. Human reviews, provides guidance:
   └─► "Actually, Y is out of scope for this task"

5. gc orc --continue
   └─► Reviewer re-reviews with human context
   └─► Approved, pipeline continues
```

---

## Implementation Phases

### Phase 1: Core Commands (Done)
- [x] `gc tasks` (basic)
- [x] `gc dump`
- [x] `gc complete`
- [x] `gc standup`
- [x] `gc tui` (basic)

### Phase 2: Task Creation & Context
- [ ] `gc tasks` (indexes, filtering, hide completed)
- [ ] `gc create` (interactive task creation)
- [ ] `gc process` (brain dump → tasks)
- [ ] Context Manager agent
- [ ] Context bundle creation at task time
- [ ] Activity log integration for decisions.md

### Phase 3: Basic Orchestration
- [ ] `gc orc` (single task)
- [ ] Sanity Check stage
- [ ] Coder stage
- [ ] Code Reviewer stage (single pass)
- [ ] Feedback loop (review → coder)
- [ ] Escalation handling

### Phase 4: Testing & Completion
- [ ] Tester stage
- [ ] Test feedback loop (test → coder)
- [ ] Repo Update stage (docs + commit + deploy)
- [ ] Full activity logging
- [ ] Agent issue reporting in outputs
- [ ] Session tracking (`data/sessions/`)
- [ ] Taskmaster session review step

### Phase 5: Parallel Execution
- [ ] Git worktree management
- [ ] `gc orc 1 3 5` (multiple tasks)
- [ ] Conflict detection
- [ ] Merge handling

### Phase 6: TUI Enhancements
- [ ] Multi-select mode (spacebar)
- [ ] Orchestration progress view
- [ ] Brain dump processing view
- [ ] Background auto-process (60s interval)

### Phase 7: Self-Learning & Polish
- [ ] `gc self-learn` command
- [ ] Learning log analysis
- [ ] Change proposal generation
- [ ] Improvement application tracking
- [ ] Config system (`data/config.json`)
- [ ] Project-level overrides
- [ ] AI-Matt training data refinement

---

---

## Self-Learning System

Ground Control learns from its mistakes. Every work session captures issues, and `gc self-learn` reviews them to propose concrete improvements.

### Core Concept

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         LEARNING LOOP                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   gc orc 1 3 5                                                          │
│        │                                                                │
│        ▼                                                                │
│   ┌─────────────────────────────────────────────────────────┐          │
│   │  WORK SESSION BEGINS                                    │          │
│   │                                                         │          │
│   │  Each agent logs issues to session_issues.json:         │          │
│   │  - Only REAL problems (not filler)                      │          │
│   │  - Specific and actionable                              │          │
│   │  - Tagged with stage and task                           │          │
│   └─────────────────────────────────────────────────────────┘          │
│        │                                                                │
│        ▼                                                                │
│   ┌─────────────────────────────────────────────────────────┐          │
│   │  WORK SESSION ENDS                                      │          │
│   │                                                         │          │
│   │  Taskmaster performs Session Review:                    │          │
│   │  - Reviews all agent-reported issues                    │          │
│   │  - Validates issues are real (not box-filling)          │          │
│   │  - Adds systemic issues observed across session         │          │
│   │  - Logs to data/learning-log.json                       │          │
│   └─────────────────────────────────────────────────────────┘          │
│        │                                                                │
│        ▼                                                                │
│   gc self-learn                                                         │
│        │                                                                │
│        ▼                                                                │
│   ┌─────────────────────────────────────────────────────────┐          │
│   │  LEARNING REVIEW                                        │          │
│   │                                                         │          │
│   │  Taskmaster reviews accumulated issues and proposes:    │          │
│   │  - Process changes                                      │          │
│   │  - Prompt improvements                                  │          │
│   │  - Context bundle enhancements                          │          │
│   │  - New validation steps                                 │          │
│   └─────────────────────────────────────────────────────────┘          │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Work Session Definition

A **work session** begins when `gc orc` is invoked and ends when:
- All tasks complete (success or escalation)
- User cancels with Ctrl+C
- Fatal error occurs

Session metadata stored in `data/sessions/session_<timestamp>.json`.

### Agent Issue Reporting

Every agent output includes an `issues` field. **CRITICAL**: This field should be empty unless there was an actual problem.

#### Agent Output Schema

```typescript
interface AgentOutput {
  stage: string;                    // "coder", "reviewer", "tester", etc.
  task_id: string;
  status: "success" | "needs_revision" | "escalate";

  // Main outputs (stage-specific)
  // ...

  // Issue reporting - ONLY populate if there was a real problem
  issues: Issue[];
}

interface Issue {
  severity: "minor" | "moderate" | "significant";
  category: IssueCategory;
  description: string;              // What happened
  impact: string;                   // How it affected the work
  suggestion: string | null;        // Agent's idea for improvement (optional)
}

type IssueCategory =
  | "review_failure"           // Code didn't pass review
  | "test_failure"             // Tests failed
  | "handoff_confusion"        // Unclear what previous stage provided
  | "missing_context"          // Context bundle was incomplete
  | "process_deviation"        // Didn't follow established process
  | "unclear_requirements"     // Requirements were ambiguous
  | "tooling_issue"            // Problem with tools/environment
  | "other";
```

#### Agent Instructions for Issue Reporting

Include this in every agent prompt:

```markdown
## Issue Reporting

Your output includes an `issues` array. Follow these rules STRICTLY:

1. **ONLY report actual problems** - If the work flowed smoothly, leave `issues` empty: `[]`
2. **Do NOT fill this field just to fill it** - Empty is correct when there are no issues
3. **Be specific** - Vague issues like "could be better" are not helpful
4. **Focus on systemic problems** - Things that could happen again, not one-off mistakes

### What IS an issue:
- Code rejected by reviewer for missing edge case that wasn't in requirements
- Test failed because test hints didn't cover an important scenario
- Had to guess about project conventions because patterns.md was incomplete
- Unclear which files from previous stage were the "real" output
- Requirements said "handle errors" but didn't specify which errors

### What is NOT an issue:
- "I completed the task successfully" (that's not an issue)
- "The code could be more elegant" (subjective, not actionable)
- "This was a complex task" (difficulty is not an issue)
- "I had to read multiple files" (normal work, not an issue)

### Examples of good issue reports:

✅ GOOD:
{
  "severity": "moderate",
  "category": "missing_context",
  "description": "patterns.md did not specify error handling conventions",
  "impact": "Reviewer rejected first submission for using exceptions instead of error returns",
  "suggestion": "Add error handling section to patterns.md"
}

✅ GOOD:
{
  "severity": "minor",
  "category": "unclear_requirements",
  "description": "Requirement said 'validate input' but didn't specify validation rules",
  "impact": "Had to make assumptions, reviewer asked for stricter validation",
  "suggestion": "Requirements should specify validation rules explicitly"
}

✅ GOOD:
{
  "severity": "significant",
  "category": "test_failure",
  "description": "test_hints.md suggested testing happy path only, missed auth edge case",
  "impact": "Tests passed but reviewer caught auth bypass vulnerability",
  "suggestion": "Context Manager should include security-focused test hints"
}

❌ BAD (do not report these):
{
  "description": "Task completed successfully"  // Not an issue
}

❌ BAD:
{
  "description": "Code could be cleaner"  // Vague, subjective
}

❌ BAD:
{
  "description": "Had to think carefully about the implementation"  // Normal work
}
```

### Taskmaster Session Review

At the end of each work session, Taskmaster performs a deep review:

```markdown
## Session Review Instructions

You are reviewing work session {session_id} which ran `gc orc` on tasks: {task_ids}.

### Step 1: Collect Agent-Reported Issues

Review issues reported by each agent. For each issue, validate:
- Is this a REAL problem or just box-filling?
- Is it specific enough to act on?
- Does it represent a pattern or one-off?

Flag and REMOVE issues that are:
- Generic ("task was complex")
- Subjective without impact ("could be better")
- Normal work described as problems

### Step 2: Identify Systemic Issues

Look across the entire session for patterns the agents might not have reported:

**Review Loops**
- How many review iterations occurred per task?
- If >1: What caused the extra rounds?
- Were the same types of feedback given repeatedly?

**Test Failures**
- Did any tests fail initially?
- Were failures due to missing test hints or implementation bugs?
- Did test fixes require multiple rounds?

**Handoff Problems**
- Did any agent express confusion about inputs?
- Were there mismatches between what one stage produced and next expected?

**Process Deviations**
- Did agents skip steps?
- Did agents add unnecessary steps?
- Were there inconsistencies in how agents followed instructions?

**Context Gaps**
- Was any agent missing information it needed?
- Did agents have to search for information that should have been provided?

### Step 3: Log Validated Issues

Write validated issues to data/learning-log.json with:
- Session ID
- Task IDs involved
- Issue details
- Your assessment of root cause
- Whether this is a recurring pattern (check previous sessions)

### Step 4: Summary

Provide brief summary:
- Total issues found: X
- Removed as non-issues: Y
- New systemic issues identified: Z
- Recurring patterns: [list]
```

### Learning Log Schema

`data/learning-log.json`:

```typescript
interface LearningLog {
  entries: LearningEntry[];
}

interface LearningEntry {
  id: string;
  session_id: string;
  task_ids: string[];
  timestamp: string;

  issues: ValidatedIssue[];

  taskmaster_notes: string;         // Taskmaster's analysis
  recurring_pattern: boolean;       // Seen in previous sessions?
  related_entries: string[];        // IDs of similar past issues
}

interface ValidatedIssue {
  original_issue: Issue;            // As reported by agent
  validated: boolean;               // Taskmaster confirmed real
  validation_notes: string;         // Why validated/rejected
  root_cause: string;               // Taskmaster's assessment
  affected_component: string;       // What needs to change
}
```

### `gc self-learn` Command

```bash
gc self-learn                    # Review all unaddressed issues
gc self-learn --session <id>     # Review specific session
gc self-learn --dry-run          # Show proposals without applying
gc self-learn --category review_failure  # Focus on specific category
```

#### Self-Learn Flow

```markdown
## Self-Learn Instructions

You are reviewing the learning log to propose improvements.

### Step 1: Analyze Issue Patterns

Group issues by:
- Category (review_failure, test_failure, etc.)
- Affected component (context bundle, agent prompts, process)
- Frequency (one-off vs recurring)

Prioritize:
1. Recurring issues (seen 2+ times)
2. Significant severity issues
3. Issues with clear root causes

### Step 2: Propose Concrete Changes

For each addressable issue pattern, propose a SPECIFIC change:

**Format:**
```
ISSUE PATTERN: [description of recurring problem]
OCCURRENCES: [count] times across [sessions]
ROOT CAUSE: [why this keeps happening]

PROPOSED CHANGE:
- Component: [what to modify]
- Change: [exact modification]
- Rationale: [why this fixes the root cause]
```

### Examples of Good Proposals:

**Example 1: Missing Error Handling Conventions**

ISSUE PATTERN: Coder uses exceptions, reviewer wants error returns (3 occurrences)
OCCURRENCES: 3 times across sessions S1, S3, S5
ROOT CAUSE: patterns.md doesn't specify error handling style

PROPOSED CHANGE:
- Component: Context Manager → patterns.md generation
- Change: Add to patterns.md template:
  ```
  ## Error Handling
  This project uses [return values | exceptions] for error handling.
  - Functions return (result, error) tuples
  - Never panic except for programmer errors
  - Wrap errors with context: fmt.Errorf("doing X: %w", err)
  ```
- Rationale: Explicit convention prevents review round-trips

---

**Example 2: Test Hints Missing Security Cases**

ISSUE PATTERN: Security issues caught in review, not tests (2 occurrences)
OCCURRENCES: 2 times across sessions S2, S4
ROOT CAUSE: test_hints.md focuses on functional tests only

PROPOSED CHANGE:
- Component: Context Manager → test_hints.md generation
- Change: Add security section to test_hints template:
  ```
  ## Security Test Hints
  Consider testing:
  - Authentication bypass attempts
  - Input validation edge cases (empty, null, too long, special chars)
  - Authorization (can user A access user B's data?)
  - Rate limiting if applicable
  ```
- Rationale: Proactive security hints catch issues before review

---

**Example 3: Handoff Confusion Between Coder and Reviewer**

ISSUE PATTERN: Reviewer unsure which files are "the changes" vs existing (2 occurrences)
OCCURRENCES: 2 times across sessions S1, S2
ROOT CAUSE: Coder output doesn't clearly list modified files

PROPOSED CHANGE:
- Component: Coder agent prompt
- Change: Add to Coder output requirements:
  ```
  Your output MUST include:
  ## Files Modified
  - path/to/file1.go (new file)
  - path/to/file2.go (modified lines 45-60, 120-135)
  - path/to/file3.go (deleted)

  ## Summary of Changes
  [Brief description of what changed and why]
  ```
- Rationale: Explicit file list eliminates reviewer confusion

---

### Examples of Bad Proposals (avoid these):

❌ BAD: "Improve the prompts" (vague, not actionable)
❌ BAD: "Be more careful" (not a systemic fix)
❌ BAD: "Add more context" (what context? where?)

### Step 3: Present for Approval

List all proposed changes for human review:
- What will change
- Why it will help
- Any risks or tradeoffs

Human approves/rejects/modifies each proposal.

### Step 4: Apply Approved Changes

For each approved change:
1. Make the modification (prompt, template, process doc)
2. Log the change in activity-log.json
3. Mark related issues as "addressed" in learning-log.json
4. Note which sessions/issues led to this change (traceability)
```

### Learning Data Files

| File | Purpose |
|------|---------|
| `data/sessions/session_<ts>.json` | Individual session data |
| `data/learning-log.json` | Validated issues with analysis |
| `data/improvements-applied.json` | History of changes made via self-learn |

### Implementation Phases for Self-Learn

Add to existing phases:

**Phase 4 Addition:**
- [ ] Agent issue reporting in outputs
- [ ] Session tracking (`data/sessions/`)
- [ ] Taskmaster session review step

**Phase 7 Addition:**
- [ ] `gc self-learn` command
- [ ] Learning log analysis
- [ ] Change proposal generation
- [ ] Improvement application

---

## Agent Additions

The orchestration pipeline requires new agent definitions:

| Agent | Role | When Called |
|-------|------|-------------|
| `context-manager` | Builds context bundles | Task creation, assignment |
| `coder` | Implements code changes | Orchestration - Coder stage |
| `reviewer` | Reviews code quality | Orchestration - Review stage |
| `tester` | Runs and writes tests | Orchestration - Test stage |
| `repo-maintainer` | Docs, commit, deploy | Orchestration - Repo Update stage |

### Task Schema Addition

Add `context_bundle` field to task:

```typescript
interface Task {
  // ... existing fields ...

  // Context bundle (built at creation time)
  context_bundle: {
    built_at: string;                    // When context was gathered
    built_by: string;                    // "context-manager" usually
    files: {
      requirements: string;              // Path to requirements.md
      relevant_code: string[];           // Paths to code snippets
      project_context: string;           // Path to project_context.md
      patterns: string;                  // Path to patterns.md
      decisions: string;                 // Path to decisions.md
      conversations: string | null;      // Path to conversations.md (if any)
      test_hints: string;                // Path to test_hints.md
    };
  } | null;
}
```

---

## Open Questions (For Future)

1. **Agent Communication**: How do parallel agents share discovered information?
2. **Caching**: Can we cache context gathering for related tasks?
3. **Partial Success**: What if 2 of 3 parallel tasks succeed but 1 fails?
4. **Rollback**: Should we support automated rollback on deployment failure?
5. **Metrics**: What execution metrics should we track for optimization?

---

## Appendix: Claude Code Integration

All agent stages use Claude Code CLI:

```bash
# Basic invocation
claude --print "Your prompt here"

# With context files
claude --print --context file1.go --context file2.go "Your prompt"

# For longer sessions
claude  # Interactive mode

# Non-interactive with output capture
claude --print "prompt" > output.md
```

**Key Points**:
- Use subscription via CLI, not SDK API
- Each pipeline stage is a separate Claude Code invocation
- Context flows via files, not conversation
- Timeouts configured in config.json
