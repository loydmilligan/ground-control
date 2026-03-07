# Ground Control Manual Test Plan

## Overview

This document provides walkthrough scenarios to validate Ground Control's end-to-end functionality and user experience. Each scenario simulates realistic usage patterns.

---

## Pre-Test Setup

```bash
# Ensure you're in the project directory
cd /home/mmariani/Projects/ground-control

# Build the CLI
go build -o gc ./cmd/gc

# Verify it runs
./gc --help
```

**Expected**: Help text showing available commands (dump, process, create, tasks, orc, etc.)

---

## Scenario 1: Idea to Task (Brain Dump Flow)

**Goal**: Capture a quick idea and convert it to a structured task.

### Step 1.1: Capture an idea

```bash
./gc dump "Add a gc status command that shows current session state and recent activity"
```

**Expected**:
- Confirmation message that idea was captured
- New entry appears in `data/brain-dump.json`

**Check**:
```bash
cat data/brain-dump.json | jq '.[-1]'
```

### Step 1.2: View unprocessed brain dumps

```bash
./gc process --list
```

**Expected**:
- Shows the brain dump we just added
- Marked as unprocessed

### Step 1.3: Process the brain dump

```bash
./gc process
```

**Expected**:
- Interactive prompt asking about the entry
- AI analysis suggests category, complexity, requirements
- Option to approve/modify the generated task
- Context bundle is created in `data/context_bundles/`
- Task added to `data/tasks.json`

**Observe**:
- [ ] Was the analysis reasonable?
- [ ] Were the suggested requirements helpful?
- [ ] Did the flow feel smooth or clunky?

### Step 1.4: Verify the task exists

```bash
./gc tasks
```

**Expected**:
- New task appears in list with index number
- Shows title, complexity, state (created)

---

## Scenario 2: Direct Task Creation

**Goal**: Create a task directly without going through brain dump.

### Step 2.1: Create a task interactively

```bash
./gc create
```

**Expected**:
- Prompts for title, description, type, complexity
- Asks about requirements and constraints
- Builds context bundle automatically
- Shows summary before confirming

**Observe**:
- [ ] Were the prompts clear?
- [ ] Was it obvious what each field means?
- [ ] Did context bundle creation feel seamless?

### Step 2.2: Verify task and context bundle

```bash
./gc tasks
ls data/context_bundles/
```

**Expected**:
- Task appears in list
- Context bundle directory exists with requirements.md, etc.

---

## Scenario 3: Task Listing and Filtering

**Goal**: Verify task list is usable and filters work.

### Step 3.1: List active tasks

```bash
./gc tasks
```

**Expected**:
- Shows non-completed tasks by default
- Each task has an index number (1, 2, 3...)
- Shows title, type, complexity, state

### Step 3.2: List all tasks including completed

```bash
./gc tasks --all
```

**Expected**:
- Shows all tasks including completed ones
- Completed tasks are visually distinct (dimmed or marked)

### Step 3.3: Filter by state

```bash
./gc tasks --state created
```

**Expected**:
- Only shows tasks in "created" state

### Step 3.4: JSON output

```bash
./gc tasks --json
```

**Expected**:
- Valid JSON array of tasks
- Can be piped to jq for processing

**Observe**:
- [ ] Is the default view useful?
- [ ] Are indexes consistent and predictable?
- [ ] Is the information density right (not too sparse, not overwhelming)?

---

## Scenario 4: Pipeline Dry Run

**Goal**: Preview what the orchestrator will do without executing.

### Step 4.1: Check task index

```bash
./gc tasks
```

Note the index of a task you want to test (e.g., 1)

### Step 4.2: Dry run the pipeline

```bash
./gc orc --dry-run 1
```

**Expected**:
- Shows task details
- Shows pipeline stages that will run: Sanity → Coder → Reviewer → Tester → Repo Update
- Does NOT actually execute anything
- Clear indication this is a dry run

**Observe**:
- [ ] Is it clear what will happen?
- [ ] Is the pipeline flow understandable?

---

## Scenario 5: Full Pipeline Execution

**Goal**: Run a task through the complete pipeline.

> **Note**: This requires Claude Code CLI to be available. For testing without Claude, we can verify the pipeline structure runs and handles the Claude unavailability gracefully.

