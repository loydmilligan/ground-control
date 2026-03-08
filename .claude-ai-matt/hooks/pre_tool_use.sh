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
