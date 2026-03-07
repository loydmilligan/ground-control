# Requirements for: Add session management commands - gc sessions --list, --c...

## Description

Add session management commands - gc sessions --list, --cleanup, --cancel. Currently 7 sessions stuck in 'running' state with no way to clean them up from CLI. Also add stale session detection (if >1 hour old with no updates, mark as stale). Moderate priority.

## Background

Created from brain dump: dump_1772856066855

## Acceptance Criteria

