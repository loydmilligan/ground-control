# Feature & Workflow Inventory

> **Version**: 0.1.0 | **Last Updated**: 2026-03-11

This document tracks all features, user stories, and workflows in Ground Control / Flight Deck, including their implementation type (corner-cut vs native) and step-by-step definitions.

---

## Schema Definitions

### Feature

A **Feature** is a major function of the software (e.g., auth system, CRUD screen, project adoption).

```yaml
feature:
  id: string                    # feat_xxx
  name: string                  # Human-readable name
  description: string           # What this feature does
  status: planned | in_progress | corner_cut | native | hybrid

  ui_elements:                  # Frontend/interface components
    - id: string
      name: string
      description: string
      implementation: corner_cut | native | none

  functions:                    # Backend/logic components
    - id: string
      name: string
      description: string
      implementation: corner_cut | native | none

  user_stories: [story_id]      # Related user stories
  workflows: [workflow_id]      # Related workflows
```

### User Story

A **User Story** is a single main task with multiple steps. One task, clear goal, few failure points.

```yaml
user_story:
  id: string                    # story_xxx
  name: string                  # Brief name
  actor: string                 # Who performs this (user, fd_claude, project_claude)
  goal: string                  # What they want to accomplish
  preconditions: [string]       # What must be true before starting

  steps:
    - number: int
      action: string            # What the actor does
      system_response: string   # What the system does/shows
      output: string            # What is produced (file, state change, etc.)
      failure_points: [string]  # What could go wrong

  success_criteria: string      # How we know it worked
  implementation: corner_cut | native | hybrid
  prompt_file: string           # If corner_cut, which prompt handles this
```

### Workflow

A **Workflow** is multiple user stories strung together to accomplish a larger goal.

```yaml
workflow:
  id: string                    # wf_xxx
  name: string
  description: string
  actor: string

  stories:                      # Ordered list of user stories
    - story_id: string
      session_boundary: boolean # Does session END after this story?
      handoff_artifact: string  # What is passed to next story

  implementation: corner_cut | native | hybrid
  entry_point: string           # How user initiates (command, keybinding, etc.)
```

### Implementation Types

| Type | Description |
|------|-------------|
| **corner_cut** | Prompt-based, Claude guides user through conversation |
| **native** | Built into tools (Go code, TUI, CLI commands) |
| **hybrid** | Mix of prompts and native code |
| **none** | Not yet implemented |

### Session Boundaries

Sessions END between workflow steps when:
- A significant artifact is produced (file, config, decision)
- User review/approval is needed
- Context switch to different project/directory
- Handoff between FD Claude and Project Claude

---

## Inventory

### Workflows

#### WF-001: Start New Project

```yaml
workflow:
  id: wf_001
  name: Start New Project
  description: |
    User initiates a new project from within Flight Deck.
    Covers: directory creation, adoption, vision definition,
    initial roadmap, and handoff to implementation.
  actor: user + fd_claude
  implementation: corner_cut
  entry_point: FD TUI keybinding 'n' OR /new-project command

  stories:
    - story_id: story_001
      session_boundary: false
      handoff_artifact: null

    - story_id: story_002
      session_boundary: true
      handoff_artifact: .gc/project.json + .gc/analysis.json

    - story_id: story_003
      session_boundary: true
      handoff_artifact: .gc/vision.md (or embedded in roadmap)

    - story_id: story_004
      session_boundary: true
      handoff_artifact: .gc/roadmap.json with initial features

    - story_id: story_005
      session_boundary: true
      handoff_artifact: project ready for /start-work
```

---

### User Stories

#### STORY-001: Create Project Directory

