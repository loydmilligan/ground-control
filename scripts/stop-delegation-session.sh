#!/bin/bash
# Stop Delegation Session
# Cleanly shuts down delegation window and watchdog

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
STATE_FILE="$PROJECT_DIR/data/delegation/state.json"

WINDOW_NAME="gc-delegation"
KEEP_WINDOW=${KEEP_WINDOW:-false}

log() {
    echo "[delegation] $1"
}

# Stop watchdog if running
if [ -f "$STATE_FILE" ]; then
    WATCHDOG_PID=$(jq -r '.watchdog_pid // empty' "$STATE_FILE" 2>/dev/null)
    if [ -n "$WATCHDOG_PID" ] && kill -0 "$WATCHDOG_PID" 2>/dev/null; then
        log "Stopping watchdog (PID: $WATCHDOG_PID)..."
        kill "$WATCHDOG_PID" 2>/dev/null || true
        sleep 1
    fi

    # Close monitor pane if it exists
    MONITOR_PANE=$(jq -r '.monitor_pane // empty' "$STATE_FILE" 2>/dev/null)
    if [ -n "$MONITOR_PANE" ]; then
        log "Closing monitor pane ($MONITOR_PANE)..."
        tmux kill-pane -t "$MONITOR_PANE" 2>/dev/null || true
    fi
fi

# Update state to cancelled
if [ -f "$STATE_FILE" ]; then
    log "Updating state to cancelled..."
    jq '.status = "cancelled" | .user = "human_matt" | .watchdog_pid = null' "$STATE_FILE" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "$STATE_FILE"
fi

# Kill delegation window unless KEEP_WINDOW is set
if [ "$KEEP_WINDOW" != "true" ]; then
    if tmux list-windows -F '#{window_name}' 2>/dev/null | grep -q "^${WINDOW_NAME}$"; then
        log "Killing delegation window..."
        tmux kill-window -t "$WINDOW_NAME" 2>/dev/null || true
    else
        log "Delegation window not found (already closed?)"
    fi
else
    log "Keeping delegation window open for review"
fi

log "Delegation session stopped"
log ""
log "Review history: gc handoff --history"
log "View last state: cat $STATE_FILE"
