#!/bin/bash
# Start Delegation Session
# Creates dedicated tmux window with Worker Claude and AI Matt

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
STATE_FILE="$PROJECT_DIR/data/delegation/state.json"

WINDOW_NAME="gc-delegation"
INTERACTIONS=${1:-5}
NO_AUTH_FLAG=""

# Check for --no-auth flag
for arg in "$@"; do
    if [ "$arg" = "--no-auth" ]; then
        NO_AUTH_FLAG="--no-auth"
    fi
done

log() {
    echo "[delegation] $1"
}

error() {
    echo "[delegation] ERROR: $1" >&2
    exit 1
}

# Check if delegation window already exists
if tmux list-windows -F '#{window_name}' 2>/dev/null | grep -q "^${WINDOW_NAME}$"; then
    error "Delegation window already exists. Run 'gc delegate --cancel' first."
fi

log "Creating delegation session for $INTERACTIONS interactions..."

# Get current session and window to return to later
ORIGINAL_SESSION=$(tmux display-message -p '#{session_name}')
ORIGINAL_WINDOW=$(tmux display-message -p '#{window_index}')

# Create delegation window (in background, don't switch to it)
log "Creating dedicated window: $WINDOW_NAME"
WINDOW_ID=$(tmux new-window -d -n "$WINDOW_NAME" -P -F '#{window_id}' -c "$PROJECT_DIR")

# Get pane 0 ID (created with window)
WORKER_PANE=$(tmux list-panes -t "$WINDOW_NAME" -F '#{pane_id}' | head -1)

# Split window horizontally for AI Matt
log "Splitting window for AI Matt"
AI_MATT_PANE=$(tmux split-window -h -t "$WINDOW_NAME" -P -F '#{pane_id}' -c "$PROJECT_DIR")

# Ensure equal sizing
tmux select-layout -t "$WINDOW_NAME" even-horizontal

log "Worker pane: $WORKER_PANE"
log "AI Matt pane: $AI_MATT_PANE"

# Update state.json with new pane IDs
log "Updating delegation state..."
mkdir -p "$PROJECT_DIR/data/delegation"

cat > "$STATE_FILE" << EOF
{
  "user": "ai_matt",
  "interaction_mode": "interactions",
  "mode_count": $INTERACTIONS,
  "target_tasks": null,
  "completed_tasks": [],
  "status": "initializing",
  "started_at": "$(date -Iseconds)",
  "last_handoff_at": null,
  "handoff_count": 0,
  "tmux_session": "$ORIGINAL_SESSION",
  "tmux_window": "$WINDOW_NAME",
  "claude_pane": "$WORKER_PANE",
  "ai_matt_pane": "$AI_MATT_PANE",
  "worker_pane": "$WORKER_PANE",
  "watchdog_pid": null
}
EOF

# Clear inbox/outbox
> "$PROJECT_DIR/data/delegation/inbox.md"
> "$PROJECT_DIR/data/delegation/outbox.md"

# Start AI Matt in right pane
log "Starting AI Matt..."
tmux send-keys -t "$AI_MATT_PANE" "cd '$PROJECT_DIR' && '$SCRIPT_DIR/start-ai-matt.sh'" Enter

# Start Worker Claude in left pane
log "Starting Worker Claude..."
tmux send-keys -t "$WORKER_PANE" "cd '$PROJECT_DIR' && claude" Enter

# Wait for both Claude instances to fully start (look for ❯ prompt)
log "Waiting for Claude instances to start..."
wait_for_claude() {
    local pane_id=$1
    local timeout=30
    local waited=0
    while [ $waited -lt $timeout ]; do
        if tmux capture-pane -t "$pane_id" -p | grep -q '❯'; then
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
    done
    return 1
}

if ! wait_for_claude "$WORKER_PANE"; then
    log "WARNING: Worker Claude may not have started properly"
fi

if ! wait_for_claude "$AI_MATT_PANE"; then
    log "WARNING: AI Matt may not have started properly"
fi

sleep 1

# Send initial context to Worker Claude
WORKER_INIT="You are the WORKER agent in a delegation session with AI Matt.

## Your Role
- Execute tasks assigned to you
- Consult AI Matt for decisions using: gc handoff --to-ai-matt -m \"your question\"
- Check for AI Matt's responses using: gc handoff --check-outbox
- You have $INTERACTIONS interactions available

## Important
- AI Matt is in the pane to your right
- Use gc handoff commands for communication, NOT direct tmux messaging
- When AI Matt responds, continue your work based on the guidance
- If you finish early or get stuck, report status

## To Start
Run 'gc handoff --status' to see current delegation state.
Then begin your assigned task."

log "Sending initialization to Worker Claude..."
tmux send-keys -t "$WORKER_PANE" "$WORKER_INIT"
sleep 1
tmux send-keys -t "$WORKER_PANE" Enter

# Verify submission
sleep 2
if tmux capture-pane -t "$WORKER_PANE" -p | grep -q '❯ .\+'; then
    log "Retrying Worker init submission..."
    tmux send-keys -t "$WORKER_PANE" Enter
fi

# Send initial context to AI Matt
AI_MATT_INIT="You are AI Matt in a delegation session. Read @agents/ai-matt.md for your full personality.

## Quick Reminders
- You make decisions as Matt would
- Use gc handoff --check-inbox to see pending consultations
- Write responses to data/delegation/outbox.md
- Run gc handoff --to-claude when your response is ready
- You have LIMITED permissions - suggest changes, don't make them directly

Standing by for consultations."

log "Sending initialization to AI Matt..."
tmux send-keys -t "$AI_MATT_PANE" "$AI_MATT_INIT"
sleep 1
tmux send-keys -t "$AI_MATT_PANE" Enter

# Verify submission
sleep 2
if tmux capture-pane -t "$AI_MATT_PANE" -p | grep -q '❯ .\+'; then
    log "Retrying AI Matt init submission..."
    tmux send-keys -t "$AI_MATT_PANE" Enter
fi

# Start watchdog in background
log "Starting watchdog..."
nohup "$SCRIPT_DIR/delegation-watchdog.sh" > "$PROJECT_DIR/data/delegation/watchdog.log" 2>&1 &
WATCHDOG_PID=$!

# Add monitor pane to user's current window
log "Adding monitor pane to your window..."
MONITOR_PANE=$(tmux split-window -h -t "${ORIGINAL_SESSION}:${ORIGINAL_WINDOW}" -P -F '#{pane_id}' -c "$PROJECT_DIR" "$SCRIPT_DIR/delegation-monitor.sh $NO_AUTH_FLAG")

# Update state with all pane IDs
jq ".watchdog_pid = $WATCHDOG_PID | .status = \"worker_active\" | .monitor_pane = \"$MONITOR_PANE\"" "$STATE_FILE" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "$STATE_FILE"

log "Watchdog PID: $WATCHDOG_PID"
log "Monitor pane: $MONITOR_PANE"

# Switch back to main pane (left side where user was)
tmux select-pane -t "${ORIGINAL_SESSION}:${ORIGINAL_WINDOW}.0"

# Stay on original window (don't switch to delegation)
log "Delegation session ready!"
log ""
log "Monitor pane added to your window (right side)"
log "Delegation running in background window: $WINDOW_NAME"
log ""
log "To view delegation: tmux select-window -t $WINDOW_NAME"
log "To cancel: gc delegate --cancel"

echo ""
echo "Worker pane: $WORKER_PANE"
echo "AI Matt pane: $AI_MATT_PANE"
echo "Monitor pane: $MONITOR_PANE"
echo "Watchdog PID: $WATCHDOG_PID"
