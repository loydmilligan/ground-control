# Start Work

Begin working on a roadmap feature or issue.

## Usage

```
/start-work <id>
```

## Input

- **ID**: Feature ID (`feat_xxx`) or Issue ID (`issue_xxx`)

## Process

1. **Find the item**:
   - If `feat_*`: Read from `.gc/roadmap.json`
   - If `issue_*`: Read from `.gc/issues.json`

2. **Validate item exists** and is not already completed

3. **Update item status** to `in_progress`

4. **Update state.json**:
   ```json
   {
     "session": {
       "current_focus": "{{id}}",
       "status": "active"
     }
   }
   ```

5. **Load context**:
   - Read `.gc/fd-onboarding.md` for FD integration rules
   - Read `.gc/project.json` for tech stack
   - Check related items in roadmap/issues

6. **Present work context**:
   - Item title and description
   - Related items (if any)
   - Tech stack constraints
   - FD integration reminders

## Output

```
Starting work on: {{id}}
Title: {{title}}

Context:
- Tech stack: {{languages}}, {{frameworks}}
- Related: {{related_items}}

FD Integration Reminders:
- Request commits via /commit "message"
- Log issues via /issue "description"
- Update progress via /progress <pct>
- Complete via /complete

Ready to begin. What's the first step?
```

## FD Integration Reminders

When working on this item, remember:

1. **Don't commit directly** - Use `/commit "message"` to request FD handle it
2. **Track blockers** - Use `/issue "blocker description" --blocks {{id}}`
3. **Update progress** - Use `/progress 30` periodically
4. **Log learnings** - Use `/learn friction "description"` for process issues
5. **Complete properly** - Use `/complete` when done

## Context Files

- **Read**: `.gc/roadmap.json`, `.gc/issues.json`, `.gc/state.json`, `.gc/fd-onboarding.md`, `.gc/project.json`
- **Write**: `.gc/roadmap.json` or `.gc/issues.json` (status), `.gc/state.json` (current_focus)

## Example

```
/start-work feat_1710000000
```

Output:
```
Starting work on: feat_1710000000
Title: Topic grouping for notifications using AI

Context:
- Tech stack: Go, SQLite
- Related: None

FD Integration Reminders:
- Request commits via /commit "message"
- Update progress via /progress <pct>
- Complete via /complete

Ready to begin. What's the first step?
```
