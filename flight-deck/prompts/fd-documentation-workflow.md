# Flight Deck Documentation Workflow

Use this prompt when writing documentation in Flight Deck for any project.

## When FD Does Documentation

- API documentation
- README updates
- Architecture docs
- User guides
- Changelog updates
- Setup/installation docs

## Workflow

### 1. Receive Documentation Request

From project's `.gc/requests.jsonl`:
```jsonl
{"type":"docs","summary":"Document auth API","payload":{"scope":"API reference for auth endpoints","files":["src/auth.go"]},"status":"pending","at":"..."}
```

### 2. Understand the Scope

Determine where docs should live:

| Doc Type | Location |
|----------|----------|
| API Reference | Project repo (docs/ or inline) |
| Architecture | Project repo or FD artifacts |
| README | Project repo |
| User Guide | Project repo |
| Internal Notes | FD artifacts |

### 3. Research

- Read the code being documented
- Check existing docs for style/format
- Identify what users need to know

### 4. Write Documentation

#### For Project Repo Docs

Create draft in `flight-deck/artifacts/` first, then dispatch to project:

```markdown
# [Documentation Title]

## Overview

[Brief description of what this documents]

## [Main Content Sections]

...
```

#### Documentation Principles

- **Audience First**: Who is reading this?
- **Progressive Disclosure**: Simple first, details later
- **Examples**: Show, don't just tell
- **Current**: Matches actual code behavior
- **Scannable**: Headers, bullets, code blocks

### 5. API Documentation Template

```markdown
# [API/Feature Name] API

## Overview

[What this API does]

## Authentication

[How to authenticate]

## Endpoints

### [Method] /path/to/endpoint

[Brief description]

**Request**
```json
{
  "field": "value"
}
```

**Response**
```json
{
  "result": "value"
}
```

**Errors**

| Code | Description |
|------|-------------|
| 400 | Bad request |
| 401 | Unauthorized |

**Example**
```bash
curl -X POST https://api.example.com/endpoint \
  -H "Authorization: Bearer TOKEN" \
  -d '{"field": "value"}'
```
```

### 6. README Template

```markdown
# Project Name

> One-line description

## Features

- Feature 1
- Feature 2

## Quick Start

```bash
# Installation
npm install project-name

# Usage
project-name --help
```

## Documentation

- [API Reference](docs/api.md)
- [Configuration](docs/config.md)

## Contributing

[How to contribute]

## License

[License info]
```

### 7. Save Draft

Save to `flight-deck/artifacts/`:
```
flight-deck/artifacts/
└── docs-{project}-{topic}-{date}.md
```

### 8. Dispatch to Project

If docs belong in project repo, add to project's `.gc/inbox/`:

```json
{
  "type": "docs_ready",
  "draft_path": "flight-deck/artifacts/docs-notifai-auth-api-20260310.md",
  "target_path": "docs/auth-api.md",
  "summary": "Auth API documentation ready for review",
  "at": "2026-03-10T12:00:00Z"
}
```

### 9. Update Request Status

```jsonl
{"id":"req_001","type":"docs",...,"status":"completed","result":"Draft ready: docs-notifai-auth-api-20260310.md"}
```

## Documentation Checklist

- [ ] Accurate (matches actual behavior)
- [ ] Complete (covers all features/endpoints)
- [ ] Clear (easy to understand)
- [ ] Examples (working code samples)
- [ ] Formatted (consistent style)
- [ ] Linked (references other docs)
- [ ] Versioned (notes version if relevant)

## Tips

- Use the project's existing doc style
- Include copy-pasteable examples
- Anticipate common questions
- Keep it updated as code changes
- Link to related docs/resources
