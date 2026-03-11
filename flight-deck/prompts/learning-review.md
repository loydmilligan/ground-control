# Learning Review Prompt

Use this prompt during weekly reviews to triage learnings and identify improvements.

## Purpose

Review accumulated learnings from all projects to:
- Identify high-impact improvement opportunities
- Convert learnings to actionable tasks
- Archive processed learnings

## Process

### 1. Gather Learnings

Collect `.gc/learning.jsonl` from all projects:
```bash
gc sync  # Aggregates project state including learnings
```

Or manually review each project's learning log.

### 2. Categorize by Impact

Rate each learning:

| Impact | Criteria |
|--------|----------|
| **High** | Blocks work, causes errors, frequent friction |
| **Medium** | Slows work, causes confusion, occasional friction |
| **Low** | Minor annoyance, rare occurrence, nice-to-fix |

### 3. Triage Template

For each learning, decide:

```
Learning: [summary]
Type: friction | process_failure | idea
Actor: user | fd_cc | proj_cc
Impact: high | medium | low

Analysis:
- What caused this?
- How often does it happen?
- What's the fix?

Decision:
[ ] Create improvement task
[ ] Update documentation
[ ] Add to backlog
[ ] Won't fix (explain why)
[ ] Already addressed
```

### 4. Create Improvement Tasks

For high-impact items, create tasks:

```json
{
  "title": "Fix: [learning summary]",
  "type": "simple",
  "description": "Address learning: [detail]",
  "context": {
    "background": "From learning review: [learning id]",
    "requirements": ["specific fix needed"]
  }
}
```

### 5. Mark as Processed

Update learning entries:
```jsonl
{"id":"learn_001",...,"processed":true,"resolution":"Created task task_123"}
```

## Review Checklist

- [ ] Collected learnings from all projects
- [ ] Categorized by type (friction, failure, idea)
- [ ] Rated by impact (high, medium, low)
- [ ] High-impact items have action plans
- [ ] Improvement tasks created where needed
- [ ] Processed learnings marked
- [ ] Patterns identified across projects

## Patterns to Watch For

- **Same friction in multiple projects** → Systemic issue
- **Repeated process failures** → Process needs redesign
- **Cluster of ideas around one area** → Opportunity for improvement
- **User friction** → Highest priority (affects human experience)

## Output

After review, create a summary:

```markdown
# Learning Review - [Date]

## High Impact (Action Required)
- [Learning 1] → Created task #X
- [Learning 2] → Updated docs

## Medium Impact (Backlogged)
- [Learning 3] → Added to improvement backlog

## Low Impact (Noted)
- [Learning 4] → Will monitor

## Patterns Observed
- [Pattern description]

## Process Changes
- [Any immediate process changes made]
```
