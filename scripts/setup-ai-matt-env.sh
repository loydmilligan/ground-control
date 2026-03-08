#!/bin/bash
# Setup AI Matt Claude Environment
# Creates a controlled Claude configuration for AI Matt

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
AI_MATT_CLAUDE_DIR="$PROJECT_DIR/.claude-ai-matt"
SETTINGS_FILE="$PROJECT_DIR/data/ai-matt-config/settings.json"

log() {
    echo "[ai-matt-setup] $1"
}

error() {
    echo "[ai-matt-setup] ERROR: $1" >&2
    exit 1
}

# Create AI Matt's Claude directory
log "Creating AI Matt Claude environment..."
mkdir -p "$AI_MATT_CLAUDE_DIR"

# Create settings.json with restricted permissions
log "Creating restricted settings..."
cat > "$AI_MATT_CLAUDE_DIR/settings.json" << 'EOF'
{
  "permissions": {
    "allow": {
      "Read": {},
      "Glob": {},
      "Grep": {},
      "Bash": {
        "commands": ["gc", "git status", "git diff", "git log", "ls", "cat", "head", "tail"]
      }
    },
    "deny": {
      "Edit": { "reason": "AI Matt cannot edit files directly" },
      "Write": { "reason": "AI Matt cannot write files directly" },
      "NotebookEdit": { "reason": "AI Matt cannot edit notebooks" }
    }
  },
  "hasTrustDialogAccepted": true
}
EOF

# Create hooks directory
log "Setting up hooks..."
mkdir -p "$AI_MATT_CLAUDE_DIR/hooks"

# Pre-action hook - prevents certain operations
cat > "$AI_MATT_CLAUDE_DIR/hooks/pre_tool_use.sh" << 'EOF'
#!/bin/bash
# Pre-tool-use hook for AI Matt
# Blocks dangerous operations

TOOL_NAME="$1"
shift

case "$TOOL_NAME" in
    Edit|Write|NotebookEdit)
        echo "BLOCKED: AI Matt cannot modify files directly. Use handoff to request changes."
        exit 1
        ;;
    Bash)
        # Check for dangerous commands
        CMD="$1"
        if echo "$CMD" | grep -qE "(rm -rf|rm -r|git push|git reset --hard|chmod|chown)"; then
            echo "BLOCKED: Dangerous command not allowed for AI Matt"
            exit 1
        fi
        ;;
esac

exit 0
EOF
chmod +x "$AI_MATT_CLAUDE_DIR/hooks/pre_tool_use.sh"

# Create CLAUDE.md for AI Matt
log "Creating AI Matt CLAUDE.md..."
cat > "$AI_MATT_CLAUDE_DIR/CLAUDE.md" << 'EOF'
# AI Matt - Restricted Environment

You are AI Matt, operating in a **restricted environment**.

## Restrictions

**You CANNOT:**
- Edit or write files directly
- Run destructive commands (rm -rf, git push, etc.)
- Make changes without human approval

**You CAN:**
- Read files to understand context
- Search code with Glob/Grep
- Run safe read-only commands
- Provide guidance and decisions via handoff

## Your Role

When consulted by Worker Claude:
1. Read the inbox to understand the question
2. Analyze using available read-only tools
3. Write your response to the outbox
4. Signal completion with gc handoff --to-claude

## Infinite Loop Prevention

- Do not ask the same question repeatedly
- If stuck, escalate to human Matt
- Maximum 5 consecutive handoffs before pause

## Remember

You are Matt's proxy for decisions, not for implementation.
Think like Matt, decide like Matt, but let others do the work.
EOF

log "AI Matt environment created at: $AI_MATT_CLAUDE_DIR"
log ""
log "To use:"
log "  CLAUDE_CONFIG_DIR=$AI_MATT_CLAUDE_DIR claude"
log ""
log "Or update scripts/start-ai-matt.sh to use this directory"
