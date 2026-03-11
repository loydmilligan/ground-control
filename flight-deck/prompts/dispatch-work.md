# Dispatch Work

Send work from Flight Deck to a project's inbox.

## Usage (FD Claude only)

```
/dispatch --project <name> --type <work_type> "description"
```

## Input

- **Project**: Target project name
- **Type**: coding, testing, fix, review, docs
- **Description**: What needs to be done

## Process

1. **Locate project** from registry (`~/.gc/global.json`)

2. **Create work item** with unique ID:
   ```json
   {
     "id": "work_{{timestamp}}",
     "type": "{{type}}",
     "description": "{{description}}",
     "dispatched_at": "{{now}}",
     "dispatched_by": "fd",
     "status": "pending",
     "related_items": []
   }
   ```

3. **Write to project inbox**:
   - Path: `{{project_path}}/.gc/inbox/work_{{timestamp}}.json`

4. **Log dispatch** in FD activity

## Output

```
Dispatched to {{project}}:
  ID: work_{{timestamp}}
  Type: {{type}}
  Description: {{description}}

Project Claude will see this in .gc/inbox/ on next session.
```

## Work Types

| Type | Description | Example |
|------|-------------|---------|
| `coding` | Implement feature or fix | "Implement user auth" |
| `testing` | Write or run tests | "Add unit tests for parser" |
| `fix` | Bug fix | "Fix null pointer in handler" |
| `review` | Code review needed | "Review PR #42" |
| `docs` | Documentation | "Update README with new API" |

## Context Files

- **Read**: `~/.gc/global.json` (registry)
- **Write**: `{{project}}/.gc/inbox/work_*.json`

## Example

```
/dispatch --project notifai --type coding "Implement topic grouping AI classifier"
```

Creates `/home/user/Projects/notifai/.gc/inbox/work_1710000000.json`:
```json
{
  "id": "work_1710000000",
  "type": "coding",
  "description": "Implement topic grouping AI classifier",
  "dispatched_at": "2026-03-11T10:00:00Z",
  "dispatched_by": "fd",
  "status": "pending"
}
```

## Project Claude Pickup

When project Claude starts a session, it should:
1. Check `.gc/inbox/` for pending work items
2. Present them to the user
3. Use `/start-work` to begin (creates roadmap/issue entry)
4. Mark inbox item as `started`

## Notes

- This is for FD → Project communication
- Project → FD uses `.gc/requests.jsonl` instead
- Inbox items are separate from roadmap/issues (they get converted on pickup)
