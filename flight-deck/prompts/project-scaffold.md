# Project Scaffold

Create the project plan, tasks, architecture, and file structure.

## Context

You are finalizing project setup. This is step 5 of 5.

**Previous steps created**:
- `docs/design.md` - project vision
- `docs/features.md` - features and user stories
- `docs/ui-spec.md` - UI specification

**Your goal**: Create remaining docs and scaffold the project structure.

## Deliverables

1. `docs/project-plan.md` - phased implementation plan (NO time estimates)
2. `docs/tasks.md` - atomic task breakdown
3. `docs/architecture.md` - technical architecture
4. `docs/workflows.md` - development workflows
5. `README.md` - project overview
6. `CLAUDE.md` - Claude/FD integration
7. Directory structure with placeholder files

## Process

### Phase 1: Review All Docs

Read and synthesize:
- design.md → understand the vision
- features.md → understand what to build
- ui-spec.md → understand the interface

### Phase 2: Create Project Plan

Write `docs/project-plan.md`:

```markdown
# [Project Name] - Project Plan

## Phases

### Phase 1: Foundation
**Goal**: [What this phase achieves]

- [ ] [Task group 1]
- [ ] [Task group 2]

### Phase 2: Core Features
**Goal**: [What this phase achieves]

- [ ] [Task group 3]
- [ ] [Task group 4]

### Phase 3: Polish & Launch
**Goal**: [What this phase achieves]

- [ ] [Task group 5]
- [ ] [Task group 6]

## Dependencies

[Phase/task dependencies]

## Risks

- [Risk 1]: [Mitigation]
```

**IMPORTANT**: Do NOT include time estimates. Phases are ordered, not timed.

### Phase 3: Create Task Breakdown

Write `docs/tasks.md`:

```markdown
# [Project Name] - Tasks

## Phase 1: Foundation

### Setup
- [ ] Initialize project structure
- [ ] Set up build tooling
- [ ] Configure linting/formatting

### [Category]
- [ ] [Atomic task 1]
- [ ] [Atomic task 2]

## Phase 2: Core Features

### F-001: [Feature Name]
- [ ] [Task for user story 1]
- [ ] [Task for user story 2]

...
```

Tasks should be atomic - completable in one sitting.

### Phase 4: Create Architecture Doc

Write `docs/architecture.md`:

```markdown
# [Project Name] - Architecture

## Tech Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| Frontend | [tech] | [why] |
| Backend | [tech] | [why] |
| Database | [tech] | [why] |

## Directory Structure

```
project/
├── src/           # Source code
├── docs/          # Documentation
├── tests/         # Test files
└── ...
```

## Key Components

### [Component 1]
[Description and responsibilities]

### [Component 2]
[Description and responsibilities]

## Data Flow

[How data moves through the system]

## External Dependencies

| Dependency | Purpose | Version |
|------------|---------|---------|
| [dep] | [why] | [ver] |
```

### Phase 5: Create Workflows Doc

Write `docs/workflows.md`:

```markdown
# [Project Name] - Development Workflows

## Getting Started

```bash
# Clone and setup
git clone [repo]
cd [project]
[setup commands]
```

## Development

```bash
# Run dev server
[command]

# Run tests
[command]
```

## Deployment

[Deployment process]

## Contributing

[Contribution guidelines]
```

### Phase 6: Create README

Write `README.md`:

```markdown
# [Project Name]

[One-line description]

## Overview

[What this project does, 2-3 sentences]

## Quick Start

```bash
[Quick start commands]
```

## Documentation

- [Design](docs/design.md)
- [Features](docs/features.md)
- [Architecture](docs/architecture.md)

## License

[License]
```

### Phase 7: Create CLAUDE.md

Write `CLAUDE.md`:

```markdown
# [Project Name] - Claude Instructions

## Project Overview

[Brief description]

## Tech Stack

[List technologies]

## Key Commands

```bash
[Common commands]
```

## Flight Deck Integration

This project is managed by Flight Deck. See `.gc/fd-onboarding.md` for details.

### Slash Commands

| Command | Purpose |
|---------|---------|
| `/roadmap-item` | Add feature |
| `/issue` | Report bug |
| `/start-work` | Begin task |
| `/progress` | Update progress |
| `/commit` | Request commit |
| `/complete` | Finish work |

## Conventions

[Project-specific conventions]
```

### Phase 8: Create Directory Structure

Based on architecture, create directories and placeholder files:

```bash
mkdir -p src tests docs
touch src/.gitkeep tests/.gitkeep
# ... based on tech stack
```

## Output

After creating all files:

```
Project scaffold complete!

Created:
- docs/project-plan.md
- docs/tasks.md
- docs/architecture.md
- docs/workflows.md
- README.md
- CLAUDE.md
- [Directory structure]

Next steps:
1. Review generated docs
2. Run: /start-work [first task]
3. Begin coding!
```

## Session End

This session ends when all files are created.
The workflow is complete - project is ready for implementation.
