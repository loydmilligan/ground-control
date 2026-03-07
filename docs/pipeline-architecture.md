# Pipeline Architecture v2

## Overview

Pipelines are composable sequences of stages that process tasks. Each task type maps to a pipeline. Pipelines can call other pipelines as sub-stages, enabling complex workflows from simple building blocks.

---

## Core Concepts

### Pipeline

A named sequence of stages that processes a task.

```yaml
pipeline: coding
stages:
  - sanity
  - coder
  - reviewer  # loops with coder
  - tester    # loops with coder
  - commit
```

### Stage

An AI agent with defined inputs, outputs, and instructions.

```yaml
stage: coder
agent: coder
inputs:
  - task.context_bundle
  - feedback (optional, from reviewer/tester)
outputs:
  - implementation_notes.md
  - code changes
loops_with: reviewer, tester
```

### Sub-Pipelines

A stage can invoke another pipeline. Parent with subs = flat string of stages.

```yaml
pipeline: create_app
stages:
  - artifact_generation(project_plan.template)
  - artifact_generation(tasks.template, input: project_plan.md)
  - foreach: tasks
    pipeline: coding
  - integration
  - deploy
```

### Loops

Iterative stages with conditionals. Stage A runs, output checked, may loop back.

```
coder → reviewer
  ↑        │
  └────────┘  (if NEEDS_REVISION)
```

---

## Task Types → Pipelines

| Task Type | Pipeline | Stages |
|-----------|----------|--------|
| `coding` | coding | sanity → coder → reviewer → tester → commit |
| `simple` | simple | sanity → coder → commit |
| `research` | research | sanity → researcher → summary |
| `ai-planning` | artifact_generation | sanity → planner_chat → artifact_output |
| `human-input` | human_input | notify → wait_human → capture_response |
| `commit` | commit | sanity → commit |

---

## Stage Definitions

### Existing Stages

| Stage | Purpose | Inputs | Outputs |
|-------|---------|--------|---------|
| `sanity` | Validate task has context, requirements, no blockers | task | pass/fail |
| `coder` | Implement the task using Claude CLI | context_bundle, feedback? | code changes, notes |
| `reviewer` | Review code, decide APPROVED/NEEDS_REVISION/ESCALATE | code changes | decision, feedback |
| `tester` | Run tests, report pass/fail | code changes | test results, feedback |
| `commit` | Stage changes, generate message, commit | code changes | commit hash |

### New Stages

| Stage | Purpose | Inputs | Outputs |
|-------|---------|--------|---------|
| `researcher` | Investigate topics, gather information | topics[], context | findings.md |
| `summary` | Synthesize research into deliverable | findings.md | summary.md |
| `planner_chat` | Chat loop to populate template variables | template, user | filled variables |
| `artifact_output` | Generate final artifact from template + variables | template, variables | artifact.md |
| `notify` | Alert human that input is needed | task, questions | notification |
| `wait_human` | Pause until human provides input | questions | responses |
| `capture_response` | Record human responses, update task | responses | task update |

---

## Parallelization

### Rules

1. Taskmaster evaluates at stage start: "Can tasks in this stage be parallelized?"
2. Recommendation made, stage agent can ignore
3. Limit: max N concurrent agents (configurable, default: 3)
4. Dependencies respected: blocked tasks wait

### Example

```
Tasks: [A, B, C, D]
Dependencies: C blocked_by A

Stage: coder
Taskmaster thinks: "A, B, D can parallelize. C waits for A."
Executes: A, B, D in parallel (up to limit)
When A completes: C can start
```

---

## Failure Handling

```
Stage fails
    │
    ▼
Taskmaster analyzes failure
    │
    ├── Recoverable? → Adjust inputs, retry stage
    │                      │
    │                      └── Still fails? → Escalate
    │
    └── Unrecoverable? → Escalate immediately
```

### Escalation

- Session paused
- Human notified with failure context
- Resume with `gc orc --continue` after human provides guidance

---

## Human Input Protocol

When AI needs human input:

1. **AI creates questions file**
   ```markdown
   # Human Input Required

   ## Questions

   ### Q1: Database choice
   Which database should we use?
   - [ ] PostgreSQL (Recommended) - Best for relational data, robust
   - [ ] SQLite - Simple, file-based, good for small projects
   - [ ] MongoDB - Document store, flexible schema

   ### Q2: Authentication method
   ...
   ```

2. **Stage output includes flag**
   ```json
   {
     "status": "needs_human_input",
     "questions_file": "data/context/task_123/questions.md"
   }
   ```

3. **Taskmaster sees flag, offers options**
   ```
   Your input is required to continue task X.

   Questions:
   1. Database choice (PostgreSQL recommended)
   2. Authentication method

   Options:
   [A] Answer now
   [B] Delegate to AI Matt
   [C] Remind me later
   ```

4. **Responses captured, task continues**

---

