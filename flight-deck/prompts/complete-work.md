# Complete Work

Mark current work item as complete.

## Usage

```
/complete [--id <id>] [--summary "what was accomplished"] [--commit "commit message"]
```

## Input

- **ID**: Optional - defaults to current focus from state.json
- **Summary**: Brief description of what was accomplished
- **Commit**: If provided, also request a commit via FD

## Process

1. **Get work item**:
   - Use provided ID or get from `.gc/state.json` current_focus
   - Error if no ID provided and no current focus

2. **Find the item**:
   - If `feat_*`: Update in `.gc/roadmap.json`
   - If `issue_*`: Update in `.gc/issues.json`

3. **Update item**:
   ```json
   {
     "status": "completed",
     "completion_pct": 100,
     "completed_at": "{{now}}",
     "updated_at": "{{now}}"
   }
   ```

4. **Clear state focus**:
   ```json
   {
     "session": {
       "current_focus": null,
       "status": "idle"
     }
   }
   ```

5. **Log activity** in `.gc/state.json`

6. **If --commit provided**, append to `.gc/requests.jsonl`:
   ```json
   {"type":"commit","summary":"{{commit message}}","files":[],"at":"{{now}}","status":"pending"}
   ```

## Output

```
Completed: {{id}}
Title: {{title}}

Summary: {{summary}}

{{if commit requested}}
Commit requested: "{{commit message}}"
FD will handle this on next sync.
{{/if}}

What's next?
- /roadmap-item "new feature" - Add new work
- /start-work <id> - Start another item
- /advisor - Get recommendations (in FD)
```

## Verification Checklist

Before completing, verify:
- [ ] Feature/fix works as expected
- [ ] Tests pass (if applicable)
- [ ] No obvious regressions
- [ ] Code is reasonably clean

If verification fails, use `/progress <lower_pct>` instead.

## Context Files

- **Read**: `.gc/state.json`, `.gc/roadmap.json` or `.gc/issues.json`
- **Write**: `.gc/roadmap.json` or `.gc/issues.json`, `.gc/state.json`, `.gc/requests.jsonl` (if commit)

## Example

```
/complete --summary "Added AI topic grouping with 95% accuracy" --commit "Add topic grouping feature"
```

Output:
```
Completed: feat_1710000000
Title: Topic grouping for notifications using AI

Summary: Added AI topic grouping with 95% accuracy

Commit requested: "Add topic grouping feature"
FD will handle this on next sync.

What's next?
- /roadmap-item "new feature" - Add new work
- /start-work <id> - Start another item
```

## Notes

- Sprint progress automatically updates when items complete
- FD sees completion after `gc sync`
- Consider using `/learn idea "..."` to capture insights from this work
