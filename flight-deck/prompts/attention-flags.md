# Attention Flags Prompt

Use this prompt to set and manage attention flags in project state.

## What are Attention Flags?

Attention flags indicate something needs user attention. They appear in the Flight Deck Hangar view to highlight projects that need action.

## Setting Attention Flags

Edit `.gc/state.json` and add to the `attention` array:

```json
{
  "attention": [
    {
      "type": "blocked",
      "priority": "high",
      "message": "Waiting on API design decision",
      "since": "2026-03-10T12:00:00Z"
    }
  ]
}
```

## Flag Types

| Type | When to Use | Icon |
|------|-------------|------|
| `blocked` | Work cannot continue without resolution | 🚫 |
| `review_needed` | Code/design ready for review | 👀 |
| `stale` | No activity for extended period | ⏰ |
| `deadline` | Approaching hard deadline | ⚠️ |
| `request` | Pending FD request needs attention | 📬 |
| `error` | Something failed and needs fixing | ❌ |

## Priority Levels

| Priority | Criteria | Dashboard Display |
|----------|----------|-------------------|
| `high` | Blocking, urgent, deadline today | Red, top of list |
| `medium` | Important, should address soon | Yellow |
| `low` | Nice to address when convenient | Gray |

## When to Add Flags

### Blocked
```json
{
  "type": "blocked",
  "priority": "high",
  "message": "Need database schema design",
  "since": "2026-03-10T12:00:00Z"
}
```
Add when:
- Waiting on external dependency
- Need decision before continuing
- Blocked by another task

### Review Needed
```json
{
  "type": "review_needed",
  "priority": "medium",
  "message": "PR #42 ready for review",
  "since": "2026-03-10T12:00:00Z"
}
```
Add when:
- Code ready for review
- Design doc ready for feedback
- Feature complete, needs approval

### Stale
```json
{
  "type": "stale",
  "priority": "low",
  "message": "No activity in 7 days",
  "since": "2026-03-10T12:00:00Z"
}
```
Add when:
- No commits in 7+ days
- Sprint not updated
- Issues not addressed

### Deadline
```json
{
  "type": "deadline",
  "priority": "high",
  "message": "MVP due March 15",
  "since": "2026-03-10T12:00:00Z"
}
```
Add when:
- Hard deadline approaching (< 3 days)
- Milestone due soon
- External commitment

### Request
```json
{
  "type": "request",
  "priority": "medium",
  "message": "2 pending FD requests",
  "since": "2026-03-10T12:00:00Z"
}
```
Add when:
- Unprocessed items in `.gc/requests.jsonl`
- Waiting for FD response

## Clearing Flags

Remove from `attention` array when:
- Blocker resolved
- Review completed
- Activity resumed
- Deadline passed or met
- Request processed

## Example: Multiple Flags

```json
{
  "attention": [
    {
      "type": "blocked",
      "priority": "high",
      "message": "API design decision needed",
      "since": "2026-03-10T10:00:00Z"
    },
    {
      "type": "request",
      "priority": "medium",
      "message": "Commit request pending",
      "since": "2026-03-10T11:00:00Z"
    }
  ]
}
```

## Dashboard Display

Hangar shows attention count per project:
```
PROJECT       PHASE      STATUS    ATTN
► NotifAI     features   ●active   ⚠2
  OtherProj   design     ○idle
```

Details shown when project selected.