### Step 5.1: Run orchestrator

```bash
./gc orc -v 1
```

**Expected** (with Claude available):
- Sanity check passes (or fails with clear reason)
- Coder stage invokes Claude, shows progress
- Reviewer stage evaluates code
- If revision needed, loops back to Coder
- Tester runs tests
- Repo Update creates commit (if changes exist)
- Session summary at end

**Expected** (without Claude):
- Sanity check passes
- Coder stage fails with clear error about Claude CLI
- Error message explains what's needed

**Observe**:
- [ ] Is progress visible and understandable?
- [ ] Are errors actionable?
- [ ] Does verbose mode (-v) provide useful detail?

### Step 5.2: Check session state

```bash
ls data/sessions/
cat data/sessions/session_*.json | jq '.status'
```

**Expected**:
- Session file exists
- Status reflects outcome (completed, failed, or paused)

---

## Scenario 6: Session Resumption

**Goal**: Verify paused sessions can be resumed.

### Step 6.1: Create a situation that pauses (if possible)

This happens when:
- Max iterations reached (escalation)
- Human input required

### Step 6.2: Resume the session

```bash
./gc orc --continue
```

**Expected**:
- Finds the paused session
- Shows progress so far
- Continues from where it left off

**If no paused session**:
- Clear message saying no session to resume

---

## Scenario 7: Error Handling

**Goal**: Verify errors are helpful, not cryptic.

### Step 7.1: Run non-existent task

```bash
./gc orc 999
```

**Expected**:
- Clear error: "invalid task index: 999"
- Suggests using `gc tasks` to see valid indexes

### Step 7.2: Run task without context bundle

Create a task manually in tasks.json without a context_bundle field, then:

```bash
./gc orc <that-task-index>
```

**Expected**:
- Sanity check fails
- Error explains: "task has no context bundle"
- Suggests how to fix: "run 'gc create' or rebuild context"

### Step 7.3: Invalid command

```bash
./gc notacommand
```

**Expected**:
- Error message
- Shows available commands or suggests --help

---

## Scenario 8: TUI Mode

**Goal**: Verify the interactive TUI works and provides value over CLI.

### Step 8.1: Launch TUI

```bash
./gc tui
```

**Expected**:
- Full-screen interface appears
- Task list displayed with state icons, importance, complexity
- Help hint visible (press `?` for help)

### Step 8.2: Navigate tasks

- Use `↑`/`↓` or `j`/`k` to move through list
- Press `Enter` to view task details
- Press `Esc` to return to list

**Expected**:
- Smooth navigation
- Detail view shows full context, requirements, outputs
- Clear way to get back to list

### Step 8.3: Filter tasks

- Press `/` to enter filter mode
- Type part of a task title
- Press `Enter` to apply

**Expected**:
- List filters to matching tasks
- Clear indication filter is active

### Step 8.4: Change task state

- Select a task
- Press `s` to open state change menu
- Select a new state

**Expected**:
- State options shown clearly
- Change persists (check `data/tasks.json`)
- Visual update in list view

### Step 8.5: View help

- Press `?` for help

**Expected**:
- Keyboard shortcuts displayed
- Clear and complete

### Step 8.6: Exit

- Press `q` to quit

**Expected**:
- Clean exit, back to terminal

**Observe**:
- [ ] Is TUI faster than CLI for browsing tasks?
- [ ] Is state changing easier here than via CLI?
- [ ] Any missing functionality you expected?
- [ ] Does it feel polished or rough?

---

## Scenario 9: End-to-End Flow

**Goal**: Complete journey from idea to committed code.

### Step 8.1: Start fresh

```bash
# Capture idea
./gc dump "Add helper function to format task duration as human-readable string"

# Process it
./gc process

# Verify task
./gc tasks
```

### Step 8.2: Execute

```bash
# Run pipeline
./gc orc -v 1
```

### Step 8.3: Verify completion

```bash
# Check task state
./gc tasks --all

# Check for commit
git log -1

# Check learning log
cat data/learning-log.json | jq '.[-1]'
```

**Observe**:
- [ ] Did the entire flow feel connected?
- [ ] Were there any jarring transitions?
- [ ] Would a new user understand what's happening?

