# Ground Control System Review

**Date**: 2026-03-08
**Purpose**: Deep review for security assessment before adding web UI
**Status**: DRAFT - Nothing decided about next phase

---

## Executive Summary

Ground Control is a local-first, CLI-based task orchestration system where AI agents (Claude) manage task flow. The system currently runs entirely on localhost with JSON file storage and relies on Claude Code CLI for AI execution.

**Proposed next phase**: Add a web UI behind authentication (go-better-auth) and expose via Cloudflare Tunnel. This document catalogs the current state to support a security review before proceeding.

---

## Current Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        LOCAL MACHINE                             │
│                                                                  │
│  ┌──────────┐    ┌──────────┐    ┌───────────────────────────┐  │
│  │  gc CLI  │───>│ Go App   │───>│ data/*.json (filesystem)  │  │
│  └──────────┘    └──────────┘    └───────────────────────────┘  │
│       │                │                                         │
│       │                v                                         │
│       │         ┌──────────────┐                                │
│       │         │ Claude Code  │ (subprocess via CLI)           │
│       │         │    CLI       │                                │
│       └────────>│              │                                │
│                 └──────────────┘                                │
│                        │                                         │
│                        v                                         │
│                 ┌──────────────┐                                │
│                 │ Anthropic    │ (API calls to cloud)           │
│                 │   API        │                                │
│                 └──────────────┘                                │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Description | Risk Surface |
|-----------|-------------|--------------|
| `gc` CLI binary | Go binary, all commands | Process execution |
| `data/*.json` | Task, project, session data | File read/write |
| `claude` subprocess | Claude Code CLI integration | Shell execution, API calls |
| tmux integration | Multi-pane orchestration | Process control |
| Scripts (`scripts/`) | Bash helpers for delegation | Shell execution |

---

## Complete Feature Inventory

### CLI Commands

| Command | Description | Security Notes |
|---------|-------------|----------------|
| `gc tasks` | List/filter tasks | Read-only |
| `gc dump "idea"` | Capture brain dump | Write to brain-dump.json |
| `gc create` | Create task interactively | Write to tasks.json |
| `gc process` | Convert brain dumps to tasks | Read/write JSON |
| `gc orc [task]` | Run orchestration pipeline | **Spawns Claude subprocess** |
| `gc complete` | Mark task complete | Write + optional command execution |
| `gc status` | Show session state | Read-only |
| `gc standup` | Daily standup ritual | Read-only |
| `gc sprint` | Sprint management | Read/write |
| `gc brainstorm` | Ideation session | Write |
| `gc delegate` | Delegate to AI Matt | **Spawns Claude, tmux** |
| `gc handoff` | Agent handoff signals | Read/write delegation state |
| `gc consult` | Consult specialized agent | **Spawns Claude** |
| `gc sessions` | Manage orchestration sessions | Read/write, can cancel |
| `gc history` | View past sessions | Read-only |
| `gc self-learn` | Analyze learning patterns | Read-only |
| `gc ingest` | Parse Claude logs | Read external files |
| `gc tui` | Interactive terminal UI | All of above |

### AI Agent Execution

The system spawns Claude Code CLI as a subprocess:

```go
// internal/claude/claude.go
cmd := exec.Command("claude",
    "--print", "text",
    "--permission-mode", c.config.PermissionMode,
    "-p", prompt,
)
```

**Permission modes supported**:
- `default` - Normal Claude permissions (asks for approval)
- `acceptEdits` - Auto-accept file edits
- `bypassPermissions` - Skip all permission checks (**dangerous**)

### Delegation System

AI Matt delegation creates a multi-agent setup:

1. **Worker Claude**: Executes tasks
2. **AI Matt**: Makes decisions as user proxy
3. **Monitor**: Watches communication (optional)
4. **Watchdog**: Detects stuck states

**Current protections**:
- Password-protected approvals (SHA256 hash in env)
- AI Matt runs in restricted environment (`.claude-ai-matt/`)
- Pre-tool hooks block dangerous operations
- Read-only mode for AI Matt

### Data Storage

All data in `data/` directory as JSON:

| File | Contents | Sensitivity |
|------|----------|-------------|
| `tasks.json` | All task definitions | Medium - contains project details |
| `projects.json` | Project definitions | Low |
| `brain-dump.json` | Raw ideas | Low |
| `activity-log.json` | Audit trail, AI training | Medium - decision history |
| `agents.json` | Agent definitions | Low |
| `sessions/` | Orchestration session state | Medium |
| `delegation/` | Delegation state, inbox/outbox | High - contains decisions |
| `learning-log.json` | AI learning patterns | Medium |
| `ai-matt-insights.json` | Personality data | Low |

### Scripts

| Script | Purpose | Risk |
|--------|---------|------|
| `delegation-monitor.sh` | Watch delegation communication | Terminal control |
| `delegation-watchdog.sh` | Detect stuck states | Auto-submit text to Claude |
| `start-delegation-session.sh` | Create tmux session | Process spawning |
| `stop-delegation-session.sh` | Cleanup tmux | Process killing |
| `start-ai-matt.sh` | Launch AI Matt | Claude subprocess |
| `gc-hash-password.sh` | Generate password hash | Credential handling |
| `setup-ai-matt-env.sh` | Create restricted env | File system modification |

---

## Security Assessment: Current State

### Strengths

1. **Local-first**: No network exposure currently
2. **File-based storage**: No database attack surface
3. **Subprocess isolation**: Claude runs as separate process
4. **Password protection**: Delegation approvals require password
5. **AI Matt restrictions**: Read-only, blocked operations
6. **Audit logging**: Activity log tracks decisions

### Weaknesses / Risks

1. **Shell execution**: Multiple paths execute shell commands
   - `gc complete` can run verification commands
   - Claude subprocess can execute arbitrary code
   - Scripts use `eval`, `tmux send-keys`

2. **Trust model unclear**:
   - Claude Code has full filesystem access by default
   - `bypassPermissions` mode available
   - No sandboxing beyond Claude's own safeguards

3. **Credential storage**:
   - Password hash in `.env` file
   - No encryption at rest for data files

4. **Process control**:
   - tmux send-keys can inject commands
   - Watchdog auto-submits pending text

5. **No authentication**: CLI assumes trusted local user

6. **No rate limiting**: No limits on Claude API calls

7. **Prompt injection surface**:
   - Task descriptions flow to Claude prompts
   - Brain dumps become agent input
   - Malicious task content could manipulate agent

---

## Proposed Next Phase: Web UI

**NOT DECIDED - For discussion only**

### Concept

Add a web interface to Ground Control:
- Authentication via go-better-auth
- Exposed via Cloudflare Tunnel
- View/manage tasks, sessions, delegation
- Potentially trigger orchestration remotely

### Initial Thoughts (Claude's perspective)

**Benefits**:
- Mobile/remote access to task management
- Better visibility into multi-agent sessions
- Easier monitoring of delegation
- Could enable notifications/webhooks

**Concerns requiring careful design**:

1. **Authentication boundary**:
   - go-better-auth handles user auth
   - But what authorizes Claude subprocess execution?
   - Need clear separation: "view tasks" vs "run orchestration"

2. **Cloudflare Tunnel exposure**:
   - Tunnel itself is secure (no port exposure)
   - But web app becomes the trust boundary
   - Any web vuln = potential code execution

3. **Session hijacking risk**:
   - If session tokens compromised, attacker could trigger Claude
   - Need tight session controls, maybe require re-auth for dangerous ops

4. **API surface**:
   - REST API for web UI creates new attack surface
   - Need input validation, rate limiting, CSRF protection

5. **Real-time features**:
   - WebSocket for live session monitoring?
   - Additional complexity and attack surface

### Potential Lockdown Measures

**Before adding web UI**:

1. **Audit command execution paths**:
   - Map every path from input to shell execution
   - Add explicit allow-lists where possible
   - Consider removing `bypassPermissions` mode entirely

2. **Sandbox Claude execution**:
   - Container/VM for Claude subprocess?
   - Restrict filesystem access to `data/` only?
   - Network isolation (block outbound except Anthropic API)?

3. **Encrypt sensitive data**:
   - Encrypt delegation inbox/outbox
   - Encrypt activity log (contains decision patterns)

4. **Add operation tiers**:
   - Tier 1: Read-only (view tasks, history) - basic auth
   - Tier 2: Write (create tasks, brain dump) - auth + maybe MFA
   - Tier 3: Execute (run orchestration) - auth + MFA + rate limit
   - Tier 4: Admin (bypass permissions) - disabled remotely

5. **Implement audit logging**:
   - Log all web requests
   - Log all Claude subprocess invocations
   - Alerting on suspicious patterns

6. **Rate limiting**:
   - Limit orchestration runs per hour
   - Limit delegation sessions
   - Cost controls on Claude API usage

7. **Network architecture**:
   ```
   Internet → Cloudflare Tunnel → go-better-auth → Web API → gc (restricted)
                                       ↓
                                  [Only Tier 1-2 ops allowed remotely?]
   ```

8. **Consider split architecture**:
   - Web UI for viewing/managing tasks only
   - Orchestration still requires local terminal access
   - Reduces blast radius significantly

---

## Questions for Security Review

1. What's the threat model? (Curious outsider? Targeted attack? Insider?)

2. Should orchestration (Claude execution) ever be remotely triggerable?

3. What's the acceptable risk for prompt injection via task content?

4. Should AI Matt delegation be available via web UI, or local-only?

5. What authentication factors are appropriate? (Password? MFA? Hardware key?)

6. Should there be separate "admin" and "operator" roles?

7. Is Cloudflare Tunnel the right approach, or VPN/Tailscale instead?

8. What monitoring/alerting is needed?

9. Budget for security measures? (Affects tooling choices)

10. Timeline constraints? (Affects depth of hardening possible)

---

## File Manifest

For reference, complete list of files in the repository:

### Core Application
```
cmd/gc/main.go              # CLI entry point
internal/cmd/*.go           # Command implementations
internal/claude/claude.go   # Claude CLI integration
internal/data/data.go       # Data store operations
internal/pipeline/*.go      # Orchestration pipeline stages
internal/tui/tui.go         # Terminal UI
internal/types/types.go     # Type definitions
```

### Data (gitignored sensitive content)
```
data/tasks.json
data/projects.json
data/brain-dump.json
data/activity-log.json
data/agents.json
data/rituals.json
data/sessions/*.json
data/delegation/state.json
data/delegation/inbox.md
data/delegation/outbox.md
data/delegation/history.jsonl
data/learning-log.json
data/ai-matt-insights.json
data/context/*/            # Task context bundles
```

### Scripts
```
scripts/delegation-monitor.sh
scripts/delegation-watchdog.sh
scripts/start-delegation-session.sh
scripts/stop-delegation-session.sh
scripts/start-ai-matt.sh
scripts/gc-hash-password.sh
scripts/setup-ai-matt-env.sh
```

### Agent Prompts
```
agents/ai-matt.md
agents/coder.md
agents/context-engineer.md
agents/ingestion.md
agents/planner.md
agents/researcher.md
agents/reviewer.md
agents/taskmaster.md
agents/tester.md
```

### Configuration
```
.claude-ai-matt/            # Restricted AI Matt environment
  CLAUDE.md
  settings.json
  hooks/pre_tool_use.sh
```

### Documentation
```
docs/architecture.md
docs/decisions.md
docs/command-design.md
docs/ai-matt-delegation-system.md
docs/pipeline-architecture.md
CLAUDE.md                   # AI agent instructions
README.md
CHANGELOG.md
LICENSE
```

---

## Repository

**GitHub**: https://github.com/loydmilligan/ground-control (private)

---

*This document is a snapshot for security review. No decisions have been made about implementation approach for the web UI phase.*
