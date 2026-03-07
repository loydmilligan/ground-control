# AI Matt Delegation System

## Overview

This system allows the human user (Matt) to delegate decision-making to AI Matt for a specified number of interactions. Two Claude Code sessions run in tmux panes and communicate via files.

## Quick Start

```bash
# 1. Run setup script
./scripts/ai-matt-setup.sh

# 2. Attach to tmux session
tmux attach -t gc

# 3. In LEFT pane, start Claude and begin delegation
claude
# (then in Claude): gc delegate --interactions 5

# 4. Work normally. Handoffs happen automatically.
```

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                            tmux session: gc                       │
│                                                                    │
│  ┌────────────────────────┐      ┌────────────────────────────┐  │
│  │  Pane 0: Claude        │      │  Pane 1: AI Matt           │  │
│  │  (Worker Agent)        │      │  (Decision Agent)          │  │
│  │                        │      │                            │  │
│  │  - Works on tasks      │      │  - Reads consultations     │  │
│  │  - Writes to inbox     │      │  - Makes decisions         │  │
│  │  - Reads from outbox   │      │  - Writes to outbox        │  │
│  │  - Continues working   │      │  - Waits for next consult  │  │
│  └────────────────────────┘      └────────────────────────────┘  │
│              │                              ▲                      │
│              │    data/delegation/          │                      │
│              │   ┌─────────────────┐        │                      │
│              └──►│  inbox.md       │────────┘                      │
│                  │  outbox.md      │◄────────                      │
│                  │  state.json     │                               │
│                  └─────────────────┘                               │
└──────────────────────────────────────────────────────────────────┘
```

## File Structure

```
data/delegation/
├── state.json      # Shared state (who's active, mode, count)
├── inbox.md        # Claude → AI Matt messages
├── outbox.md       # AI Matt → Claude messages
└── history.jsonl   # Log of all handoffs for debugging
```

## State Schema

```json
{
  "user": "human_matt",
  "interaction_mode": "interactions",
  "mode_count": 0,
  "target_tasks": [],
  "started_at": null,
  "last_handoff_at": null,
  "handoff_count": 0,
  "status": "idle",
  "error": null
}
```

**Fields:**
- `user`: `"human_matt"` | `"ai_matt"` - who receives end-of-action reports
- `interaction_mode`: `"interactions"` | `"tasks"` - what mode_count means
- `mode_count`: positive integer, decremented after each AI Matt response
- `target_tasks`: array of task IDs (for task mode)
- `started_at`: ISO timestamp when delegation started
- `last_handoff_at`: ISO timestamp of last handoff
- `handoff_count`: total handoffs this delegation session
- `status`: `"idle"` | `"worker_active"` | `"waiting_for_ai_matt"` | `"error"`
- `error`: error message if status is "error"

---

## Setup Instructions

### 1. Start tmux session

```bash
# Create new tmux session named 'gc'
tmux new-session -d -s gc -n main

# Split into two panes (left: Claude, right: AI Matt)
tmux split-window -h -t gc:main

# Select left pane for Claude
tmux select-pane -t gc:main.0
```

### 2. Start Claude (Worker Agent) - Pane 0

```bash
# In left pane
cd /home/mmariani/Projects/ground-control
claude
```

When Claude starts, give it this context:
```
You are working in AI Matt delegation mode. Before ending each action:
1. Run: cat data/delegation/state.json
2. If user="ai_matt" and mode_count > 0:
   - Write your summary to data/delegation/inbox.md
   - Run: gc handoff --to-ai-matt
   - STOP and wait
3. If user="human_matt" or mode_count <= 0:
   - End normally, wait for human input
```

### 3. Start AI Matt (Decision Agent) - Pane 1

```bash
# In right pane
cd /home/mmariani/Projects/ground-control
claude --system-prompt agents/ai-matt.md
```

When AI Matt starts, give it this context:
```
You are AI Matt running in delegation mode. Your workflow:
1. Wait for notification that inbox.md has content
2. Run: gc handoff --check-inbox
3. If there's a consultation:
   - Read the summary in data/delegation/inbox.md
   - Make your decision using your AI Matt guidelines
   - Write your response to data/delegation/outbox.md
   - Run: gc handoff --to-claude
