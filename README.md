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

# See what Taskmaster created
gc tasks

# Run a ritual
gc standup
gc weekly

# Start a task (opens chat with appropriate agent)
gc start task_123
```

## Project Status

**Phase: Planning**

See [docs/architecture.md](docs/architecture.md) for full design.
See [docs/decisions.md](docs/decisions.md) for design decisions.

## Lineage

Ground Control is a spiritual successor to Mission Control, focused on:
- Agent orchestration over UI dashboards
- Verification over trust
- Simplicity over features
- CLI-first over web-first

## License

MIT
