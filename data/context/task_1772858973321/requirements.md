# Requirements for: Add Sprint entity to Ground Control

## Description

Add Sprint entity to Ground Control. Sprint is a lightweight grouping of related tasks toward a goal or work session. Schema: id, name, description, goal, tasks[] (task IDs), status (active/completed/paused), created_at, completed_at. Add to data/sprints.json. Commands: gc sprint create, gc sprint list, gc sprint add-task <sprint> <task>, gc sprint status. Keep it simple - not full project management, just work session grouping. Should be able to run gc orc on a sprint to execute all its tasks.

## Background

Created from brain dump: dump_1772858967710

## Acceptance Criteria

