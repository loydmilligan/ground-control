# Feature Ideas: TUI Feature Parity

## Problem Statement
The CLI has grown significantly with new commands (sprint, artifact, brainstorm, app, orc, delegate, etc.) but the TUI only supports basic task viewing and state changes. Users who prefer the visual interface are missing key functionality.

## Target Users
- Users who prefer visual/interactive interfaces over CLI
- Users managing multiple tasks/sprints at once
- Users who want to see orchestration progress in real-time

## Ideas

### High Priority (Core Functionality)
1. **Brain dump capture in TUI** - Press 'd' to quick-capture an idea without leaving TUI
2. **Task creation in TUI** - Press 'c' to create a task with guided prompts
3. **Sprint view/management** - Tab to switch to sprint view, see sprint progress, add/remove tasks
4. **Orchestration progress view** - Real-time view of pipeline execution with stage status

### Medium Priority (Enhanced UX)
5. **Status dashboard** - Home screen showing: active sessions, pending tasks, recent activity
6. **Process brain dumps** - View unprocessed dumps, process them inline
7. **Quick actions menu** - Press 'a' for action menu: dump, create, orc, etc.
8. **Task detail editing** - Edit task fields directly in TUI

### Nice to Have
9. **Artifact generation wizard** - Walk through template population in TUI
10. **Brainstorm mode** - Interactive brainstorm with Claude in split pane
11. **Delegation controls** - Manage AI Matt delegation from TUI
12. **History view** - See past sessions, completed tasks, activity log

## Constraints
- Keep keyboard-driven (vim-style)
- Must work in standard terminal (no mouse required)
- Should feel snappy, not sluggish
- Bubble Tea framework patterns

## Next Steps
- Create sprint with Phase 1 tasks (1-4)
- Phase 2 for medium priority (5-8)
- Phase 3 for nice-to-haves (9-12)
