# Flight Deck Project Analysis

You are part of **Flight Deck**, an AI development orchestration system that manages Claude sessions across multiple projects.

## Your Task

Analyze this repository thoroughly and generate a structured analysis that will be used to:
1. Configure how AI agents work with this project
2. Set up appropriate constraints and conventions
3. Track the project in the Flight Deck dashboard

## Analysis Instructions

Examine the following (check all that exist):

### Package/Dependency Files
- `package.json`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`
- `go.mod`, `go.sum`
- `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile`
- `Cargo.toml`
- `build.gradle`, `pom.xml`

### Configuration Files
- `.eslintrc*`, `.prettierrc*`, `biome.json`
- `tsconfig.json`, `jsconfig.json`
- `pytest.ini`, `setup.cfg`, `tox.ini`
- `Makefile`, `justfile`
- `.editorconfig`

### CI/CD
- `.github/workflows/`
- `.gitlab-ci.yml`
- `.circleci/`
- `Jenkinsfile`

### Documentation
- `README.md`, `CONTRIBUTING.md`
- `docs/` directory
- `CLAUDE.md` or similar AI instructions

### Project Structure
- Source directories (`src/`, `lib/`, `app/`, `internal/`)
- Test directories (`test/`, `tests/`, `__tests__/`, `*_test.go`)
- Monorepo indicators (`packages/`, `apps/`, `workspaces` in package.json)

## Output Format

Output valid JSON matching this schema to `.gc/analysis.json`:

```json
{
  "name": "project-name",
  "description": "Brief description of what the project does",
  "languages": [
    {"name": "TypeScript", "confidence": 0.95, "version": "5.0"}
  ],
  "frameworks": [
    {"name": "React Native", "confidence": 0.90, "version": "0.72"}
  ],
  "test_runner": {
    "name": "Jest",
    "confidence": 0.85,
    "config_file": "jest.config.js"
  },
  "package_manager": "npm",
  "ci_system": "github-actions",
  "existing_conventions": [
    "ESLint with Airbnb config",
    "Prettier for formatting",
    "Husky pre-commit hooks"
  ],
  "existing_task_management": "Linear integration via package.json scripts",
  "key_files": [
    {"path": "src/index.ts", "purpose": "Main entry point"},
    {"path": "src/config.ts", "purpose": "Configuration constants"}
  ],
  "suggested_constraints": [
    "Run 'npm run lint' before committing",
    "Never modify generated files in dist/",
    "Use existing component patterns in src/components/"
  ],
  "monorepo": false,
  "workspaces": []
}
```

## Confidence Guidelines

- **0.9-1.0**: Explicit declaration found (e.g., in package.json dependencies)
- **0.7-0.9**: Strong indicators (file extensions, imports, config files)
- **0.5-0.7**: Inferred from context (directory structure, comments)
- **Below 0.5**: Uncertain, flag for human review

## Important

1. **Be thorough** — Check multiple sources to confirm detections
2. **Be honest about confidence** — Don't inflate scores
3. **Note conflicts** — If conventions seem inconsistent, mention in suggested_constraints
4. **Focus on AI relevance** — Key files should be ones an AI agent needs to understand

Output ONLY the JSON to `.gc/analysis.json`. No other text or explanation.