4. Wait for next consultation
```

### 4. Attach to session

```bash
tmux attach -t gc
```

Use `Ctrl-b` + arrow keys to switch between panes.

---

## Delegation Commands

### Start delegation
```bash
# Delegate for N interactions
gc delegate --interactions 5

# Delegate until specific tasks complete
gc delegate --tasks task_xxx,task_yyy

# Check status
gc delegate --status

# Cancel
gc delegate --cancel
```

### Handoff commands (used by agents)
```bash
# Claude: signal handoff to AI Matt
gc handoff --to-ai-matt

# AI Matt: check for pending consultations
gc handoff --check-inbox

# AI Matt: signal response ready for Claude
gc handoff --to-claude

# Claude: check for AI Matt's response
gc handoff --check-outbox

# Either: check current state
gc handoff --status
```

---

## Communication Protocol

### Claude's Post-Action Check

```
┌─────────────────────────────────────────────────────────┐
│ END OF ACTION                                           │
│                                                         │
│ 1. Read data/delegation/state.json                      │
│                                                         │
│ 2. IF user = "human_matt" OR mode_count <= 0:           │
│    └─► Stop. Wait for human input. (normal behavior)   │
│                                                         │
│ 3. IF user = "ai_matt" AND mode_count > 0:              │
│    a. Write summary to data/delegation/inbox.md         │
│    b. Run: gc handoff --to-ai-matt                      │
│    c. Output: "═══ HANDED OFF TO AI MATT ═══"           │
│    d. STOP. Wait for outbox.md to be populated.         │
└─────────────────────────────────────────────────────────┘
```

### AI Matt's Response Flow

```
┌─────────────────────────────────────────────────────────┐
│ CONSULTATION RECEIVED                                   │
│                                                         │
│ 1. Run: gc handoff --check-inbox                        │
│    └─► Shows inbox content and state                    │
│                                                         │
│ 2. Read data/delegation/inbox.md                        │
│                                                         │
│ 3. Make decision using AI Matt guidelines               │
│    - Consider confidence level                          │
│    - If LOW confidence: set error state, alert human    │
│                                                         │
│ 4. Write response to data/delegation/outbox.md          │
│    Format:                                              │
│    ## Decision                                          │
│    [CONTINUE | PAUSE | CHANGE_DIRECTION | ESCALATE]     │
│                                                         │
│    ## Response                                          │
│    [Your guidance for Claude]                           │
│                                                         │
│    ## Confidence                                        │
│    [HIGH | MEDIUM | LOW]                                │
│                                                         │
│ 5. Run: gc handoff --to-claude                          │
│    └─► Updates state, decrements mode_count             │
│                                                         │
│ 6. Output: "═══ RESPONSE SENT TO CLAUDE ═══"            │
│                                                         │
│ 7. Wait for next consultation                           │
└─────────────────────────────────────────────────────────┘
```

### Claude Receives Response

```
┌─────────────────────────────────────────────────────────┐
│ CHECKING FOR AI MATT RESPONSE                           │
│                                                         │
│ 1. Run: gc handoff --check-outbox                       │
│    └─► Shows outbox content if available                │
│                                                         │
│ 2. IF outbox has response:                              │
│    a. Read data/delegation/outbox.md                    │
│    b. Parse decision and guidance                       │
│    c. Clear outbox (mark as processed)                  │
│    d. Continue working based on AI Matt's direction     │
│                                                         │
│ 3. IF no response yet:                                  │
│    └─► Wait and check again                             │
└─────────────────────────────────────────────────────────┘
```

---

## Error Scenarios & Recovery

### E1: AI Matt doesn't respond (timeout)

**Detection:** Claude checks outbox, finds nothing after reasonable time.

**Recovery:**
```bash
# Check state
gc handoff --status

# If stuck in "waiting_for_ai_matt" for too long:
gc delegate --cancel
# This resets to human_matt mode

# Optionally restart AI Matt pane:
tmux send-keys -t gc:main.1 "C-c"  # Ctrl-C to AI Matt
tmux send-keys -t gc:main.1 "claude --system-prompt agents/ai-matt.md" Enter
```

**Prevention:** AI Matt should always respond, even if just to escalate.

### E2: File corruption / parse error

**Detection:** JSON parse fails or required fields missing.

**Recovery:**
```bash
# Reset delegation state
gc delegate --reset