---

## UX Checklist

After running scenarios, reflect on:

### Clarity
- [ ] Are command names intuitive? (dump, process, create, tasks, orc)
- [ ] Is "orc" clear enough, or should it be "run" or "execute"?
- [ ] Are error messages actionable?

### Flow
- [ ] Does brain dump → process → execute feel natural?
- [ ] Is the relationship between tasks and context bundles clear?
- [ ] Is session state visible enough?

### Feedback
- [ ] Is progress visible during long operations?
- [ ] Are success/failure states obvious?
- [ ] Is verbose mode helpful without being overwhelming?

### Recovery
- [ ] Can you recover from errors easily?
- [ ] Is session resumption discoverable?
- [ ] Are escalations (human intervention needed) clear?

---

## Issues Found

Document any issues discovered during testing:

| Scenario | Step | Issue | Severity | Notes |
|----------|------|-------|----------|-------|
| 1 | 1.2 | `gc process --list` doesn't exist | Minor | Test plan assumed flag existed. Use `--dry-run` instead to preview. Consider adding `--list` flag for convenience. |
| 1 | 1.3 | Context bundle path inconsistent | Minor | Test plan said `data/context_bundles/` but actual path is `data/context/`. Update test plan and docs to match reality. |
| 1 | 1.3 | Output doesn't confirm bundle creation | Moderate | User has to manually check if bundle exists. Command should explicitly state "Context bundle created at: data/context/task_xxx/" after task creation. |
| - | - | Task type ignored by pipeline | **Major** | `gc orc` runs same coding pipeline for ALL task types. Planning, research, human-input tasks incorrectly go through Coder→Reviewer→Tester. Need type-based routing. |
| 5 | 5.1 | Claude CLI failure message unhelpful | Moderate | "context deadline exceeded" doesn't explain cause. Should suggest: Is Claude CLI installed? Is it in PATH? Is it authenticated? |
| 5 | 5.1 | `--context` flag doesn't exist | **Major** | Code uses `claude --context <file>` but this flag doesn't exist in Claude CLI. Fixed: now passes context via stdin. |
| 5 | 5.1 | Timeout defaulted to 0 | **Major** | Config struct didn't set timeout, defaulting to 0 = immediate failure. Fixed: now uses DefaultConfig(). |
| 5 | 5.1 | Error messages not diagnostic | Moderate | "context deadline exceeded" gave no clue about root cause. Should show: what command was run, what the actual error was, suggestions to check. |
| 5 | 5.1 | No progress during Claude execution | Moderate | 4+ minutes of blank screen while Claude works. Should show elapsed time, spinner, and streaming output from Claude. |
| 5 | 5.1 | Context bundle `relevant_code/` is empty | Moderate | Context manager not populating relevant code snippets. Claude has to read codebase itself, wasting time. Should include types, example commands, data store methods. |
| 5 | 5.1 | Reviewer output parsing fails on markdown | Moderate | Reviewer said `**APPROVED**` but parser expected `APPROVED`. Fixed: now strips markdown bold markers before parsing. |
| 5 | 5.1 | No graceful cancellation handling | Moderate | Ctrl+C leaves files in unknown state. Need: (1) mark partial work, (2) rollback option, (3) resume from checkpoint. |

---

## Recommendations

After testing, list suggested improvements:

1.
2.
3.

---

## Known Gaps (Review During Testing)

Features that are designed but not yet implemented:

### TUI Enhancements (Phase 6)
- [ ] Multi-select mode for batch operations
- [ ] Orchestration progress view (watch pipeline run)
- [ ] Brain dump processing from TUI
- [ ] Task creation from TUI

### Self-Learning System (Phase 7)
- [ ] `gc self-learn` command - analyze learning-log.json
- [ ] Pattern detection across sessions
- [ ] Proposal generation for improvements
- [ ] Human approval workflow
- [ ] Change application tracking

**Note**: Issue collection during `gc orc` DOES work - issues are written to `data/learning-log.json`. What's missing is the command to analyze and act on them.

### Other Potential Gaps
- [ ] `gc status` - show current session state
- [ ] `gc history` - view past sessions
- [ ] Better context bundle inspection

