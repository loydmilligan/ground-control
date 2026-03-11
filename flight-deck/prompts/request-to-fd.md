# Request to Flight Deck Prompt

Use this prompt when a project Claude needs to request work from Flight Deck.

## How to Send a Request

Append a JSON line to `.gc/requests.jsonl`:

```jsonl
{"id":"req_TIMESTAMP","type":"TYPE","summary":"Brief description","payload":{},"status":"pending","at":"2026-03-10T12:00:00Z"}
```

## Request Types

### Commit Request
When code is ready to be committed:
```jsonl
{"id":"req_20260310120000","type":"commit","summary":"Add user authentication","payload":{"files":["src/auth.go","src/auth_test.go"],"message":"Add OAuth and email/password authentication"},"status":"pending","at":"2026-03-10T12:00:00Z"}
```

### Review Request
When you want code reviewed:
```jsonl
{"id":"req_20260310120001","type":"review","summary":"Review auth implementation","payload":{"files":["src/auth.go"],"focus":"Security and error handling"},"status":"pending","at":"2026-03-10T12:00:00Z"}
```

### Documentation Request
When docs need to be written:
```jsonl
{"id":"req_20260310120002","type":"docs","summary":"Document auth API","payload":{"scope":"API reference for auth endpoints","files":["src/auth.go"]},"status":"pending","at":"2026-03-10T12:00:00Z"}
```

### Decision Request
When you need a design/architecture decision:
```jsonl
{"id":"req_20260310120003","type":"decision","summary":"OAuth vs JWT for session management","payload":{"options":["oauth","jwt","both"],"context":"Need to decide auth token strategy"},"status":"pending","at":"2026-03-10T12:00:00Z"}
```

### Help Request
When you're blocked:
```jsonl
{"id":"req_20260310120004","type":"help","summary":"Blocked on database schema design","payload":{"reason":"Not sure how to model user relationships","need":"Schema design guidance"},"status":"pending","at":"2026-03-10T12:00:00Z"}
```

## Request Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique ID (req_TIMESTAMP) |
| `type` | Yes | commit, review, docs, decision, help |
| `summary` | Yes | Brief description (< 100 chars) |
| `payload` | No | Type-specific additional data |
| `status` | Yes | Always "pending" when creating |
| `at` | Yes | ISO timestamp |

## Status Values

- `pending` - Waiting for FD to process
- `processing` - FD is working on it
- `completed` - FD has finished

## When to Request

- Code is ready → `commit`
- Want feedback → `review`
- Need docs → `docs`
- Need decision → `decision`
- Stuck → `help`

## Important

- Don't commit directly - request a commit
- Don't write cross-project docs - request docs
- Don't spin wheels - request help early