```yaml
user_story:
  id: story_001
  name: Create Project Directory
  actor: user (via FD)
  goal: Create a new directory for the project
  preconditions:
    - User is in FD TUI or FD Claude session
    - User knows desired project name/path

  steps:
    - number: 1
      action: User triggers new project (keybinding 'n' or /new-project)
      system_response: FD prompts for project path/name
      output: User input captured
      failure_points:
        - User cancels
        - Invalid path characters

    - number: 2
      action: User enters project path (e.g., ~/Projects/my-new-app)
      system_response: FD validates path doesn't exist or is empty
      output: Validated path
      failure_points:
        - Directory already exists with files
        - Permission denied
        - Parent directory doesn't exist

    - number: 3
      action: FD creates directory
      system_response: Directory created, confirmation shown
      output: Empty directory at specified path
      failure_points:
        - Filesystem error

  success_criteria: Empty directory exists at specified path
  implementation: native (TUI code)
  prompt_file: null
```

#### STORY-002: Adopt Project into FD

```yaml
user_story:
  id: story_002
  name: Adopt Project into FD
  actor: system (FD)
  goal: Initialize .gc/ sidecar and register project
  preconditions:
    - Directory exists
    - Directory is not already adopted (or --force)

  steps:
    - number: 1
      action: FD runs gc adopt on the new directory
      system_response: Adoption process begins
      output: Process started
      failure_points:
        - gc binary not found

    - number: 2
      action: gc adopt creates .gc/ structure
      system_response: Creates project.json, state.json, roadmap.json, etc.
      output: .gc/ directory with initial files
      failure_points:
        - Write permission denied

    - number: 3
      action: gc adopt registers in ~/.gc/global.json
      system_response: Project added to registry
      output: Project visible in FD
      failure_points:
        - Registry corruption

  success_criteria: .gc/ exists, project appears in FD Hangar
  implementation: native (gc adopt command)
  prompt_file: null
```

#### STORY-003: Define Project Vision

```yaml
user_story:
  id: story_003
  name: Define Project Vision
  actor: user + fd_claude
  goal: Capture what the project is and what problem it solves
  preconditions:
    - Project is adopted
    - User has context (reference files, ideas, etc.)

  steps:
    - number: 1
      action: FD opens Claude session with /project-vision prompt
      system_response: Claude asks "What is this project? What problem does it solve?"
      output: Conversation started
      failure_points:
        - Session fails to start

    - number: 2
      action: User describes project, shares reference files
      system_response: Claude reads files, asks clarifying questions
      output: Claude has context
      failure_points:
        - Files not found
        - Context too large

    - number: 3
      action: User answers clarifying questions
      system_response: Claude synthesizes understanding
      output: Shared understanding of project
      failure_points:
        - Miscommunication

    - number: 4
      action: Claude summarizes vision and confirms
      system_response: "Here's what I understand: [summary]. Is this correct?"
      output: Confirmed vision summary
      failure_points:
        - User disagrees, needs iteration

    - number: 5
      action: Claude writes vision to artifact
      system_response: Vision saved to .gc/vision.md or embedded in roadmap.json
      output: Persisted vision document
      failure_points:
        - Write failure

  success_criteria: Vision document exists, user confirmed accuracy
  implementation: corner_cut
  prompt_file: prompts/project-vision.md
```

#### STORY-004: Create Initial Roadmap

```yaml
user_story:
  id: story_004
  name: Create Initial Roadmap
  actor: user + fd_claude
  goal: Define MVP features and initial roadmap items
  preconditions:
    - Vision is defined (story_003 complete)
    - User knows rough scope

  steps:
    - number: 1
      action: FD opens Claude session with /project-roadmap prompt
      system_response: Claude reviews vision, asks "What's the minimum to prove this works?"
      output: MVP scoping conversation started
      failure_points:
        - Vision file not found

    - number: 2
      action: User describes MVP scope
      system_response: Claude suggests feature breakdown
      output: Draft feature list
      failure_points:
        - Scope creep

    - number: 3
      action: User confirms/adjusts features
      system_response: Claude sets priorities (high=MVP, medium/low=later)
      output: Prioritized feature list
      failure_points:
        - Disagreement on priorities

    - number: 4
      action: Claude generates roadmap.json entries
      system_response: Creates feat_* entries with titles, descriptions, priorities
      output: .gc/roadmap.json populated
      failure_points:
        - JSON write failure

    - number: 5
      action: Claude confirms roadmap written
      system_response: "Created X features: [list]. Ready to start?"
      output: Roadmap confirmed
      failure_points:
        - None

  success_criteria: roadmap.json has MVP features, user confirmed
  implementation: corner_cut
  prompt_file: prompts/project-roadmap.md
```

