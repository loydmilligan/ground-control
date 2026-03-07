# Project Context

## From CLAUDE.md

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

## From README.md

# Ground Control

> Vibe coding project management — where AI agents pass you around getting decisions, and projects flow from idea to deployed.

## What is Ground Control?

Ground Control is an agent-driven task orchestration system for solo developers who vibe code with AI. Instead of you managing tasks, a **Taskmaster agent** manages the flow — routing work between specialized agents and you, ensuring nothing falls through the cracks.

### The Problem with Current AI Coding

You: "Build me an Android app that sends messages to my TV"
AI: "Done! It's production ready!"
You: *tests it* "This doesn't work at all"

Ground Control fixes this by:
1. **Structured tasks** with explicit outputs and verification
2. **Agent handoffs** with context preservation
3. **Human checkpoints** where decisions actually matter
4. **Verification requirements** before anything is "done"

## Core Concepts

### Taskmaster Agent
The orchestrator. Knows all tasks, priorities, and your patterns. Creates tasks, routes them to appropriate agents, reviews completed work, decides what's next.

### Task Types
- **simple**: Do X (wash car, pay bill)
- **ai-planning**: Chat with planner agent to design something
- **research**: Investigate topics (can be parallel)
- **coding**: Write code with verification
- **human-input**: Need a decision from you

### The Flow
```
Brain Dump (idea)
    → Ingestion Agent (categorize)
    → Taskmaster (create structured task)
    → Agent executes (with verification)
    → Taskmaster reviews (decide next steps)
    → Loop until done
```

### AI-Matt (Future)
Train an agent on your decisions. Set `assigned_human: "ai-matt"` and watch it build autonomously.

## Quick Start

```bash
# Capture an idea
gc dump "Android app to send messages to my TVs"

