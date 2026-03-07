#!/bin/bash
# AI Matt Delegation System Setup
#
# This script sets up the tmux environment for AI Matt delegation.
# It creates two panes: one for Claude (worker) and one for AI Matt (decision maker).
#
# Usage:
#   ./scripts/ai-matt-setup.sh         # Full setup
#   ./scripts/ai-matt-setup.sh attach  # Attach to existing session
#   ./scripts/ai-matt-setup.sh reset   # Kill and recreate session
#   ./scripts/ai-matt-setup.sh status  # Check session status
#
# After setup:
#   1. In left pane (Claude): Start working on tasks
#   2. Run: gc delegate --interactions 5
#   3. Claude will hand off to AI Matt automatically
#   4. AI Matt (right pane) responds with decisions

set -e

SESSION_NAME="gc"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_dependencies() {
    if ! command -v tmux &> /dev/null; then
        log_error "tmux is not installed. Please install it first."
        exit 1
    fi

    if ! command -v claude &> /dev/null; then
        log_error "claude CLI is not installed. Please install Claude Code first."
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq is not installed. Please install it first."
        exit 1
    fi

    log_info "All dependencies found"
}

session_exists() {
    tmux has-session -t "$SESSION_NAME" 2>/dev/null
}

show_status() {
    echo "═══ AI Matt Session Status ═══"
    echo ""

    if session_exists; then
        log_info "Session '$SESSION_NAME' is running"
        echo ""
        echo "Panes:"
        tmux list-panes -t "$SESSION_NAME" -F "  Pane #{pane_index}: #{pane_current_command} (#{pane_width}x#{pane_height})"
        echo ""
        echo "To attach: tmux attach -t $SESSION_NAME"
    else
        log_warn "Session '$SESSION_NAME' is not running"
        echo ""
        echo "To start: ./scripts/ai-matt-setup.sh"
    fi

    echo ""
    echo "Delegation state:"
    if [[ -f "$PROJECT_DIR/data/delegation/state.json" ]]; then
        cat "$PROJECT_DIR/data/delegation/state.json" | jq -r '"  User: \(.user)\n  Mode: \(.interaction_mode)\n  Remaining: \(.mode_count)\n  Status: \(.status)"'
    else
        echo "  No delegation state file"
    fi
}

kill_session() {
    if session_exists; then
        log_info "Killing existing session '$SESSION_NAME'..."
        tmux kill-session -t "$SESSION_NAME"
    fi
}

create_session() {
    log_info "Creating tmux session '$SESSION_NAME'..."

    # Create session with first window
    tmux new-session -d -s "$SESSION_NAME" -n main -c "$PROJECT_DIR"

    # Split horizontally (left: Claude, right: AI Matt)
    tmux split-window -h -t "$SESSION_NAME:main" -c "$PROJECT_DIR"

    # Set up left pane (Claude - pane 0)
    tmux select-pane -t "$SESSION_NAME:main.0"
    tmux send-keys -t "$SESSION_NAME:main.0" "# Claude (Worker Agent) - Pane 0" Enter
    tmux send-keys -t "$SESSION_NAME:main.0" "# Start with: claude" Enter
    tmux send-keys -t "$SESSION_NAME:main.0" "# Then: gc delegate --interactions N" Enter
    tmux send-keys -t "$SESSION_NAME:main.0" "cd '$PROJECT_DIR'" Enter
    tmux send-keys -t "$SESSION_NAME:main.0" "clear" Enter

    # Set up right pane (AI Matt - pane 1)
    tmux select-pane -t "$SESSION_NAME:main.1"
    tmux send-keys -t "$SESSION_NAME:main.1" "# AI Matt (Decision Agent) - Pane 1" Enter
    tmux send-keys -t "$SESSION_NAME:main.1" "# Wait for consultations from Claude" Enter
    tmux send-keys -t "$SESSION_NAME:main.1" "# Check inbox: gc handoff --check-inbox" Enter
    tmux send-keys -t "$SESSION_NAME:main.1" "# Respond: gc handoff --to-claude -m 'your response'" Enter
    tmux send-keys -t "$SESSION_NAME:main.1" "cd '$PROJECT_DIR'" Enter
    tmux send-keys -t "$SESSION_NAME:main.1" "clear" Enter

    # Select Claude pane as active
    tmux select-pane -t "$SESSION_NAME:main.0"

    log_info "Session created with 2 panes"
}