#### STORY-005: Set Project Phase and Handoff

```yaml
user_story:
  id: story_005
  name: Set Project Phase and Handoff
  actor: fd_claude
  goal: Configure project for implementation phase
  preconditions:
    - Roadmap exists
    - User ready to begin implementation

  steps:
    - number: 1
      action: Claude updates project.json with phase
      system_response: Sets phase to "mvp" or "design" as appropriate
      output: project.json updated
      failure_points:
        - Write failure

    - number: 2
      action: Claude summarizes what's ready
      system_response: Lists files created, next steps
      output: Handoff summary
      failure_points:
        - None

    - number: 3
      action: Claude provides next action
      system_response: "To start: cd [path] && claude, then /start-work feat_xxx"
      output: Clear next step
      failure_points:
        - None

  success_criteria: Project configured, user knows next action
  implementation: corner_cut
  prompt_file: prompts/project-handoff.md
```

---

## Features

#### FEAT-001: Project Initialization

```yaml
feature:
  id: feat_001
  name: Project Initialization
  description: |
    Complete system for creating and setting up new projects
    within Flight Deck, from directory creation through
    implementation readiness.
  status: corner_cut

  ui_elements:
    - id: ui_001
      name: New Project Keybinding
      description: 'n' key in FD TUI triggers new project flow
      implementation: native (needs building)

    - id: ui_002
      name: Project Path Input
      description: Text input for project path/name
      implementation: native (needs building)

  functions:
    - id: fn_001
      name: Directory Creation
      description: Create project directory at specified path
      implementation: native (simple mkdir)

    - id: fn_002
      name: Project Adoption
      description: Run gc adopt on new directory
      implementation: native (existing gc adopt)

    - id: fn_003
      name: Vision Capture
      description: Guide user through vision definition
      implementation: corner_cut (prompt)

    - id: fn_004
      name: Roadmap Generation
      description: Create initial roadmap from vision
      implementation: corner_cut (prompt)

    - id: fn_005
      name: Phase Configuration
      description: Set project phase and prepare handoff
      implementation: corner_cut (prompt)

  user_stories: [story_001, story_002, story_003, story_004, story_005]
  workflows: [wf_001]
```

---

## Brainstorm Queue

Items that need brainstorming before implementation:

| ID | Item | Status | Notes |
|----|------|--------|-------|
| BQ-001 | Start New Project workflow prompts | pending | Need to define exact prompt boundaries |
| BQ-002 | Session handoff mechanism | pending | How does one session pass to next? |

---

## Implementation Status

| ID | Name | Type | Status | Notes |
|----|------|------|--------|-------|
| WF-001 | Start New Project | workflow | pending | Needs prompts |
| STORY-001 | Create Project Directory | story | pending | Needs TUI code |
| STORY-002 | Adopt Project | story | native | gc adopt exists |
| STORY-003 | Define Vision | story | pending | Needs prompt |
| STORY-004 | Create Roadmap | story | pending | Needs prompt |
| STORY-005 | Phase & Handoff | story | pending | Needs prompt |

---

## Glossary

| Term | Definition |
|------|------------|
| **Feature** | Major function of software (auth, CRUD, etc.) |
| **UI Element** | Frontend/interface component of a feature |
| **Function** | Backend/logic component of a feature |
| **User Story** | Single main task with multiple steps |
| **Workflow** | Multiple user stories strung together |
| **Corner-cut** | Prompt-based implementation |
| **Native** | Built into tools (Go code, TUI, CLI) |
| **Session Boundary** | Point where Claude session ends, artifact handed off |
| **Handoff Artifact** | File/state passed between workflow steps |
