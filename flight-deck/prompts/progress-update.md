# Progress Update

Update completion percentage on current work item.

## Usage

```
/progress <percentage> [--note "description"]
```

## Input

- **Percentage**: 0-100 completion percentage
- **Note**: Optional description of what was accomplished

## Process

1. **Get current focus** from `.gc/state.json`

2. **Validate** there is active work (current_focus is set)

3. **Find the item**:
   - If `feat_*`: Update in `.gc/roadmap.json`
   - If `issue_*`: Update in `.gc/issues.json`

4. **Update item**:
   ```json
   {
     "completion_pct": {{percentage}},
     "updated_at": "{{now}}"
   }
   ```

5. **Log activity** in `.gc/state.json`:
   ```json
   {
     "recent_activity": [
       {
         "timestamp": "{{now}}",
         "type": "progress",
         "summary": "Updated {{id}} to {{percentage}}%"
       }
     ]
   }
   ```

## Output

```
Progress updated: {{id}} → {{percentage}}%

{{progress_bar}}

{{note if provided}}
```

Progress bar visualization:
- 0-25%: `[####................] 25%`
- 50%: `[##########..........] 50%`
- 75%: `[###############.....] 75%`
- 100%: `[####################] 100% - Ready to complete!`

## Context Files

- **Read**: `.gc/state.json`, `.gc/roadmap.json` or `.gc/issues.json`
- **Write**: `.gc/roadmap.json` or `.gc/issues.json`, `.gc/state.json`

## Example

```
/progress 60 --note "Implemented core grouping logic, working on AI classification"
```

Output:
```
Progress updated: feat_1710000000 → 60%

[############........] 60%

Implemented core grouping logic, working on AI classification
```

## Notes

- If no current focus is set, prompt user to `/start-work <id>` first
- At 100%, suggest using `/complete` to finish the item
- Progress is visible in Flight Deck after `gc sync`
