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

## Tech Stack (TBD)

- Language: TBD (Node/TypeScript likely)
- Storage: JSON files
- Interface: CLI + Claude Code
- UI: Optional, later

## Key Design Principles

1. **Taskmaster is always in the loop** — No fire-and-forget
2. **Verification before done** — Prove the work exists
3. **Context flows through tasks** — Not separate inbox
4. **CLI-first** — UI is optional view
5. **Local-first** — JSON files, no external dependencies
6. **Learn from decisions** — Activity log trains AI-Matt

## Related Documentation

- [Architecture](docs/architecture.md) — Full system design
- [Decisions](docs/decisions.md) — Design decision log

## Lineage

Ground Control is a focused successor to Mission Control, keeping what worked (task structure, projects, brain dump) and fixing what didn't (daemon execution, verification, complexity).
