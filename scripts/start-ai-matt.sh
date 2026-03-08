#!/bin/bash
# Start AI Matt with restricted permissions
# This script launches Claude with the AI Matt personality and limited tool access

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
AI_MATT_CLAUDE_DIR="$PROJECT_DIR/.claude-ai-matt"

cd "$PROJECT_DIR"

echo "Starting AI Matt..."

# Check if restricted environment exists
if [ -d "$AI_MATT_CLAUDE_DIR" ]; then
    echo "Using restricted Claude environment: $AI_MATT_CLAUDE_DIR"
    echo ""

    # Export the custom config directory
    export CLAUDE_CONFIG_DIR="$AI_MATT_CLAUDE_DIR"

    # Launch Claude with AI Matt personality in restricted mode
    exec claude --system-prompt "$PROJECT_DIR/agents/ai-matt.md"
else
    echo "WARNING: Restricted environment not found at $AI_MATT_CLAUDE_DIR"
    echo "Run 'scripts/setup-ai-matt-env.sh' to create it"
    echo ""
    echo "Starting with default permissions (NOT RECOMMENDED for production)"

    # Launch Claude with AI Matt personality (unrestricted mode)
    exec claude --system-prompt "$PROJECT_DIR/agents/ai-matt.md"
fi