## Artifact Generation

### Template Structure

```markdown
# {artifact_name}.template

## Document

{The actual document content with <variable_name> placeholders}

## Variables

| Variable | Line | Type | Description |
|----------|------|------|-------------|
| project_name | 3 | string | Short identifier for the project |
| project_goal | 5 | string | One sentence: "Users can ___" |
| tech_stack | 12 | list | Languages, frameworks, tools |

## Planning Guidance

### project_name
Ask: "What should we call this? Something short you'll type often."
Examples: "ground-control", "task-flow", "my-app"

### project_goal
Ask: "If this project succeeds, what's different? Complete: 'Users can ___'"
Push for specificity. Vague → ask follow-up.

### tech_stack
[Teaching mode]
"Let's figure out the right tools. What kind of app is this?"
- Web app → suggest: React/Vue + Node/Python + Postgres
- CLI tool → suggest: Go/Rust/Python
- Mobile → suggest: React Native / Flutter

Explain tradeoffs briefly if user unsure.
```

### Artifact Dependencies

Artifacts can require other artifacts as input:

```yaml
artifact: tasks.md
template: tasks.template
requires:
  - project_plan.md
generation:
  - Parse project_plan.md for features
  - For each feature, generate task structure
  - Ask pointed questions to refine
```

---

## Context Flow

Context flows through stages via:

1. **Task context bundle** - files in `data/context/task_xxx/`
2. **Stage outputs** - each stage writes outputs, next stage reads
3. **Session state** - `data/sessions/session_xxx.json` tracks all

```
sanity
  └─→ coder (reads: context_bundle)
        └─→ reviewer (reads: coder output)
              └─→ coder (reads: reviewer feedback) [loop]
                    └─→ tester (reads: code changes)
                          └─→ commit (reads: all changes)
```

Sub-pipelines inherit parent context automatically.

---

## Configuration

### Pipeline Definitions

Store in `data/pipelines/` or `config/pipelines/`:

```yaml
# coding.pipeline.yaml
name: coding
description: Standard code implementation pipeline
stages:
  - name: sanity
    required: true
  - name: coder
    loops_with: [reviewer, tester]
    max_iterations: 3
  - name: reviewer
    decision_outputs: [APPROVED, NEEDS_REVISION, ESCALATE]
  - name: tester
    skip_if: no_tests_defined
  - name: commit
    skip_if: no_changes
```

### Limits

```yaml
# config/orchestration.yaml
parallelization:
  max_concurrent_agents: 3
  max_concurrent_tasks: 5

timeouts:
  stage_default: 30m
  coder: 45m
  planner_chat: 60m

iterations:
  max_revision_loops: 3
  max_test_loops: 2
```

---

## User Profile (Future)

```json
{
  "id": "matt",
  "knowledge_areas": {
    "go": "proficient",
    "python": "expert",
    "kubernetes": "beginner",
    "databases": "intermediate"
  },
  "preferences": {
    "verbosity": "concise",
    "teaching_mode": true,
    "default_recommendations": true
  },
  "decision_history": [
    {"topic": "database", "chose": "postgres", "context": "web app"}
  ]
}
```

Used by planner to adjust teaching depth and recommendations.

---

## Implementation Plan

### Phase 1: Task Type Routing (Task #5)
- Add `getPipelineForTask()` in orc.go
- Route to existing stages based on type
- Update `--dry-run` to show correct pipeline

### Phase 2: Missing Stages
- Implement `researcher`, `summary` stages
- Implement `notify`, `wait_human`, `capture_response` stages
- Implement `planner_chat`, `artifact_output` stages

### Phase 3: Composable Pipelines
- Pipeline definition files (YAML)
- Sub-pipeline invocation
- Foreach construct for task lists

### Phase 4: Artifact Generation
- Template file format
- Planning guidance parser
- Chat loop implementation
- Template variable population

### Phase 5: Parallelization
- Taskmaster parallelization analysis
- Concurrent agent management
- Dependency-aware scheduling

### Phase 6: CreateAppPipeline
- Compose all pieces
- End-to-end: idea → deployed app

---

## File Structure

```
ground-control/
├── config/
│   ├── orchestration.yaml
│   └── pipelines/
│       ├── coding.yaml
│       ├── research.yaml
│       ├── artifact_generation.yaml
│       └── create_app.yaml
├── data/
│   ├── templates/
│   │   ├── project_plan.template
│   │   ├── tasks.template
│   │   └── architecture.template
│   └── user_profile.json
└── internal/
    └── pipeline/
        ├── registry.go      # Pipeline/stage registry
        ├── executor.go      # Pipeline execution engine
        ├── parallelizer.go  # Parallelization logic
        └── stages/
            ├── sanity.go
            ├── coder.go
            ├── reviewer.go
            ├── tester.go
            ├── commit.go
            ├── researcher.go
            ├── planner.go
            └── human_input.go
```
