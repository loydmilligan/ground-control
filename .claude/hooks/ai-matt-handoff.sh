#!/bin/bash
# AI Matt Handoff Hook
#
# This hook runs after Claude finishes responding (Stop event).
# It checks if delegation is active and notifies AI Matt if needed.
#
# Usage: Called automatically by Claude Code's Stop hook
#
# Environment:
#   CLAUDE_PROJECT_DIR - project directory
#   stdin - JSON with last_assistant_message
#
# Exit codes:
#   0 - Success (normal or handoff initiated)
#   1 - Error (logged but doesn't block)

set -e

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(pwd)}"
STATE_FILE="$PROJECT_DIR/data/delegation/state.json"
INBOX_FILE="$PROJECT_DIR/data/delegation/inbox.md"
GC_BIN="$PROJECT_DIR/gc"

# Read input from stdin
INPUT=$(cat)

# Check if state file exists
if [[ ! -f "$STATE_FILE" ]]; then
    exit 0
fi

# Parse state
USER=$(jq -r '.user // "human_matt"' "$STATE_FILE" 2>/dev/null)
MODE_COUNT=$(jq -r '.mode_count // 0' "$STATE_FILE" 2>/dev/null)
STATUS=$(jq -r '.status // "idle"' "$STATE_FILE" 2>/dev/null)
TMUX_SESSION=$(jq -r '.tmux_session // "gc"' "$STATE_FILE" 2>/dev/null)
AI_MATT_PANE=$(jq -r '.ai_matt_pane // "0.1"' "$STATE_FILE" 2>/dev/null)

# If not delegated to AI Matt, do nothing
if [[ "$USER" != "ai_matt" ]] || [[ "$MODE_COUNT" -le 0 ]]; then
    exit 0
fi

# If already waiting for AI Matt, do nothing
if [[ "$STATUS" == "waiting_for_ai_matt" ]]; then
    exit 0
fi

# Extract last assistant message from input
LAST_MESSAGE=$(echo "$INPUT" | jq -r '.last_assistant_message // ""' 2>/dev/null)

if [[ -z "$LAST_MESSAGE" ]]; then
    exit 0
fi

# Write to inbox
mkdir -p "$(dirname "$INBOX_FILE")"
cat > "$INBOX_FILE" << EOF
# Consultation from Claude

**Time**: $(date -Iseconds)
**Handoff via**: Stop hook

## Summary

$LAST_MESSAGE
EOF

# Update state
jq '.status = "waiting_for_ai_matt" | .last_handoff_at = "'"$(date -Iseconds)"'" | .handoff_count = (.handoff_count + 1)' \
    "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"

# Notify AI Matt pane (if tmux session exists)
if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
    # Send notification to AI Matt pane
    tmux send-keys -t "$TMUX_SESSION:$AI_MATT_PANE" "" # Just to wake it up
    tmux send-keys -t "$TMUX_SESSION:$AI_MATT_PANE" "echo '═══ NEW CONSULTATION FROM CLAUDE ═══' && $GC_BIN handoff --check-inbox" Enter
fi

echo "═══ HANDED OFF TO AI MATT ═══"
exit 0
