# Flight Deck Integration

This project is managed by **Flight Deck (FD)**, part of Ground Control (GC).

## What is Flight Deck?

Flight Deck is a central orchestration dashboard that:
- Manages multiple projects from a single TUI
- Coordinates non-coding work (docs, commits, reviews, deployments)
- Tracks issues, roadmaps, and sprints across projects
- Provides visibility into all project states
- Suggests what to work on next (AI Advisor)

## Quick Start Workflow

### Starting a New Feature
1. `/roadmap-item "Feature title" --start-now`
2. Work on the implementation
3. `/progress 50` (update periodically)
4. `/commit "Add feature X"` (when ready to commit)
5. `/complete --summary "What was done"`

### Reporting a Bug Found While Working
1. `/issue "Bug description" --priority high`
2. Continue with current work or switch: `/start-work issue_xxx`

### Checking for Dispatched Work
On session start, check `.gc/inbox/` for work items from FD:
```bash
ls .gc/inbox/
```
If items exist, review and start with `/start-work`

## Your Responsibilities (Project Claude)

As the Claude Code session for this project, you are responsible for:

### 1. Coding & Testing
You handle all code implementation and testing. Write code, run tests, debug issues.

### 2. Issue Tracking
Add issues to `.gc/issues.json` as you discover them:
```json
{
  "issues": [
    {
      "id": "issue_001",
      "title": "Bug description",
      "type": "bug",
      "priority": "high",
      "status": "open",
      "created_at": "2026-03-10T10:00:00Z"
    }
  ]
}
```

### 3. Progress Updates
Update `.gc/state.json` with session activity. The sidecar files track:
- Current session state
- Files touched
- Token/cost usage

### 4. Request FD Help
Use `.gc/requests.jsonl` when you need help with:
- Documentation written
- Code reviewed
- Commits made
- Deployments triggered
- Design decisions

## FD's Responsibilities

Flight Deck handles:
- **Commits and git operations** - FD dispatches commit requests
- **Documentation generation** - FD writes docs in flight-deck/artifacts/
- **Code reviews** - FD reviews PRs and provides feedback
- **Deployments** - FD coordinates deployment processes
- **Cross-project coordination** - FD manages dependencies between projects
- **Sprint/roadmap management** - FD tracks progress across projects

## How to Request FD Work

Append to `.gc/requests.jsonl`:

```jsonl
{"id":"req_001","type":"commit","summary":"Add auth feature","files":["src/auth.go"],"at":"2026-03-10T12:00:00Z","status":"pending"}
{"id":"req_002","type":"review","summary":"Review auth implementation","at":"2026-03-10T12:00:00Z","status":"pending"}
{"id":"req_003","type":"docs","summary":"Document auth API","at":"2026-03-10T12:00:00Z","status":"pending"}
{"id":"req_004","type":"decision","summary":"OAuth vs JWT?","options":["oauth","jwt"],"at":"2026-03-10T12:00:00Z","status":"pending"}
```

### Request Types

| Type | When to Use |
|------|-------------|
| `commit` | Code is ready to be committed |
| `review` | Want code/PR reviewed |
| `docs` | Need documentation written |
| `decision` | Need a design/architecture decision |
| `help` | Blocked and need assistance |

## Reporting Process Issues

If this workflow causes friction or problems:

1. **Note it** in `.gc/learning.jsonl`:
```jsonl
{"id":"learn_001","type":"friction","actor":"proj_cc","summary":"Commit process unclear","detail":"Wasn't sure if I should commit or wait for FD","at":"2026-03-10T12:00:00Z"}
```

2. **Continue** with a workaround if possible
3. **FD will review** learnings and improve the process

### Learning Types

| Type | When to Use |
|------|-------------|
| `friction` | Process felt awkward or unclear |
| `process_failure` | Something went wrong |
| `idea` | Improvement suggestion |

## Key Files in .gc/

| File | Purpose |
|------|---------|
| `state.json` | Your session state, costs, activity |
| `project.json` | Project config, altitude, approvals |
| `issues.json` | Project's issue/bug list |
| `roadmap.json` | Project's feature roadmap |
| `requests.jsonl` | Requests to FD (append-only) |
| `learning.jsonl` | Process improvement notes |
| `fd-onboarding.md` | This file |
| `sessions/` | Historical session records |
| `inbox/` | Work dispatched from FD |

## What You Should NOT Do

- **Don't use `gc orc`** - FD orchestrates
- **Don't use `gc delegate/handoff`** - FD coordinates
- **Don't make commits directly** - Request via requests.jsonl
- **Don't write documentation for other projects** - That's FD's job

## What You CAN Do

- Edit code in this project
- Run tests and debug
- Add issues and roadmap items to .gc/
- Request help via requests.jsonl
- Log learnings in learning.jsonl
- Use `gc tasks` to see your tasks
- Use `gc complete` to mark tasks done

## Getting Help

If you're stuck:
1. Add a request: `{"type":"help","summary":"Blocked on X","detail":"...","at":"...","status":"pending"}`
2. FD will see it on next sync and respond
3. Check `.gc/inbox/` for FD's response

## Slash Command Reference

All commands should be treated as conceptual shortcuts. Actual implementation uses `.gc/` files.

### Work Management
| Command | Description |
|---------|-------------|
| `/start-work <id>` | Start working on a task, issue, or roadmap item |
| `/progress <percent>` | Update progress on current work (0-100) |
| `/complete --summary "..."` | Mark current work complete with summary |
| `/pause --reason "..."` | Pause current work with reason |

### Issue Tracking
| Command | Description |
|---------|-------------|
| `/issue "title" [--priority high\|med\|low]` | Create new issue |
| `/bug "title" [--priority high]` | Create new bug (alias for `/issue --type bug`) |
| `/close-issue <id> --reason "..."` | Close an issue with resolution notes |

### Roadmap/Features
| Command | Description |
|---------|-------------|
| `/roadmap-item "title" [--start-now]` | Add item to roadmap, optionally start immediately |
| `/feature "title"` | Alias for `/roadmap-item` |

### Git Operations
| Command | Description |
|---------|-------------|
| `/commit "message"` | Request commit (appends to requests.jsonl) |
| `/review [--scope files\|pr]` | Request code review |

### Documentation
| Command | Description |
|---------|-------------|
| `/docs "what to document"` | Request documentation from FD |

### Help & Decisions
| Command | Description |
|---------|-------------|
| `/help "what you're stuck on"` | Request help from FD |
| `/decide "question" --options "A,B,C"` | Request architectural/design decision |

### Learning & Feedback
| Command | Description |
|---------|-------------|
| `/friction "what felt awkward"` | Log process friction (learning.jsonl) |
| `/idea "improvement suggestion"` | Log improvement idea (learning.jsonl) |

### Status & Inspection
| Command | Description |
|---------|-------------|
| `/status` | Show current work, session state |
| `/inbox` | Check for dispatched work from FD |
| `/tasks` | List all tasks for this project |

**Note**: These are conceptual commands. The actual implementation updates `.gc/` files directly. Use these as a mental model for the workflow.

## Remember

1. **You code, FD coordinates** - Focus on implementation
2. **Track issues as you find them** - Don't let bugs slip
3. **Request help when stuck** - Don't spin wheels
4. **Log friction** - Every pain point helps improve the process
5. **Check inbox** - FD may have dispatched work for you
