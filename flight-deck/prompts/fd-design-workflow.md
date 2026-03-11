# Flight Deck Design Workflow

Use this prompt when doing design work in Flight Deck for any project.

## When FD Does Design

- Architecture decisions
- API design
- Data model design
- UI/UX design documents
- System design docs
- Feature specifications

## Workflow

### 1. Receive Design Request

From project's `.gc/requests.jsonl`:
```jsonl
{"type":"decision","summary":"Need auth system design","payload":{"scope":"Full auth flow including OAuth"},"status":"pending","at":"..."}
```

### 2. Create Design Document

Create in `flight-deck/artifacts/designs/`:

```
flight-deck/artifacts/designs/
└── {project}-{topic}-{date}.md
```

Example: `notifai-auth-system-20260310.md`

### 3. Design Document Template

```markdown
# Design: [Topic]

**Project**: [project name]
**Date**: [date]
**Status**: Draft | Review | Approved
**Request ID**: [from requests.jsonl]

## Context

[Why this design is needed]

## Requirements

- [ ] Requirement 1
- [ ] Requirement 2

## Options Considered

### Option A: [Name]

**Description**: [How it works]

**Pros**:
- Pro 1
- Pro 2

**Cons**:
- Con 1

### Option B: [Name]

**Description**: [How it works]

**Pros**:
- Pro 1

**Cons**:
- Con 1
- Con 2

## Recommendation

[Which option and why]

## Implementation Notes

[Key details for implementer]

## Open Questions

- [ ] Question 1
- [ ] Question 2

## Decision

**Chosen**: [Option X]
**Rationale**: [Why]
**Approved by**: [user/date]
```

### 4. Save Decision Record

Also save to `flight-deck/artifacts/decisions/`:

```markdown
# Decision: [Short title]

**Date**: [date]
**Project**: [project]
**Category**: architecture | api | data | ui

## Context

[Brief context]

## Decision

[What was decided]

## Consequences

[What this means for implementation]
```

### 5. Update Request Status

In project's `.gc/requests.jsonl`, mark as completed:
```jsonl
{"id":"req_001","type":"decision",...,"status":"completed","result":"See design doc: notifai-auth-system-20260310.md"}
```

### 6. Notify Project

If project has active session, send message about design completion.

Or add to project's `.gc/inbox/`:
```json
{
  "type": "design_complete",
  "design_doc": "flight-deck/artifacts/designs/notifai-auth-system-20260310.md",
  "summary": "Auth system design completed. Recommending OAuth + JWT.",
  "at": "2026-03-10T12:00:00Z"
}
```

## Tips

- Include diagrams where helpful (ASCII or Mermaid)
- Be specific about implementation details
- Highlight any constraints or limitations
- Note dependencies on other work
- Consider security implications
- Think about future extensibility
