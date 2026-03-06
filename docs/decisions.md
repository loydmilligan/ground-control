# Ground Control Design Decisions

This document captures the key design decisions made during Ground Control planning, with rationale.

---

## Decision 1: New System vs. Evolving Mission Control

**Decision**: Build Ground Control as a new system, not an evolution of Mission Control.

**Rationale**:
- Mission Control accumulated features that don't serve the core use case
- The daemon execution model is fundamentally flawed (marks tasks done without verification)
- Fresh start allows rethinking from first principles
- MC remains available as a reference for what worked

**Date**: 2026-03-05

---

## Decision 2: Taskmaster-Driven vs. Daemon Polling

**Decision**: Taskmaster agent actively orchestrates instead of passive daemon polling.

**Rationale**:
- Daemon model: Poll for tasks → spawn agents → hope for best
- Taskmaster model: Always in loop → reviews → decides → routes
- Taskmaster can make intelligent decisions about sequencing
- Eliminates "fire and forget" failures

**Date**: 2026-03-05

---

## Decision 3: Complexity Tiers vs. Time Estimates

**Decision**: Use 5 complexity tiers instead of time estimates.

**Implementation**:
- AI picks complexity 1-5 based on abstract criteria (trivial → substantial)
- AI does NOT see what tiers translate to in minutes
- System internally maps: 1→2min, 2→5min, 3→10min, 4→20min, 5→30min
- Track actuals to validate/refine tiers

**Rationale**:
- AI is terrible at estimating time directly
- AI is better at estimating relative complexity/size
- Hiding the time translation prevents AI from doing mental time math
- Real-world observation: No AI task ever takes 30 minutes

**Date**: 2026-03-05

---

## Decision 4: CLI-First, UI Optional

**Decision**: Build for CLI/file-based operation first. UI is a view, not the controller.

**Rationale**:
- Mission Control's UI tried to orchestrate, leading to complexity
- Files + Claude Code = sufficient interface for solo dev
- UI can be added later as read-only dashboard
- Reduces scope significantly for V1

**Date**: 2026-03-05

---

## Decision 5: No Separate Inbox/Messaging

**Decision**: Agents communicate through task fields, not a separate inbox.

**How**:
- Input: `task.context`
- Output: `task.outputs` + `task.suggested_next_steps`
- Human communication: Chat-based tasks

**Rationale**:
- MC inbox became noisy and ignored
- Task-based communication keeps context co-located
- Simpler mental model: tasks are the communication

**Date**: 2026-03-05

---

## Decision 6: Decisions as Tasks, Not Separate Entity

**Decision**: Human decisions are `type: "human-input"` tasks, not a separate decisions queue.

**Rationale**:
- MC had tasks, inbox, AND decisions — too many places to check
- A decision is just a task that needs human input
- Unified in task list with appropriate type

**Date**: 2026-03-05

---

## Decision 7: Explicit Outputs and Verification

**Decision**: Every task declares expected outputs and verification requirements.

**Implementation**:
```json
{
  "outputs": [
    { "path": "outputs/project/plan.md", "description": "Project plan" }
  ],
  "verification": {
    "type": "file_exists",
    "paths": ["outputs/project/plan.md"]
  }
}
```

**Rationale**:
- MC's core failure: tasks marked "done" without actual completion
- Explicit outputs = can verify they exist
- Verification requirements = can check programmatically
- No more "it's production ready" when nothing works

**Date**: 2026-03-05

---

## Decision 8: After-Completion Hooks

**Decision**: Tasks specify what happens when they complete via `after_completion`.

**Options**:
- `taskmaster_review`: Taskmaster evaluates and decides next steps
- `spawn_tasks`: Pre-defined follow-up tasks get created
- `none`: Task is terminal

**Rationale**:
- Creates automatic flow without daemon polling
- Taskmaster stays in the loop
- Enables process/workflow patterns

**Date**: 2026-03-05

---

## Decision 9: AI-Matt as Trained Human Simulacrum

**Decision**: Create an agent that can simulate human decisions.

**Training sources**:
- Activity log with reasoning captured
- Decision factors logged
- Human feedback when corrections made

**Usage**:
- Set `assigned_human: "ai-matt"` on project
- Set `autonomy_level` to control how much AI-Matt decides alone

**Rationale**:
- End goal: Describe idea → watch it build
- Requires agent that knows your preferences
- Must be trainable from real decisions

**Date**: 2026-03-05

---

## Decision 10: Rituals as On-Demand, Not Scheduled

**Decision**: Daily standup, weekly review are user-triggered, not cron-scheduled.

**Implementation**:
- `gc standup` runs daily standup when user wants
- `gc weekly` runs weekly review when user wants
- Optional "last run" indicator for gentle nudging

**Rationale**:
- Scheduled rituals feel like nagging
- User knows when they want orientation
- Simpler than cron infrastructure

**Date**: 2026-03-05

---

## Decision 11: Project Phases vs. Goal Hierarchy

**Decision**: Simple linear phases per project instead of complex goal/milestone hierarchy.

**Phases**:
`idea → planning → research → scaffolding → building → testing → deployed → maintenance`

**Rationale**:
- MC's goal hierarchy was over-engineered
- Most projects follow similar phase progression
- Taskmaster uses phase to contextualize decisions
- Simpler mental model

**Date**: 2026-03-05

---

## Decision 12: Linear Sync Optional, Local-First

**Decision**: Keep optional Linear integration but local JSON is source of truth.

**Rationale**:
- Linear is expensive ($8/month)
- Local files work offline
- Linear sync when you want the UI or need to share
- Don't depend on external service

**Date**: 2026-03-05

---

## Decision 13: Activity Log for Learning

**Decision**: Enhanced activity log that captures reasoning for AI-Matt training.

**Fields**:
- `reasoning`: Why this decision was made
- `alternatives_considered`: What other options existed
- `decision_factors`: What drove the choice
- `human_feedback`: If human corrected something

**Rationale**:
- Standard activity log just says what happened
- AI-Matt needs to learn why
- Training data must include decision context

**Date**: 2026-03-05

---

## Decision 14: Blocked-By as Optional, After-Completion as Primary

**Decision**: Keep `blocked_by` for explicit dependencies, but `after_completion` handles most sequencing.

**Rationale**:
- Most task sequences are linear: A → B → C
- `after_completion: taskmaster_review` handles this naturally
- `blocked_by` for cases where multiple tasks must complete first
- Simpler than complex dependency graphs

**Date**: 2026-03-05

---

## Decision 15: Multi-Topic Research Tasks

**Decision**: Research tasks can have multiple topics for parallel investigation.

**Implementation**:
```json
{
  "type": "research",
  "topics": [
    "Market research for app viability",
    "UI inspiration from similar apps",
    "Architecture options for scope"
  ]
}
```

**Rationale**:
- Often have multiple research questions at once
- More efficient than serial tasks
- Agent can parallelize or batch
- Single output synthesizes findings

**Date**: 2026-03-05

---

## Future Decisions (To Be Made)

- [ ] Tech stack for CLI (Node/TypeScript? Python? Rust?)
- [ ] Storage format (JSON files vs. SQLite)
- [ ] Chat interface implementation
- [ ] How AI-Matt training actually works
- [ ] UI approach if/when we add it
