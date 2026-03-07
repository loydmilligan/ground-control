# AI Matt Delegation System - Test Plan

## Overview

This document contains tests for the Claude ↔ AI Matt delegation system using tmux-cli for inter-pane communication.

## Prerequisites

```bash
# Verify setup
tmux-cli status                    # Should show 2 panes
./gc handoff --reset               # Clean state
./gc delegate --status             # Should show human_matt mode
```

---

## Test Suite 1: Basic Delegation Flow

### Test 1.1: Start Delegation
```bash
./gc delegate --interactions 3
```
**Expected:**
- [ ] Output shows "User: ai_matt"
- [ ] mode_count = 3
- [ ] state.json updated

**Verify:**
```bash
./gc delegate --status
cat data/delegation/state.json | jq '{user, mode_count}'
```

### Test 1.2: First Handoff (Claude → AI Matt)
```bash
./gc handoff --to-ai-matt -m "Test 1.2: First handoff. Please acknowledge."
```
**Expected:**
- [ ] Inbox written to data/delegation/inbox.md
- [ ] AI Matt pane receives notification: `[CLAUDE] New consultation #1...`
- [ ] Notification is auto-submitted (not stuck in input)
- [ ] Status shows "waiting_for_ai_matt"

**Verify:**
```bash
cat data/delegation/inbox.md
tmux-cli capture --pane=1 | grep "CLAUDE"
./gc handoff --status
```

### Test 1.3: AI Matt Response (AI Matt → Claude)
*AI Matt should:*
1. Run `gc handoff --check-inbox`
2. Write response to outbox
3. Run `gc handoff --to-claude`

**Expected:**
- [ ] Claude pane receives: `[AI_MATT] <response>`
- [ ] Response is auto-submitted
- [ ] mode_count decremented to 2
- [ ] Status back to "worker_active"

**Verify:**
```bash
./gc delegate --status
./gc handoff --history
```

### Test 1.4: Complete Full Cycle (3 interactions)
Repeat handoffs until mode_count reaches 0.

**Expected:**
- [ ] After 3rd response: user = "human_matt"
- [ ] Status = "completed"
- [ ] Message: "✓ Delegation complete - all interactions used"

---

## Test Suite 2: Message Delivery Verification

### Test 2.1: Verify Notification Reaches AI Matt
```bash
./gc delegate --interactions 1
./gc handoff --to-ai-matt -m "Test 2.1: Verify delivery"
sleep 2
tmux-cli capture --pane=1 | tail -10
```
**Expected:**
- [ ] `[CLAUDE] New consultation #1` appears in AI Matt's output
- [ ] Message was submitted (not in input box)

### Test 2.2: Verify Response Reaches Claude
*After AI Matt responds:*
```bash
tmux-cli capture --pane=0 | grep "AI_MATT"
```
**Expected:**
- [ ] `[AI_MATT]` prefix visible in Claude's input/output
- [ ] Full response text present

### Test 2.3: Verify Outbox Matches Delivered Message
```bash
# After AI Matt responds
OUTBOX=$(cat data/delegation/outbox.md)
DELIVERED=$(tmux-cli capture --pane=0 | grep -A20 "AI_MATT")
# Compare content
```
**Expected:**
- [ ] Core response content matches between outbox and delivered message

---

## Test Suite 3: Edge Cases

### Test 3.1: Handoff When Not Delegated
```bash
./gc handoff --reset
./gc handoff --to-ai-matt -m "Should fail - not delegated"
```
**Expected:**
- [ ] Message: "Not in AI Matt delegation mode. Use 'gc delegate' first."
- [ ] No notification sent to AI Matt

### Test 3.2: Handoff When mode_count = 0
```bash
./gc delegate --interactions 1
./gc handoff --to-ai-matt -m "First"
# AI Matt responds...
./gc handoff --to-ai-matt -m "Second - should fail"
```
**Expected:**
- [ ] After first response: user = "human_matt"
- [ ] Second handoff says "Not in AI Matt delegation mode"

### Test 3.3: AI Matt Response With No Inbox
```bash
./gc handoff --reset
rm -f data/delegation/inbox.md
./gc handoff --check-inbox
```
**Expected:**
- [ ] Message: "No pending consultation in inbox."

### Test 3.4: Response With No Outbox
```bash
rm -f data/delegation/outbox.md
./gc handoff --to-claude
```
**Expected:**
- [ ] Error: "no response in outbox. Write your response to ... first"

### Test 3.5: Double Response (AI Matt responds twice)
```bash
./gc delegate --interactions 2
./gc handoff --to-ai-matt -m "Test"
# AI Matt responds once
# AI Matt tries to respond again without new consultation
./gc handoff --to-claude
```
**Expected:**
- [ ] Second --to-claude should fail (inbox cleared, no new consultation)

---

## Test Suite 4: LOW Confidence Escalation

### Test 4.1: AI Matt LOW Confidence Triggers Escalation
```bash
./gc delegate --interactions 3
./gc handoff --to-ai-matt -m "Test LOW confidence handling"
```
*AI Matt writes response with:*
```markdown
## Confidence
LOW
```
*Then runs `gc handoff --to-claude`*

**Expected:**
- [ ] Message: "⚠ LOW CONFIDENCE DETECTED"
- [ ] Message: "AI Matt has LOW confidence. Pausing delegation and escalating to human."
- [ ] user = "human_matt"
- [ ] status = "escalated"
- [ ] mode_count NOT decremented (still has remaining interactions)

