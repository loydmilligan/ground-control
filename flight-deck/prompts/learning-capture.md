# Learning Capture Prompt

Use this prompt to capture process friction, failures, and improvement ideas.

## Why Capture Learnings?

Every friction point is an improvement opportunity. By capturing these moments:
- We identify what's not working
- We gather ideas for improvement
- We build a better workflow over time

## How to Capture

Append a JSON line to `.gc/learning.jsonl`:

```jsonl
{"id":"learn_TIMESTAMP","type":"TYPE","actor":"ACTOR","summary":"Brief description","detail":"Detailed explanation","processed":false,"at":"2026-03-10T12:00:00Z"}
```

## Learning Types

### Friction
When something feels awkward or unclear:
```jsonl
{"id":"learn_20260310120000","type":"friction","actor":"proj_cc","summary":"Commit process unclear","detail":"Wasn't sure if I should commit directly or wait for FD to do it. The workflow isn't documented.","processed":false,"at":"2026-03-10T12:00:00Z"}
```

### Process Failure
When something went wrong:
```jsonl
{"id":"learn_20260310120001","type":"process_failure","actor":"user","summary":"Lost work due to unclear handoff","detail":"Made changes in FD that should have been in project. No clear guidance on where to work.","processed":false,"at":"2026-03-10T12:00:00Z"}
```

### Idea
When you have an improvement suggestion:
```jsonl
{"id":"learn_20260310120002","type":"idea","actor":"fd_cc","summary":"Auto-detect stale issues","detail":"Could automatically flag issues that haven't been updated in 7+ days as potentially stale.","processed":false,"at":"2026-03-10T12:00:00Z"}
```

## Actor Values

| Actor | Who |
|-------|-----|
| `user` | Human user (Matt) |
| `fd_cc` | Flight Deck Claude |
| `proj_cc` | Project Claude |

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique ID (learn_TIMESTAMP) |
| `type` | Yes | friction, process_failure, idea |
| `actor` | Yes | Who noticed this |
| `summary` | Yes | Brief description (< 100 chars) |
| `detail` | No | Detailed explanation |
| `processed` | Yes | Always false when creating |
| `at` | Yes | ISO timestamp |

## When to Capture

- Something feels awkward → friction
- Something broke → process_failure
- You think "this could be better" → idea
- You have to work around something → friction
- You're confused about what to do → friction

## Review Process

1. Learnings accumulate in `.gc/learning.jsonl`
2. During weekly review, FD triages by impact
3. High-impact items become improvement tasks
4. Processed learnings marked `"processed": true`

## Remember

- Capture in the moment - details fade
- No judgment - all friction is valid
- Short summaries - detail in detail field
- Better to over-capture than miss insights
