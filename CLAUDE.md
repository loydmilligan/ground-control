# Ground Control — AI Agent Instructions

## What is Ground Control?

Ground Control is a vibe coding project management system where AI agents orchestrate task flow. The Taskmaster agent manages priorities and routing, specialized agents execute work, and humans make decisions at checkpoints.

## Quick Context

```
Brain Dump → Ingestion Agent → Taskmaster → Agent Execution → Verification → Loop
```

**Key insight**: Unlike traditional task managers, Ground Control verifies work is actually done before marking complete. No more "it's production ready" when nothing works.

## Core Files

| File | Purpose |
|------|---------|
| `data/tasks.json` | All tasks with full context |
| `data/projects.json` | Project definitions |
| `data/brain-dump.json` | Raw ideas awaiting processing |
| `data/activity-log.json` | Audit trail + AI-Matt training data |
| `data/agents.json` | Agent definitions |
| `data/rituals.json` | On-demand rituals (standup, weekly) |
| `agents/*.md` | Agent instruction prompts |

## Data Schemas

### Task Schema
```typescript
interface Task {
  id: string;                    // "task_{timestamp}"
  title: string;
  description: string;
  type: "simple" | "ai-planning" | "research" | "coding" | "human-input";

  // Assignment
  agent: string | null;
  assigned_human: "matt" | "ai-matt";
  autonomy_level: "full" | "checkpoints" | "supervised";

  // Priority (complexity is AI-facing, hides time translation)
  complexity: 1 | 2 | 3 | 4 | 5;   // 1=trivial, 5=substantial
  importance: "high" | "medium" | "low";
  due_date: string | null;
  due_urgency: "hard" | "soft" | "none";

  // Context for executing agent
  context: {
    background: string;
    requirements: string[];
    constraints: string[];
    related_tasks: string[];
    project_id: string | null;
  };

  // For research tasks
  topics: string[];

  // State
  state: "created" | "assigned" | "blocked" | "active" | "waiting" | "completed";
  blocked_by: string[];
  conversation_id: string | null;

  // Outputs (agent populates)
  outputs: { path: string; description: string; exists: boolean; }[];
  suggested_next_steps: string[];

  // Flow
  after_completion: "taskmaster_review" | "spawn_tasks" | "none";
  verification: {
    type: "test_pass" | "file_exists" | "human_approval" | "none";
    command?: string;
    paths?: string[];
  };

  // Metadata
  project_id: string | null;
  tags: string[];
  created_at: string;
  updated_at: string;
  completed_at: string | null;

  // Tracking
  actual_minutes: number | null;
  tokens_used: number | null;
  lines_changed: number | null;
}
```

### Project Schema
```typescript
interface Project {
  id: string;
  name: string;
  description: string;
  status: "active" | "paused" | "completed" | "archived";
  phase: "idea" | "planning" | "research" | "scaffolding" | "building" | "testing" | "deployed" | "maintenance";
  default_human: "matt" | "ai-matt";
  allowed_agents: string[];
  repo_path: string | null;
  tech_stack: string[];
  tags: string[];
  created_at: string;
  updated_at: string;
}
```

### Brain Dump Schema
```typescript
interface BrainDump {
  id: string;
  content: string;
  processed: boolean;
  category: "idea" | "bug" | "enhancement" | "question" | "reminder" | null;
  urgency_hint: "urgent" | "normal" | null;
  converted_to: string | null;
  ingestion_notes: string | null;
  captured_at: string;
  processed_at: string | null;
}
```

### Activity Log Schema
```typescript
interface ActivityEvent {
  id: string;
  type: "task_created" | "task_assigned" | "task_completed" | "decision_made" | "project_created" | "ritual_run";
  actor: string;
  task_id: string | null;
  project_id: string | null;
  summary: string;

  // AI-Matt training data
  reasoning: string | null;
  alternatives_considered: string[];
  decision_factors: string[];
  human_feedback: string | null;

  timestamp: string;
}
```

## Agent Roster

| Agent | Role | Key Capability |
|-------|------|----------------|
| `taskmaster` | Orchestrator | Creates tasks, reviews, routes |
| `ingestion` | Categorizer | Processes brain dumps |
| `planner` | Designer | Chats about vision, creates scaffolds |
| `researcher` | Investigator | Parallel topic research |
| `coder` | Implementer | Writes code, runs tests |
| `ai-matt` | Human simulacrum | Trained on user decisions |

## Complexity Rating

When estimating task complexity, use this scale:

| Level | Description |
|-------|-------------|
| 1 | Trivial — single change, obvious |
| 2 | Small — few changes, straightforward |
| 3 | Medium — multiple changes, some thought required |
| 4 | Large — significant changes, coordination needed |
| 5 | Substantial — major changes, consider breaking down |

**Important**: These levels do NOT have time values. Do not try to estimate time. Just assess complexity.

## Task Lifecycle

