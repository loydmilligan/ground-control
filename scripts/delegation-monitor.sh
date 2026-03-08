#!/bin/bash
# Delegation Monitor
# Shows live communication between Worker Claude and AI Matt
# Also handles permission approvals with optional password protection

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
STATE_FILE="$PROJECT_DIR/data/delegation/state.json"
INBOX_FILE="$PROJECT_DIR/data/delegation/inbox.md"
OUTBOX_FILE="$PROJECT_DIR/data/delegation/outbox.md"
HISTORY_FILE="$PROJECT_DIR/data/delegation/history.jsonl"
ENV_FILE="$PROJECT_DIR/.env"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m' # No Color

# Session authentication
SESSION_AUTHENTICATED=false
REQUIRE_AUTH=false
PASSWORD_HASH=""  # Only stores hash, never plaintext

# Check for --no-auth flag
NO_AUTH=false
for arg in "$@"; do
    if [ "$arg" = "--no-auth" ]; then
        NO_AUTH=true
    fi
done

# Hash function - takes plaintext, outputs hash
hash_password() {
    echo -n "$1" | sha256sum | cut -d' ' -f1
}

# Load password HASH from .env if exists
# We only ever read the hash - never the plaintext password
load_auth_config() {
    if [ -f "$ENV_FILE" ]; then
        # Read the HASH, not plaintext - monitor never knows the actual password
        PASSWORD_HASH=$(grep -E "^GC_APPROVAL_PASSWORD_HASH=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- | tr -d '"' | tr -d "'")
        if [ -n "$PASSWORD_HASH" ] && [ "$NO_AUTH" = false ]; then
            REQUIRE_AUTH=true
        fi
    fi
}

# Authenticate session at start
# Uses hash comparison - we hash user input and compare to stored hash
# Monitor never knows the actual password, only verifies hash match
authenticate_session() {
    if [ "$REQUIRE_AUTH" = false ]; then
        SESSION_AUTHENTICATED=true
        return 0
    fi

    echo -e "${YELLOW}═══ AUTHENTICATION REQUIRED ═══${NC}"
    echo -e "${DIM}This monitor session requires password authentication.${NC}"
    echo -e "${DIM}(Using cryptographic hash verification)${NC}"
    echo ""

    local attempts=3
    while [ $attempts -gt 0 ]; do
        echo -n -e "${BOLD}Enter approval password: ${NC}"
        read -s entered_password
        echo ""

        # Hash the entered password and compare to stored hash
        local entered_hash=$(hash_password "$entered_password")
        # Clear plaintext from memory immediately
        entered_password=""

        if [ "$entered_hash" = "$PASSWORD_HASH" ]; then
            SESSION_AUTHENTICATED=true
            # Update state so other sessions can detect auth completed
            if [ -f "$STATE_FILE" ]; then
                jq '.monitor_authenticated = true' "$STATE_FILE" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "$STATE_FILE"
            fi
            echo -e "${GREEN}✓ Authenticated (hash verified)${NC}"
            echo ""
            sleep 1
            return 0
        else
            attempts=$((attempts - 1))
            if [ $attempts -gt 0 ]; then
                echo -e "${RED}✗ Hash mismatch. $attempts attempts remaining.${NC}"
            fi
        fi
    done

    echo -e "${RED}✗ Authentication failed. Monitor exiting.${NC}"
    exit 1
}

# Verify password for approval
# Uses hash comparison - never stores or compares plaintext
verify_approval() {
    if [ "$REQUIRE_AUTH" = false ]; then
        return 0
    fi

    echo -n -e "${BOLD}Enter password to approve: ${NC}"
    read -s entered_password
    echo ""

    # Hash and compare - plaintext never stored
    local entered_hash=$(hash_password "$entered_password")
    entered_password=""  # Clear from memory

    if [ "$entered_hash" = "$PASSWORD_HASH" ]; then
        return 0
    else
        echo -e "${RED}✗ Hash mismatch. Approval denied.${NC}"
        return 1
    fi
}

load_auth_config

clear
echo -e "${BOLD}═══════════════════════════════════════${NC}"
echo -e "${BOLD}       DELEGATION MONITOR${NC}"
echo -e "${BOLD}═══════════════════════════════════════${NC}"
echo ""

# Authenticate before starting
authenticate_session

if [ "$REQUIRE_AUTH" = true ]; then
    echo -e "${GREEN}🔒 Secure mode active${NC}"
else
    echo -e "${DIM}🔓 Open mode (no password required)${NC}"
fi
echo ""
echo -e "${DIM}Watching for activity...${NC}"
echo -e "${DIM}Press Ctrl+C to exit${NC}"
echo ""

get_state_field() {
    jq -r ".$1 // empty" "$STATE_FILE" 2>/dev/null
}

# Track what we've shown
last_inbox_hash=""
last_outbox_hash=""
last_handoff_count=0
prompted_for_more=false

# Check for pending approval in any delegation pane
# Returns 0 if approval needed, sets APPROVAL_PANE to the pane ID
APPROVAL_PANE=""
check_approval_needed() {
    local worker_pane=$(get_state_field "worker_pane")
    local ai_matt_pane=$(get_state_field "ai_matt_pane")

    # Check both panes for approval prompts
    for pane_id in "$worker_pane" "$ai_matt_pane"; do
        if [ -z "$pane_id" ]; then
            continue
        fi

        local content=$(tmux capture-pane -t "$pane_id" -p 2>/dev/null | tail -15)

        # Look for Claude Code permission prompt patterns
        if echo "$content" | grep -qE "Do you want to|overwrite|make this edit|\[y/n\]"; then
            if echo "$content" | grep -qE "❯ 1\.|Yes.*No|1\. Yes"; then
                APPROVAL_PANE="$pane_id"
                return 0
            fi
        fi
    done
    return 1
}

# Forward approval to the pane that needs it
send_approval() {
    local choice=$1

    if [ -n "$APPROVAL_PANE" ]; then
        tmux send-keys -t "$APPROVAL_PANE" "$choice"
        sleep 0.3
        tmux send-keys -t "$APPROVAL_PANE" Enter
        echo -e "${GREEN}✓ Sent approval to pane $APPROVAL_PANE${NC}"
    else
        echo -e "${RED}✗ No approval pane set${NC}"
    fi
}

# Show inbox content (Worker → AI Matt)
show_inbox() {
    if [ -f "$INBOX_FILE" ] && [ -s "$INBOX_FILE" ]; then
        local hash=$(md5sum "$INBOX_FILE" 2>/dev/null | cut -d' ' -f1)
        if [ "$hash" != "$last_inbox_hash" ]; then
            last_inbox_hash="$hash"
            echo ""
            echo -e "${CYAN}━━━ $(date '+%H:%M:%S') Worker → AI Matt ━━━${NC}"
            # Show first 10 lines of content
            head -15 "$INBOX_FILE" | while read -r line; do
                echo -e "  ${DIM}$line${NC}"
            done
            echo ""
        fi
    fi
}

# Show outbox content (AI Matt → Worker)
show_outbox() {
    if [ -f "$OUTBOX_FILE" ] && [ -s "$OUTBOX_FILE" ]; then
        local hash=$(md5sum "$OUTBOX_FILE" 2>/dev/null | cut -d' ' -f1)
        if [ "$hash" != "$last_outbox_hash" ]; then
            last_outbox_hash="$hash"
            echo ""
            echo -e "${GREEN}━━━ $(date '+%H:%M:%S') AI Matt → Worker ━━━${NC}"
            head -15 "$OUTBOX_FILE" | while read -r line; do
                echo -e "  ${DIM}$line${NC}"
            done
            echo ""
        fi
    fi
}

# Show status
show_status() {
    local status=$(get_state_field "status")
    local handoffs=$(get_state_field "handoff_count")
    local remaining=$(get_state_field "mode_count")

    if [ "$handoffs" != "$last_handoff_count" ]; then
        last_handoff_count="$handoffs"
        echo -e "${BLUE}📊 Status: $status | Handoffs: $handoffs | Remaining: $remaining${NC}"
    fi
}

# Check if interactions exhausted and prompt to add more
check_add_interactions() {
    local remaining=$(get_state_field "mode_count")
    local status=$(get_state_field "status")

    # Only prompt once per exhaustion, and only when AI Matt is waiting
    if [ "$remaining" = "0" ] && [ "$status" = "waiting_for_ai_matt" ] && [ "$prompted_for_more" = false ]; then
        prompted_for_more=true
        echo ""
        echo -e "${YELLOW}═══ INTERACTIONS EXHAUSTED ═══${NC}"
        echo -e "${YELLOW}AI Matt has another turn but no interactions remain.${NC}"
        echo -n -e "${BOLD}Add more interactions? [number/n]: ${NC}"

        if read -t 60 response; then
            echo ""
            if [[ "$response" =~ ^[0-9]+$ ]] && [ "$response" -gt 0 ]; then
                if verify_approval; then
                    # Update state with new interactions
                    jq ".mode_count = $response" "$STATE_FILE" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "$STATE_FILE"
                    echo -e "${GREEN}✓ Added $response interactions${NC}"
                    prompted_for_more=false  # Reset so we can prompt again if needed
                fi
            else
                echo -e "${DIM}No interactions added. Delegation will end.${NC}"
            fi
        fi
    elif [ "$remaining" != "0" ]; then
        # Reset flag when we have interactions again
        prompted_for_more=false
    fi
}

# Main loop
while true; do
    # Check if delegation is still active
    status=$(get_state_field "status")
    if [ "$status" = "cancelled" ] || [ "$status" = "completed" ]; then
        echo ""
        echo -e "${YELLOW}═══ Delegation $status ═══${NC}"
        echo -e "${DIM}Monitor closing in 5 seconds...${NC}"
        sleep 5
        exit 0
    fi

    # Show any new communications
    show_inbox
    show_outbox
    show_status

    # Check if interactions exhausted
    check_add_interactions

    # Check for approval needed
    if check_approval_needed; then
        echo ""
        echo -e "${YELLOW}⚠️  ═══ APPROVAL NEEDED ═══${NC}"
        echo -e "${YELLOW}AI Matt wants to perform an action${NC}"
        echo -n -e "${BOLD}Approve? [y/n/v(iew)]: ${NC}"

        # Read with timeout so we can keep checking
        if read -t 30 -n 1 response; then
            echo ""
            case $response in
                y|Y)
                    if verify_approval; then
                        send_approval "1"
                    fi
                    ;;
                n|N)
                    send_approval "3"
                    echo -e "${RED}✗ Denied${NC}"
                    ;;
                v|V)
                    echo -e "${DIM}Showing pane content...${NC}"
                    tmux capture-pane -t "$APPROVAL_PANE" -p | tail -20
                    echo ""
                    echo -n -e "${BOLD}Now approve? [y/n]: ${NC}"
                    read -n 1 response2
                    echo ""
                    if [ "$response2" = "y" ] || [ "$response2" = "Y" ]; then
                        if verify_approval; then
                            send_approval "1"
                        fi
                    else
                        send_approval "3"
                    fi
                    ;;
            esac
        fi
    fi

    sleep 2
done
