# Flight Deck Review Workflow

Use this prompt when doing code/PR reviews in Flight Deck for any project.

## When FD Does Reviews

- Code review requests from projects
- PR reviews
- Design document reviews
- Architecture reviews

## Workflow

### 1. Receive Review Request

From project's `.gc/requests.jsonl`:
```jsonl
{"type":"review","summary":"Review auth implementation","payload":{"files":["src/auth.go","src/auth_test.go"],"focus":"Security and error handling"},"status":"pending","at":"..."}
```

### 2. Read the Code

Navigate to project and read the specified files:
```bash
cd /path/to/project
# Read files listed in payload
```

### 3. Create Review Document

Create in `flight-deck/artifacts/reviews/`:

```
flight-deck/artifacts/reviews/
└── {project}-{topic}-{date}.md
```

Example: `notifai-auth-review-20260310.md`

### 4. Review Document Template

```markdown
# Code Review: [Topic]

**Project**: [project name]
**Date**: [date]
**Reviewer**: FD Claude
**Request ID**: [from requests.jsonl]

## Files Reviewed

- `path/to/file1.go`
- `path/to/file2.go`

## Summary

[Overall assessment: Approved / Changes Requested / Needs Discussion]

## Findings

### Critical (Must Fix)

#### [Issue Title]
**File**: `path/to/file.go:123`
**Issue**: [Description]
**Suggestion**: [How to fix]

### Important (Should Fix)

#### [Issue Title]
**File**: `path/to/file.go:45`
**Issue**: [Description]
**Suggestion**: [How to fix]

### Minor (Nice to Have)

#### [Issue Title]
**File**: `path/to/file.go:78`
**Issue**: [Description]
**Suggestion**: [How to fix]

## Positive Notes

- [Good patterns observed]
- [Well-structured code]
- [Good test coverage]

## Questions

- [ ] [Clarifying question about implementation]

## Verdict

**Status**: Approved | Changes Requested | Needs Discussion
**Next Steps**: [What should happen next]
```

### 5. Review Checklist

- [ ] **Correctness**: Does the code do what it's supposed to?
- [ ] **Security**: Any vulnerabilities? Input validation? Auth checks?
- [ ] **Error Handling**: Are errors handled appropriately?
- [ ] **Performance**: Any obvious performance issues?
- [ ] **Tests**: Are there adequate tests? Do they cover edge cases?
- [ ] **Readability**: Is the code clear and maintainable?
- [ ] **Patterns**: Does it follow project patterns?
- [ ] **Documentation**: Are complex parts documented?

### 6. Update Request Status

```jsonl
{"id":"req_001","type":"review",...,"status":"completed","result":"Review complete. Changes requested. See: notifai-auth-review-20260310.md"}
```

### 7. Notify Project

Add to project's `.gc/inbox/`:
```json
{
  "type": "review_complete",
  "review_doc": "flight-deck/artifacts/reviews/notifai-auth-review-20260310.md",
  "verdict": "changes_requested",
  "summary": "2 critical issues, 3 minor suggestions. See review doc.",
  "at": "2026-03-10T12:00:00Z"
}
```

## Review Guidelines

### Be Constructive
- Explain *why* something is an issue
- Provide concrete suggestions
- Acknowledge good work

### Prioritize
- Focus on critical/security issues first
- Don't nitpick style if there are bigger problems
- Be pragmatic about scope

### Consider Context
- Is this MVP or production code?
- What's the timeline?
- What are the constraints?
