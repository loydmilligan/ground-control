# Ground Control Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         INGESTION                                │
│  Brain dump: "I want an Android app that sends messages to TVs" │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    INGESTION AGENT                               │
│  - Categorizes input (new project, bug, enhancement, question)  │
│  - Extracts key info                                            │
│  - Routes to Taskmaster                                         │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    TASKMASTER AGENT                              │
│  - Knows all tasks, priorities, patterns                        │
│  - Creates structured tasks with context + outputs              │
│  - Routes to appropriate agents                                 │
│  - Reviews completed work                                       │
│  - Decides next steps                                           │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      TASK EXECUTION                              │
│                                                                  │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │ Planner  │    │Researcher│    │  Coder   │    │ AI-Matt  │  │
│  │          │    │          │    │          │    │          │  │
│  │ Chat w/  │    │ Parallel │    │ Write +  │    │ Simulates│  │
│  │ human,   │    │ research │    │ verify   │    │ human    │  │
│  │ scaffold │    │ topics   │    │ code     │    │ decisions│  │
│  └────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘  │
│       │               │               │               │         │
│       └───────────────┴───────────────┴───────────────┘         │
│                           │                                      │
│              writes outputs + suggested_next_steps               │
│              triggers after_completion                           │
└──────────────────────────┬──────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│               TASKMASTER REVIEW (loop)                           │
│  - Reads outputs + suggested_next_steps                         │
│  - Creates new tasks                                            │
│  - Assigns to agents or human                                   │
│  - Cycle continues until project complete                       │
└─────────────────────────────────────────────────────────────────┘
```

## Core Principles

### 1. Taskmaster is the Brain
Unlike Mission Control's passive daemon, Taskmaster actively orchestrates. It's always in the loop — reviewing, deciding, routing.

### 2. Verification Before Done
No task is complete without verification. Coding tasks need tests to pass. Planning tasks need outputs to exist. Trust but verify.

### 3. Human Checkpoints
Agents don't make big decisions alone. Human-input tasks route to you (or AI-Matt) for decisions that matter.

### 4. Context Handoffs
Agents communicate through structured task fields, not chat. Context flows: `task.context` → agent → `task.outputs` + `task.suggested_next_steps` → next agent.

### 5. CLI-First
UI is optional. Everything works through files and Claude Code. UI is a view into the system, not the controller.

## Agent Roster

| Agent | Role | Creates Tasks? | Routes to Human? |
|-------|------|----------------|------------------|
| **ingestion** | Categorizes brain dumps, creates initial tasks | Yes (via Taskmaster) | No |
| **taskmaster** | Orchestrates everything, reviews, decides | Yes | Yes |
| **planner** | Chats with human about project vision, scaffolds | No | Yes (for decisions) |
| **researcher** | Investigates topics in parallel | No | No |
| **coder** | Writes code, runs tests, verifies | No | No |
| **ai-matt** | Simulates human decisions | No | N/A (is the human) |

## Task Lifecycle

```
┌─────────┐     ┌───────────┐     ┌──────────┐     ┌───────────┐
│ created │ ──▶ │ assigned  │ ──▶ │ active   │ ──▶ │ completed │
└─────────┘     └───────────┘     └──────────┘     └───────────┘
                      │                │                  │
                      ▼                ▼                  ▼
                 [blocked]        [waiting]         [after_completion
                                  (human)           triggers next]
```

**States:**
- `created`: Task exists but not assigned
- `assigned`: Taskmaster assigned to agent
- `blocked`: Waiting on another task
- `active`: Agent is working on it
- `waiting`: Waiting for human input
- `completed`: Done, outputs exist, verified

## Data Model

### Task
```typescript
interface Task {
  id: string;                          // "task_{timestamp}"
  title: string;
  description: string;

  // Type determines behavior
  type: "simple" | "ai-planning" | "research" | "coding" | "human-input";

  // Assignment
  agent: string | null;                // Which agent executes
  assigned_human: "matt" | "ai-matt";  // Who handles human-input tasks
  autonomy_level: "full" | "checkpoints" | "supervised";

  // Prioritization (hidden from AI — just complexity)
  complexity: 1 | 2 | 3 | 4 | 5;       // AI picks this
  importance: "high" | "medium" | "low";
  due_date: string | null;
  due_urgency: "hard" | "soft" | "none";

  // Context for agent
  context: {
    background: string;                // What agent needs to know
    requirements: string[];            // What must be done
    constraints: string[];             // Limitations
    related_tasks: string[];           // For context
    project_id: string | null;
  };

  // For research tasks
  topics: string[];                    // Parallel research subjects

  // Execution
  state: "created" | "assigned" | "blocked" | "active" | "waiting" | "completed";
  blocked_by: string[];                // Task IDs
  conversation_id: string | null;      // For chat-based tasks

  // Outputs (agent populates)
  outputs: {
    path: string;
    description: string;
    exists: boolean;
  }[];
  suggested_next_steps: string[];