init_state() {
    log_info "Initializing delegation state..."
    mkdir -p "$PROJECT_DIR/data/delegation"

    if [[ ! -f "$PROJECT_DIR/data/delegation/state.json" ]]; then
        cat > "$PROJECT_DIR/data/delegation/state.json" << 'EOF'
{
  "user": "human_matt",
  "interaction_mode": "interactions",
  "mode_count": 0,
  "target_tasks": [],
  "completed_tasks": [],
  "status": "idle",
  "started_at": null,
  "last_handoff_at": null,
  "handoff_count": 0,
  "error": null,
  "tmux_session": "gc",
  "claude_pane": "0.0",
  "ai_matt_pane": "0.1"
}
EOF
        log_info "Created initial state file"
    fi
}

attach_session() {
    if ! session_exists; then
        log_error "Session '$SESSION_NAME' does not exist. Run setup first."
        exit 1
    fi

    log_info "Attaching to session '$SESSION_NAME'..."
    log_info "Use Ctrl-b + arrow keys to switch panes"
    log_info "Use Ctrl-b + d to detach"
    echo ""

    tmux attach -t "$SESSION_NAME"
}

print_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  (none)   Full setup - create session and initialize"
    echo "  attach   Attach to existing session"
    echo "  reset    Kill existing session and recreate"
    echo "  status   Show session and delegation status"
    echo "  help     Show this help"
    echo ""
    echo "After setup:"
    echo "  1. Attach to session: tmux attach -t gc"
    echo "  2. In left pane: Start claude"
    echo "  3. Run: gc delegate --interactions 5"
    echo "  4. Work normally - handoffs happen automatically"
    echo "  5. AI Matt (right pane) responds to consultations"
}

# Main
case "${1:-}" in
    attach)
        attach_session
        ;;
    reset)
        kill_session
        check_dependencies
        create_session
        init_state
        echo ""
        log_info "Session reset complete"
        echo ""
        echo "To attach: tmux attach -t $SESSION_NAME"
        ;;
    status)
        show_status
        ;;
    help|--help|-h)
        print_usage
        ;;
    "")
        check_dependencies

        if session_exists; then
            log_warn "Session '$SESSION_NAME' already exists"
            echo ""
            echo "Options:"
            echo "  Attach to it:  $0 attach"
            echo "  Reset it:      $0 reset"
            echo "  Check status:  $0 status"
            exit 0
        fi

        create_session
        init_state

        echo ""
        log_info "Setup complete!"
        echo ""
        echo "═══ Quick Start ═══"
        echo ""
        echo "1. Attach to session:"
        echo "   tmux attach -t $SESSION_NAME"
        echo ""
        echo "2. In LEFT pane (Claude), start working:"
        echo "   claude"
        echo ""
        echo "3. Start delegation:"
        echo "   gc delegate --interactions 5"
        echo ""
        echo "4. Work normally. At end of each action, Claude will"
        echo "   hand off to AI Matt (right pane) automatically."
        echo ""
        echo "5. In RIGHT pane (AI Matt), respond to consultations:"
        echo "   gc handoff --check-inbox"
        echo "   # Write response to data/delegation/outbox.md"
        echo "   gc handoff --to-claude"
        echo ""
        echo "═══════════════════"
        ;;
    *)
        log_error "Unknown command: $1"
        print_usage
        exit 1
        ;;
esac
