#!/bin/bash
# Delegation Watchdog
# Monitors delegation panes for stuck states and auto-recovers

set -e

STATE_FILE="data/delegation/state.json"
CHECK_INTERVAL=${CHECK_INTERVAL:-10}  # seconds
IDLE_THRESHOLD=${IDLE_THRESHOLD:-30}  # seconds before submitting pending text
STUCK_THRESHOLD=${STUCK_THRESHOLD:-120}  # seconds before alerting

log() {
    echo "[$(date '+%H:%M:%S')] $1"
}

get_state_field() {
    jq -r ".$1 // empty" "$STATE_FILE" 2>/dev/null
}

# Detect unsubmitted text in a pane
# Returns 0 if unsubmitted text found, 1 if clean
detect_unsubmitted() {
    local pane_id=$1
    local content=$(tmux capture-pane -t "$pane_id" -p -S -10 2>/dev/null)

    if [ -z "$content" ]; then
        return 1
    fi

    # Get last non-empty line
    local last_line=$(echo "$content" | grep -v '^$' | tail -1)

    # Check for text on same line as Claude Code prompt
    if echo "$last_line" | grep -qE '❯ .+'; then
        return 0
    fi

    # Check if last line is text (not a prompt) and previous line has prompt
    if ! echo "$last_line" | grep -qE '^(❯|➜|\$|>)'; then
        local prev_line=$(echo "$content" | grep -v '^$' | tail -2 | head -1)
        if echo "$prev_line" | grep -qE '(❯|➜)'; then
            return 0
        fi
    fi

    return 1
}

# Check if pane is showing "thinking" indicator (Claude is working)
is_thinking() {
    local pane_id=$1
    local content=$(tmux capture-pane -t "$pane_id" -p -S -5 2>/dev/null)

    # Look for Claude's thinking indicators
    if echo "$content" | grep -qiE '(thinking|Moonwalking|processing)'; then
        return 0
    fi
    return 1
}

# Submit pending text in a pane
submit_pending() {
    local pane_id=$1
    log "Submitting pending text in pane $pane_id"

    # Try Enter first
    tmux send-keys -t "$pane_id" Enter
    sleep 2

    # Check if still pending - try Enter again (NOT Escape - that interrupts Claude!)
    if detect_unsubmitted "$pane_id"; then
        log "Retry: Enter again"
        tmux send-keys -t "$pane_id" Enter
        sleep 2
    fi

    # Final check
    if detect_unsubmitted "$pane_id"; then
        log "WARNING: Could not submit text in pane $pane_id"
        return 1
    fi

    log "Successfully submitted text in pane $pane_id"
    return 0
}

# Main watchdog loop
main() {
    log "Delegation watchdog started"
    log "Check interval: ${CHECK_INTERVAL}s, Idle threshold: ${IDLE_THRESHOLD}s"

    local last_activity=$(date +%s)
    local worker_idle_since=0
    local ai_matt_idle_since=0

    while true; do
        sleep "$CHECK_INTERVAL"

        # Check if delegation is active
        local status=$(get_state_field "status")
        if [ "$status" != "worker_active" ] && [ "$status" != "waiting_for_ai_matt" ]; then
            log "Delegation not active (status: $status), watchdog exiting"
            break
        fi

        # Support both old (claude_pane) and new (worker_pane) field names
        local worker_pane=$(get_state_field "worker_pane")
        if [ -z "$worker_pane" ]; then
            worker_pane=$(get_state_field "claude_pane")
        fi
        local ai_matt_pane=$(get_state_field "ai_matt_pane")

        if [ -z "$worker_pane" ] || [ -z "$ai_matt_pane" ]; then
            log "ERROR: Pane IDs not set in state"
            continue
        fi

        local now=$(date +%s)

        # Check worker pane
        if is_thinking "$worker_pane"; then
            worker_idle_since=0
            log "Worker is thinking..."
        elif detect_unsubmitted "$worker_pane"; then
            if [ $worker_idle_since -eq 0 ]; then
                worker_idle_since=$now
                log "Worker has pending text"
            elif [ $((now - worker_idle_since)) -ge $IDLE_THRESHOLD ]; then
                log "Worker idle with pending text for ${IDLE_THRESHOLD}s, submitting..."
                submit_pending "$worker_pane"
                worker_idle_since=0
            fi
        else
            worker_idle_since=0
        fi

        # Check AI Matt pane
        if is_thinking "$ai_matt_pane"; then
            ai_matt_idle_since=0
            log "AI Matt is thinking..."
        elif detect_unsubmitted "$ai_matt_pane"; then
            if [ $ai_matt_idle_since -eq 0 ]; then
                ai_matt_idle_since=$now
                log "AI Matt has pending text"
            elif [ $((now - ai_matt_idle_since)) -ge $IDLE_THRESHOLD ]; then
                log "AI Matt idle with pending text for ${IDLE_THRESHOLD}s, submitting..."
                submit_pending "$ai_matt_pane"
                ai_matt_idle_since=0
            fi
        else
            ai_matt_idle_since=0
        fi

        # Update last activity timestamp if something is happening
        if is_thinking "$worker_pane" || is_thinking "$ai_matt_pane"; then
            last_activity=$now
        fi

        # Check for overall stuck state
        if [ $((now - last_activity)) -ge $STUCK_THRESHOLD ]; then
            log "WARNING: System appears stuck for ${STUCK_THRESHOLD}s"
            log "Consider running: gc delegate --diagnose"
            # Don't exit, keep monitoring
            last_activity=$now  # Reset to avoid spam
        fi
    done

    log "Watchdog exiting"
}

# Handle signals
trap 'log "Watchdog received signal, exiting"; exit 0' SIGTERM SIGINT

main "$@"
