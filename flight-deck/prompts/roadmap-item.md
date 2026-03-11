# Roadmap Item

Add a feature to a project's roadmap.

## Usage

```
/roadmap-item "Feature title" [--project <name>] [--priority high|medium|low] [--start-now]
```

## Input

- **Title**: Brief feature description
- **Project**: Target project (defaults to current working directory)
- **Priority**: high, medium, or low (default: medium)
- **Start Now**: If set, also begin work immediately

## Process

1. **Generate ID**: Create unique ID as `feat_{timestamp}`

2. **Read current roadmap**:
   ```bash
   cat .gc/roadmap.json
   ```

3. **Add feature entry**:
   ```json
   {
     "id": "feat_1710000000",
     "title": "{{title}}",
     "description": "",
     "status": "idea",
     "priority": "{{priority}}",
     "completion_pct": 0,
     "created_at": "{{now}}",
     "updated_at": "{{now}}"
   }
   ```

4. **Write updated roadmap** to `.gc/roadmap.json`

5. **If --start-now**:
   - Change status to `in_progress`
   - Update `.gc/state.json` with `current_focus: feat_xxx`
   - Brainstorm implementation approach
   - Identify key files/components affected
   - Suggest first steps

## Output

Confirm the feature was added:
```
Added to roadmap: feat_1710000000 "{{title}}"
Status: idea | Priority: {{priority}}

To start working: /start-work feat_1710000000
```

If `--start-now` was used, also output:
```
Starting work on: feat_1710000000

Implementation approach:
1. ...
2. ...
3. ...

First step: ...
```

## Context Files

- **Read**: `.gc/roadmap.json`, `.gc/project.json`
- **Write**: `.gc/roadmap.json`, `.gc/state.json` (if --start-now)

## Example

```
/roadmap-item "Topic grouping for notifications using AI" --project notifai --start-now
```

This will:
1. Add the feature to notifai's `.gc/roadmap.json`
2. Set status to `in_progress`
3. Update state with current focus
4. Brainstorm implementation approach
5. Suggest first steps