1. **Created**: Task exists, not assigned
2. **Assigned**: Taskmaster assigned to agent
3. **Blocked**: Waiting on `blocked_by` tasks
4. **Active**: Agent working on it
5. **Waiting**: Needs human input
6. **Completed**: Done, verified, outputs exist

## Verification Requirements

Before marking a task complete, verification must pass:

- `test_pass`: Run command, must exit 0
- `file_exists`: All paths must exist
- `human_approval`: Human must approve
- `none`: No verification (rare)

## Writing Tasks

When creating tasks, always include:

1. **Clear context.background** — What does the agent need to know?
2. **Specific context.requirements** — What exactly must be done?
3. **Expected outputs** — What files/artifacts will be produced?
4. **Verification** — How do we know it's actually done?
5. **after_completion** — What happens next? (usually `taskmaster_review`)

## Activity Logging

When logging decisions, capture:

- **reasoning**: Why was this decision made?
- **alternatives_considered**: What other options existed?
- **decision_factors**: What drove the choice?

This data trains AI-Matt.

## Commands (Planned)

```bash
gc dump "idea"              # Brain dump
gc tasks                    # List tasks
gc start <task_id>          # Start task
gc complete <task_id>       # Complete with verification
gc standup                  # Daily standup ritual
gc weekly                   # Weekly review ritual
gc taskmaster               # Run Taskmaster
```

## Tech Stack

- **Language**: Go
- **CLI Framework**: Cobra
- **TUI Framework**: Bubble Tea (Charm ecosystem)
- **Styling**: Lip Gloss
- **Storage**: JSON files
- **Interface**: CLI + TUI + Claude Code

## Key Design Principles

1. **Taskmaster is always in the loop** — No fire-and-forget
2. **Verification before done** — Prove the work exists
3. **Context flows through tasks** — Not separate inbox
4. **CLI-first** — UI is optional view
5. **Local-first** — JSON files, no external dependencies
6. **Learn from decisions** — Activity log trains AI-Matt

## AI Matt Delegation System

When working with AI Matt (another Claude instance acting as Matt's decision-making proxy):

### Key Files
- `agents/ai-matt.md` — AI Matt's personality and decision guidelines
- `docs/ai-matt-delegation-system.md` — Full delegation protocol documentation
- `data/delegation/state.json` — Current delegation state
- `data/delegation/inbox.md` — Messages TO AI Matt
- `data/delegation/outbox.md` — Messages FROM AI Matt

### Starting AI Matt in Another Pane

AI Matt must be started with the system prompt:
```bash
claude --system-prompt agents/ai-matt.md
```

Or send initialization to an existing Claude session:
```bash
tmux send-keys -t 1 'Please read @agents/ai-matt.md - you are AI Matt for this session. Your job is to make decisions as Matt would. Check data/delegation/inbox.md for tasks.' Enter
```

### Handoff Protocol

When delegating to AI Matt:
1. Write task/question to `data/delegation/inbox.md`
2. Run `gc handoff --to-ai-matt`
3. AI Matt reads inbox, decides, writes to `data/delegation/outbox.md`
4. AI Matt runs `gc handoff --to-claude`
5. Worker reads outbox and continues

### Quick Commands
```bash
gc delegate --interactions 5           # Delegate for 5 interactions
gc delegate --supervised -i 5          # Supervised mode with monitor pane
gc delegate --supervised -i 5 --no-auth # Skip password for testing
gc delegate --status                   # Check delegation state
gc delegate --cancel                   # Take back control
gc handoff --to-ai-matt                # Signal handoff to AI Matt
gc handoff --check-inbox               # AI Matt checks for work
gc handoff --to-claude                 # AI Matt signals response ready
gc handoff --check-outbox              # Worker checks for response
```

### Supervised Delegation

In supervised mode (`--supervised`):
- Dedicated tmux window created for Worker Claude + AI Matt
- Monitor pane added to your window showing live communication
- Password-protected approvals (prevents AI from approving on user's behalf)

To enable password protection:
1. Run `scripts/gc-hash-password.sh` to generate a password hash
2. Add `GC_APPROVAL_PASSWORD_HASH=<hash>` to `.env`
3. Monitor will require password for session start and each approval

### AI Matt Restricted Environment

AI Matt runs in a restricted Claude environment (`.claude-ai-matt/`):
- Read-only: Cannot edit/write files directly
- Pre-tool hooks block dangerous operations
- Must request changes via handoff

Setup: `scripts/setup-ai-matt-env.sh`

## Related Documentation

- [Architecture](docs/architecture.md) — Full system design
- [Decisions](docs/decisions.md) — Design decision log
- [AI Matt Delegation](docs/ai-matt-delegation-system.md) — Delegation protocol
- [Pipeline Architecture](docs/pipeline-architecture.md) — Orchestration details

## Lineage

Ground Control is a focused successor to Mission Control, keeping what worked (task structure, projects, brain dump) and fixing what didn't (daemon execution, verification, complexity).