  // Flow control
  after_completion: "taskmaster_review" | "spawn_tasks" | "none";
  verification: {
    type: "test_pass" | "file_exists" | "human_approval" | "none";
    command?: string;                  // For test_pass
    paths?: string[];                  // For file_exists
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

### Project
```typescript
interface Project {
  id: string;                          // "proj_{name}"
  name: string;
  description: string;

  // Status
  status: "active" | "paused" | "completed" | "archived";
  phase: "idea" | "planning" | "research" | "scaffolding" | "building" | "testing" | "deployed" | "maintenance";

  // Assignment
  default_human: "matt" | "ai-matt";
  allowed_agents: string[];

  // Technical
  repo_path: string | null;
  tech_stack: string[];

  // Metadata
  tags: string[];
  created_at: string;
  updated_at: string;

  // AI-Matt training
  decision_history: string[];          // Activity log event IDs
}
```

### Brain Dump
```typescript
interface BrainDump {
  id: string;                          // "bd_{timestamp}"
  content: string;                     // Raw input

  // Ingestion
  processed: boolean;
  category: "idea" | "bug" | "enhancement" | "question" | "reminder" | null;
  urgency_hint: "urgent" | "normal" | null;

  // Outcome
  converted_to: string | null;         // Task or project ID
  ingestion_notes: string | null;      // What ingestion agent decided

  // Metadata
  captured_at: string;
  processed_at: string | null;
}
```

### Activity Log Event
```typescript
interface ActivityEvent {
  id: string;                          // "evt_{timestamp}"
  type: "task_created" | "task_assigned" | "task_completed" | "decision_made" | "project_created" | "ritual_run";

  actor: string;                       // Agent or "matt" or "ai-matt"
  task_id: string | null;
  project_id: string | null;

  // What happened
  summary: string;

  // For AI-Matt training
  reasoning: string | null;            // Why this decision
  alternatives_considered: string[];
  decision_factors: string[];

  // If human made a decision
  human_feedback: string | null;       // If human corrected something

  timestamp: string;
}
```

### Agent Definition
```typescript
interface Agent {
  id: string;                          // "taskmaster", "planner", etc.
  name: string;
  description: string;

  // Capabilities
  instructions: string;                // System prompt
  can_create_tasks: boolean;
  can_route_to_human: boolean;

  // Verification
  verification_required: {
    type: "test_pass" | "file_exists" | "human_approval" | "none";
    default_command?: string;
  };

  // Learning
  learning_enabled: boolean;           // For AI-Matt

  status: "active" | "inactive";
}
```

### Rituals
```typescript
interface Ritual {
  id: string;                          // "daily-standup", "weekly-review"
  name: string;
  description: string;
  agent: string;                       // Usually "taskmaster"
  outputs: string[];                   // Expected output files
  last_run: string | null;
}
```

## File Structure

```
ground-control/
├── data/
│   ├── tasks.json
│   ├── projects.json
│   ├── brain-dump.json
│   ├── activity-log.json
│   ├── agents.json
│   └── rituals.json
├── agents/
│   ├── taskmaster.md          # Taskmaster instructions
│   ├── ingestion.md           # Ingestion agent instructions
│   ├── planner.md             # Planner instructions
│   ├── researcher.md          # Researcher instructions
│   ├── coder.md               # Coder instructions
│   └── ai-matt.md             # AI-Matt instructions (trained)
├── outputs/                   # Agent outputs go here
│   └── {project}/
│       ├── plans/
│       ├── research/
│       └── scaffolds/
├── docs/
│   ├── architecture.md        # This file
│   └── decisions.md           # Design decisions log
├── src/                       # CLI and core logic
├── CLAUDE.md                  # Claude Code instructions
└── README.md
```

## Execution Model

### No Daemon
Ground Control doesn't run a background daemon. Execution is triggered by:

1. **User command**: `gc dump "idea"`, `gc standup`, `gc start task_123`
2. **After-completion**: Task finishes → triggers Taskmaster review
3. **Explicit invocation**: `gc taskmaster` runs Taskmaster to check state

### CLI Commands (Planned)
```bash
gc dump "idea"              # Brain dump
gc tasks                    # List tasks
gc start <task_id>          # Start working on task
gc complete <task_id>       # Mark complete (runs verification)
gc standup                  # Run daily standup ritual
gc weekly                   # Run weekly review ritual
gc taskmaster               # Run Taskmaster to review state
gc projects                 # List projects
gc project <id>             # Show project details
```

### Verification Loop
```
Agent completes task
    → Check verification requirements
    → Run verification (test, file check, etc.)
    → Pass? → Mark complete → Trigger after_completion
    → Fail? → Return to agent with failure context
```

## Time Estimation

**Problem**: AI is bad at estimating time directly.

**Solution**: Complexity tiers (hidden from AI what they mean in time).

AI sees:
> "Rate this task's complexity: 1 (trivial) to 5 (substantial)"

System translates:
| Tier | Minutes |
|------|---------|
| 1 | 2 |
| 2 | 5 |
| 3 | 10 |
| 4 | 20 |
| 5 | 30 |

AI never sees the time values. Just picks complexity. System tracks actuals to refine.

## AI-Matt Training

**Goal**: An agent that can simulate human decisions.

**Training data**:
- Activity log events with `reasoning`, `alternatives_considered`, `decision_factors`
- `human_feedback` when human corrects AI

**Usage**:
```json
{
  "assigned_human": "ai-matt",
  "autonomy_level": "full"
}
```

**Autonomy levels**:
- `full`: AI-Matt makes all decisions
- `checkpoints`: AI-Matt makes routine decisions, escalates big ones
- `supervised`: AI-Matt proposes, human approves each step
