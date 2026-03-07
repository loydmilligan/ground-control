# Relevant Decisions

## Chose Go + Bubble Tea for CLI/TUI tech stack

**When**: 2026-03-06

**Why**: TUI and CLI will be built together. Go + Bubble Tea offers best-in-class TUI experience, single binary distribution, and the Charm ecosystem is actively thriving. Accepted tradeoff of more verbose JSON handling and separate language from potential future web UI.

**Factors**:
- TUI quality is a priority — building TUI and CLI simultaneously
- Single binary distribution preferred
- Charm ecosystem (Bubble Tea, Lip Gloss, Huh) is actively developed
- Willing to accept Go's verbose JSON handling
- Web UI is future/optional, can be separate codebase

---

