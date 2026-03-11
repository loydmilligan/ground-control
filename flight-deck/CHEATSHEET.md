# Flight Deck Cheat Sheet

Quick reference for all FD commands and workflows.

## CLI Commands

```bash
gc fd                    # Launch Flight Deck TUI
gc adopt <path>          # Adopt project into FD
gc adopt <path> --force  # Re-adopt (creates backup first)
gc sync                  # Aggregate all project states
gc sync -p <project>     # Sync single project
gc sync -q               # Quiet mode (no output)
```

## Project Slash Commands

Use these when working in an adopted project's Claude session.

### Work Management

| Command | Example | Purpose |
|---------|---------|---------|
| `/roadmap-item` | `/roadmap-item "User auth"` | Add feature to roadmap |
| `/roadmap-item --start-now` | `/roadmap-item "Auth" --start-now` | Add and begin immediately |
| `/start-work` | `/start-work feat_123` | Begin work on item |
| `/progress` | `/progress 60` | Update completion % |
| `/complete` | `/complete --summary "Done"` | Mark item complete |

### Issues & Bugs

| Command | Example | Purpose |
|---------|---------|---------|
| `/issue` | `/issue "Login broken"` | Report a bug |
| `/issue --priority` | `/issue "Critical" --priority high` | High priority bug |
| `/issue --blocks` | `/issue "Blocker" --blocks feat_123` | Blocking issue |

### Git Operations

| Command | Example | Purpose |
|---------|---------|---------|
| `/commit` | `/commit "Add feature X"` | Request commit via FD |
| `/complete --commit` | `/complete --commit "Done"` | Complete + request commit |

### Learning & Feedback

| Command | Example | Purpose |
|---------|---------|---------|
| `/learn friction` | `/learn friction "Unclear process"` | Log process friction |
| `/learn idea` | `/learn idea "Auto-sync"` | Log improvement idea |
| `/learn success` | `/learn success "Workflow smooth"` | Log what worked |

## FD Claude Commands

Use these in the Flight Deck Claude session.

| Command | Purpose |
|---------|---------|
| `/advisor` | Get work recommendations |
| `/sync` | Run gc sync |
| `/dispatch` | Send work to project |
| `/review-learning` | Triage learning entries |

## Key Files

### Per-Project (`.gc/`)

| File | Purpose | Format |
|------|---------|--------|
| `CLAUDE.md` | Project Claude instructions | Markdown |
| `fd-onboarding.md` | FD integration guide | Markdown |
| `project.json` | Project config (tech stack, altitude) | JSON |
| `state.json` | Session state (focus, activity) | JSON |
| `roadmap.json` | Feature roadmap | JSON |
| `issues.json` | Bugs and issues | JSON |
| `requests.jsonl` | Requests to FD | JSONL |
| `learning.jsonl` | Process learnings | JSONL |
| `inbox/` | Work dispatched from FD | Directory |

### Global (`~/.gc/`)

| File | Purpose |
|------|---------|
| `global.json` | Registry of all projects |
| `aggregated.json` | Synced state from all projects |

## Common Workflows

### Start New Feature
```
/roadmap-item "Feature title" --start-now
# Work on implementation...
/progress 30
# Continue working...
/progress 70
# Ready to commit
/commit "Implement feature"
/complete --summary "Feature complete"
```

### Fix a Bug
```
/issue "Bug description" --priority high
/start-work issue_xxx
# Fix the bug...
/commit "Fix: bug description"
/complete
```

### Report Process Issue
```
/learn friction "The X workflow was confusing because Y"
```

### Check for Dispatched Work
```bash
ls .gc/inbox/
# If files exist, review and start work
```

## Prompt Files

All prompts in `flight-deck/prompts/`:

| Prompt | Used By | Purpose |
|--------|---------|---------|
| `roadmap-item.md` | Project | Add feature to roadmap |
| `start-work.md` | Project | Begin work on item |
| `progress-update.md` | Project | Update completion % |
| `complete-work.md` | Project | Mark item complete |
| `issue-management.md` | Project | Manage issues |
| `request-to-fd.md` | Project | Request FD help |
| `learning-capture.md` | Project | Log learnings |
| `dispatch-work.md` | FD | Send work to project |
| `advisor-recommendation.md` | FD | Work recommendations |
| `learning-review.md` | FD | Triage learnings |
| `attention-flags.md` | FD | Manage attention flags |
| `fd-design-workflow.md` | FD | Design work process |
| `fd-review-workflow.md` | FD | Code review process |
| `fd-documentation-workflow.md` | FD | Documentation workflow |
| `roadmap-management.md` | FD | Manage roadmaps |

## TUI Keybindings

| Key | Action |
|-----|--------|
| `s` | Start Claude session |
| `t` | Teleport to session |
| `x` | Stop session |
| `m` | Send message to session |
| `A` | Cycle altitude (low/mid/high) |
| `S` | Cycle session mode |
| `F12` | Return to Flight Deck |
| `tab` | Switch views |
| `q` | Quit |
