# Roadmap Management Prompt

Use this prompt when you need to add, update, or manage features in a project's `.gc/roadmap.json`.

## Adding a Feature

Edit `.gc/roadmap.json` and add to the `features` array:

```json
{
  "id": "feat_TIMESTAMP",
  "title": "Feature name",
  "description": "What this feature does",
  "status": "planned",
  "priority": "medium",
  "completion_pct": 0,
  "milestone_id": "v1.0",
  "created_at": "2026-03-10T12:00:00Z",
  "updated_at": "2026-03-10T12:00:00Z"
}
```

### Feature Status Values
- `planned` - Not started
- `in_progress` - Being worked on
- `completed` - Done and shipped
- `cancelled` - Won't do

### Priority Levels
- `high` - Must have for milestone
- `medium` - Should have
- `low` - Nice to have

## Adding a Milestone

Add to the `milestones` array:

```json
{
  "id": "v1.0",
  "name": "Version 1.0",
  "target_date": "2026-04-01",
  "feature_ids": ["feat_001", "feat_002"],
  "status": "planned"
}
```

## Updating Progress

Update `completion_pct` (0-100) and `updated_at` as work progresses.

## Example: Complete roadmap.json

```json
{
  "features": [
    {
      "id": "feat_20260310100000",
      "title": "User authentication",
      "description": "OAuth and email/password login",
      "status": "completed",
      "priority": "high",
      "completion_pct": 100,
      "milestone_id": "v1.0",
      "created_at": "2026-03-01T10:00:00Z",
      "updated_at": "2026-03-10T10:00:00Z"
    },
    {
      "id": "feat_20260310110000",
      "title": "Dark mode",
      "description": "System and manual dark mode toggle",
      "status": "in_progress",
      "priority": "medium",
      "completion_pct": 60,
      "milestone_id": "v1.1",
      "created_at": "2026-03-05T11:00:00Z",
      "updated_at": "2026-03-10T12:00:00Z"
    }
  ],
  "milestones": [
    {
      "id": "v1.0",
      "name": "MVP Release",
      "target_date": "2026-03-15",
      "feature_ids": ["feat_20260310100000"],
      "status": "completed"
    },
    {
      "id": "v1.1",
      "name": "Polish Release",
      "target_date": "2026-04-01",
      "feature_ids": ["feat_20260310110000"],
      "status": "in_progress"
    }
  ]
}
```

## When to Update Roadmap

- New feature identified
- Feature work starts (status → in_progress)
- Progress made (update completion_pct)
- Feature completed (status → completed, completion_pct → 100)
- Plans change (update status, priority, or cancel)
