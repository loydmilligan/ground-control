#!/bin/bash
# Test artifact generation with input piped in

echo "test-project
Build a CLI tool for artifact generation
Go, Cobra, Bubble Tea
Must support templates, Must generate docs" | go run ./cmd/gc artifact generate project_plan --output /tmp/test_plan.md
