# Issue Management Prompt

Use this prompt when you need to add, update, or manage issues in a project's `.gc/issues.json`.

## Adding an Issue

Edit `.gc/issues.json` and add to the `issues` array:

```json
{
  "id": "issue_TIMESTAMP",
  "title": "Brief description of the issue",
  "description": "Detailed description if needed",
  "type": "bug",
  "priority": "high",
  "status": "open",
  "labels": ["optional", "labels"],
  "created_at": "2026-03-10T12:00:00Z",
  "updated_at": "2026-03-10T12:00:00Z"
}
```

### Issue Types
- `bug` - Something is broken
- `enhancement` - Improvement to existing feature
- `question` - Need clarification
- `task` - General work item

### Priority Levels
- `high` - Blocking or critical
- `medium` - Important but not urgent
- `low` - Nice to have

### Status Values
- `open` - Not started
- `in_progress` - Being worked on
- `closed` - Resolved

## Generating Issue ID

Use timestamp-based IDs:
```
issue_YYYYMMDDHHMMSS
```

Example: `issue_20260310120000`

## Updating an Issue

Find the issue by ID and update relevant fields. Always update `updated_at`.

## Closing an Issue

Set `status` to `"closed"` and add `closed_at` timestamp.

## Example: Complete issues.json

```json
{
  "issues": [
    {
      "id": "issue_20260310100000",
      "title": "Auth flow broken on mobile",
      "description": "Login fails when using mobile browser",
      "type": "bug",
      "priority": "high",
      "status": "open",
      "labels": ["auth", "mobile"],
      "created_at": "2026-03-10T10:00:00Z",
      "updated_at": "2026-03-10T10:00:00Z"
    },
    {
      "id": "issue_20260310110000",
      "title": "Add dark mode support",
      "type": "enhancement",
      "priority": "medium",
      "status": "in_progress",
      "created_at": "2026-03-10T11:00:00Z",
      "updated_at": "2026-03-10T12:00:00Z"
    }
  ]
}
```

## When to Add Issues

- Bug discovered during development
- Enhancement idea while coding
- Technical debt identified
- Questions that need answers
- Tasks that can't be done immediately
