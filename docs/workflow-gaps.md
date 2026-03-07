# Ground Control Workflow Gaps

**Date:** 2026-03-07
**Discovered during:** NotifAI Carousel UI sprint orchestration attempt

---

## Summary

When attempting to use Ground Control to orchestrate tasks for an external project (NotifAI), we discovered several workflow gaps that prevent the system from working as intended.

---

## Gap 1: Context Bundle Requirement

### Problem
Tasks created programmatically (via JSON/jq) cannot be orchestrated because they lack context bundles.

```bash
$ ./gc orc 12
Error: sanity check failed: task has no context bundle - run 'gc create' or rebuild context
```

### Current State
- `gc create` is interactive and builds a context bundle
- Context bundles are stored in `data/context/task_xxx/`
- The sanity check stage requires a context bundle to exist
- There's no way to build a context bundle for an existing task

### Options to Fix

| Option | Description | Effort |
|--------|-------------|--------|
| A. `gc bundle` command | New command to build context bundle for existing task | Medium |
| B. `--skip-sanity` flag | Allow bypassing sanity check | Low |
| C. Auto-bundle in pipeline | First stage builds bundle if missing | Medium |
| D. Make bundles optional | Remove hard requirement | Low |

### Recommendation
Option A (`gc bundle`) is most correct. It allows:
- Importing tasks from external sources
- Rebuilding stale context bundles
- Preparing tasks created via API/JSON

---

## Gap 2: External Project Support

### Problem
Ground Control assumes all tasks are for code within the ground-control repository. There's no way to:
1. Specify which project a task belongs to
2. Tell the coder stage to work in a different directory
3. Build context bundles from external codebases

### Current State
- `context.project_id` field exists but isn't used
- Coder stage doesn't have project path awareness
- Context bundles reference local paths only

### Options to Fix

| Option | Description | Effort |
|--------|-------------|--------|
| A. `--project` flag | Add project path to orc command | Medium |
| B. Project registry | Store project definitions in `data/projects.json` | High |
| C. Task-level path | Add `working_directory` to task schema | Low |

### Recommendation
Option C (task-level working_directory) is quickest. Then expand to full project registry later.

---

## Gap 3: Task Import Workflow

### Problem
No documented workflow for importing tasks from external specifications (like NotifAI's TASKS_CAROUSEL_UI.md).

### Current State
- Tasks can be created via `gc create` (interactive)
- Tasks can be added to JSON manually (but then no context bundle)
- No `gc import` or similar command

### Ideal Workflow
```bash
# Option 1: Import from markdown
gc import --from docs/tasks.md --project /path/to/notifai

# Option 2: Import from YAML task list
gc import --from tasks.yaml --project /path/to/notifai

# Option 3: Create from template with project context
gc create --project notifai --template coding
```

---

## Gap 4: Pipeline Stage Project Awareness

### Problem
Pipeline stages (coder, reviewer, tester) don't know which project they're working on.

### Current State
- Stages assume current working directory
- No project context passed to Claude CLI
- File paths in context bundles are relative to ground-control

### Required Changes
1. Pass `working_directory` to stage execution
2. Change directory before invoking Claude CLI
3. Include project-specific CLAUDE.md in context

---

## Immediate Workarounds

Until these gaps are fixed, here are workarounds:

### Workaround 1: Create tasks via `gc create`
Run `gc create` interactively for each task. Tedious but works.

### Workaround 2: Skip sanity check
If we add `--skip-sanity` flag to `gc orc`, tasks without bundles can still run (the coder just won't have rich context).

### Workaround 3: Manual context bundles
Create the context bundle directory structure manually:
```
data/context/task_xxx/
  requirements.md    # Copy task description
  project_context.md # Copy from target project
  patterns.md        # Document coding patterns
  decisions.md       # Document design decisions
  test_hints.md      # Testing guidance
```

### Workaround 4: Run from target project
```bash
cd /path/to/notifai
/path/to/ground-control/gc orc 12
```
(Untested - may not work if gc expects its data directory)

---

## Action Items

1. [ ] Add `--skip-sanity` flag to `gc orc` (quick win)
2. [ ] Add `gc bundle <task_id>` command to build context for existing tasks
3. [ ] Add `working_directory` field to Task schema
4. [ ] Update pipeline stages to use task's working directory
5. [ ] Document the external project workflow
6. [ ] Consider `gc import` command for future

---

## Related

- Learning log entry: `notifai_orchestration_2026-03-07`
- NotifAI sprint: `NotifAI Carousel UI Phase 1-2`
- Blocked tasks: 12-20 (all NotifAI tasks)