**Verify:**
```bash
./gc delegate --status
cat data/delegation/state.json | jq '{user, status, mode_count}'
```

### Test 4.2: HIGH Confidence Continues Normally
*AI Matt writes response with:*
```markdown
## Confidence
HIGH
```
**Expected:**
- [ ] No escalation warning
- [ ] mode_count decremented
- [ ] Delegation continues

### Test 4.3: MEDIUM Confidence Continues Normally
*AI Matt writes response with:*
```markdown
## Confidence
MEDIUM
```
**Expected:**
- [ ] No escalation (only LOW triggers)
- [ ] Delegation continues

---

## Test Suite 5: State Management

### Test 5.1: Reset Clears Everything
```bash
./gc delegate --interactions 5
./gc handoff --to-ai-matt -m "Test"
./gc handoff --reset
```
**Expected:**
- [ ] state.json reset: user = "human_matt", mode_count = 0
- [ ] inbox.md deleted
- [ ] outbox.md deleted
- [ ] Message: "✓ Handoff state reset to clean"

### Test 5.2: Cancel Preserves History
```bash
./gc delegate --interactions 5
./gc handoff --to-ai-matt -m "Test 1"
# AI Matt responds
./gc handoff --to-ai-matt -m "Test 2"
# AI Matt responds
./gc delegate --cancel
```
**Expected:**
- [ ] user = "human_matt"
- [ ] handoff_count preserved in state
- [ ] history.jsonl contains records of handoffs

**Verify:**
```bash
./gc handoff --history
cat data/delegation/history.jsonl
```

### Test 5.3: Diagnose Detects Issues
```bash
# Create inconsistent state
./gc delegate --interactions 3
./gc handoff --to-ai-matt -m "Test"
# Manually delete inbox
rm data/delegation/inbox.md
./gc handoff --diagnose
```
**Expected:**
- [ ] Diagnose detects: "State is 'waiting_for_ai_matt' but inbox is empty"
- [ ] Suggests fix

---

## Test Suite 6: Task-Based Delegation

### Test 6.1: Delegate Until Tasks Complete
```bash
./gc delegate --tasks task_001,task_002
./gc delegate --status
```
**Expected:**
- [ ] interaction_mode = "tasks"
- [ ] target_tasks = ["task_001", "task_002"]

### Test 6.2: Task Completion Tracking
*This requires integration with task system - placeholder for future*

---

## Test Suite 7: History & Audit

### Test 7.1: History Records All Handoffs
```bash
./gc handoff --reset
./gc delegate --interactions 2
./gc handoff --to-ai-matt -m "First"
# AI Matt responds
./gc handoff --to-ai-matt -m "Second"
# AI Matt responds
./gc handoff --history
```
**Expected:**
- [ ] Shows 4 entries (2 to AI Matt, 2 to Claude)
- [ ] Each entry has timestamp, from, to

### Test 7.2: History Persists Across Sessions
```bash
# After running tests above
./gc handoff --reset
./gc handoff --history
```
**Expected:**
- [ ] History still shows previous entries (reset doesn't clear history)

---

## Test Suite 8: Stress & Recovery

### Test 8.1: Rapid Handoffs
```bash
./gc delegate --interactions 5
for i in 1 2 3 4 5; do
  ./gc handoff --to-ai-matt -m "Rapid test $i"
  sleep 2
  # AI Matt auto-responds
  sleep 2
done
```
**Expected:**
- [ ] All 5 handoffs complete successfully
- [ ] No race conditions or state corruption

### Test 8.2: Recovery After Pane Crash
```bash
./gc delegate --interactions 3
./gc handoff --to-ai-matt -m "Test"
# Kill AI Matt pane
tmux-cli kill --pane=1
# Recreate pane, restart claude
./gc handoff --diagnose
```
**Expected:**
- [ ] Diagnose shows waiting_for_ai_matt
- [ ] Can recover by: `./gc handoff --reset` or recreating AI Matt

### Test 8.3: Handoff Limit (20 max)
```bash
./gc delegate --interactions 25
# Run 20 handoffs...
```
**Expected:**
- [ ] After 20 handoffs: automatic pause
- [ ] Message about reaching maximum handoff limit

---

## Quick Smoke Test

Run this for a quick end-to-end verification:

```bash
# 1. Reset
./gc handoff --reset

# 2. Start delegation
./gc delegate --interactions 2

# 3. First handoff
./gc handoff --to-ai-matt -m "Smoke test: Please respond with APPROVED and HIGH confidence"

# 4. Wait for AI Matt
sleep 10

# 5. Check AI Matt responded
./gc delegate --status  # Should show mode_count = 1

# 6. Second handoff
./gc handoff --to-ai-matt -m "Final smoke test: Please respond to complete delegation"

# 7. Wait and verify
sleep 10
./gc delegate --status  # Should show human_matt mode

# 8. Check history
./gc handoff --history
```

---

## Test Results Log

| Test | Date | Result | Notes |
|------|------|--------|-------|
| 1.1 | | | |
| 1.2 | | | |
| 1.3 | | | |
| ... | | | |

---

## Known Issues

1. **Pane IDs are environment-specific**: Default pane IDs (0:1.0, 0:1.1) may not match your tmux layout. Update state.json if needed.

2. **tmux-cli send timing**: If messages appear but aren't submitted, the Enter key timing may be off. Use `--delay-enter=0.5` (now default in code).

3. **Long messages may truncate**: Very long responses might have issues with tmux-cli send. Consider testing with varying message lengths.