# This creates fresh state.json:
# {"user": "human_matt", "interaction_mode": "interactions",
#  "mode_count": 0, "status": "idle", ...}
```

### E3: State inconsistency (both think they're waiting)

**Detection:** status="waiting_for_ai_matt" but inbox is empty, or outbox has unread response.

**Recovery:**
```bash
# Check all files
gc handoff --diagnose

# Output shows:
# State: waiting_for_ai_matt
# Inbox: empty (PROBLEM: should have content)
# Outbox: has content (PROBLEM: unprocessed)
#
# Suggested fix: gc handoff --to-claude

# Apply suggested fix
gc handoff --to-claude
```

### E4: Session/pane crash

**Detection:** tmux pane is dead or unresponsive.

**Recovery:**
```bash
# Check panes
tmux list-panes -t gc

# If pane missing, recreate:
tmux split-window -h -t gc:main

# Restart the appropriate agent
# For Claude (pane 0):
tmux send-keys -t gc:main.0 "cd /home/mmariani/Projects/ground-control && claude" Enter

# For AI Matt (pane 1):
tmux send-keys -t gc:main.1 "cd /home/mmariani/Projects/ground-control && claude --system-prompt agents/ai-matt.md" Enter

# Resume from last known state
gc handoff --status
```

### E5: AI Matt returns LOW confidence

**Behavior:** This automatically pauses delegation.

**What happens:**
```bash
# AI Matt sets in outbox.md:
## Confidence
LOW

## Decision
ESCALATE

# gc handoff --to-claude detects LOW confidence:
# - Sets user = "human_matt"
# - Sets status = "escalated"
# - Does NOT decrement mode_count
# - Alerts human
```

**Human resumes:**
```bash
# Review what happened
gc handoff --history

# Either resume delegation
gc delegate --interactions 3

# Or stay in human mode and address the issue directly
```

### E6: Infinite loop / runaway

**Prevention:** Hard limit of 20 handoffs per delegation session.

**Detection:** handoff_count >= 20

**Behavior:**
```bash
# Automatic pause with message:
"Delegation paused: reached maximum handoff limit (20).
Review progress with: gc handoff --history"
```

---

## History & Debugging

All handoffs are logged to `data/delegation/history.jsonl`:

```json
{"timestamp": "2024-01-15T10:30:00Z", "from": "claude", "to": "ai_matt", "summary_preview": "Finished implementing...", "handoff_number": 1}
{"timestamp": "2024-01-15T10:31:00Z", "from": "ai_matt", "to": "claude", "decision": "CONTINUE", "confidence": "HIGH", "handoff_number": 1}
```

View history:
```bash
gc handoff --history          # Show recent handoffs
gc handoff --history --full   # Show full content
```

---

## Quick Reference

### For Human (Matt)

```bash
# Start delegation
gc delegate --interactions 5

# Check what's happening
gc delegate --status
gc handoff --status

# Something wrong?
gc handoff --diagnose

# Take back control
gc delegate --cancel

# Review what happened
gc handoff --history
```

### For Claude (Worker)

```
At end of each action:
1. cat data/delegation/state.json
2. If user="ai_matt" && mode_count > 0:
   - Write summary to inbox.md
   - gc handoff --to-ai-matt
   - STOP
3. To check for AI Matt's response:
   - gc handoff --check-outbox
   - If response exists, continue working
```

### For AI Matt (Decision Agent)

```
When notified of consultation:
1. gc handoff --check-inbox
2. Read inbox.md, make decision
3. Write to outbox.md with Decision/Response/Confidence
4. gc handoff --to-claude
5. Wait for next consultation
```

---

## Safety Guarantees

1. **Human always has control**: `gc delegate --cancel` works instantly
2. **Low confidence = pause**: AI Matt LOW confidence auto-escalates to human
3. **Hard limits**: Max 20 handoffs per delegation session
4. **Full audit trail**: Every handoff logged with timestamps
5. **Graceful degradation**: Any error defaults to human_matt mode
6. **No silent failures**: All errors produce visible output

---

## Implementation Status

- [ ] `gc delegate` command (basic version exists, needs update)
- [ ] `gc handoff` command (new)
- [ ] State file management
- [ ] Inbox/outbox file management
- [ ] History logging
- [ ] Error recovery commands
- [ ] Claude post-action check integration
- [ ] AI Matt session setup script
- [ ] tmux setup script
