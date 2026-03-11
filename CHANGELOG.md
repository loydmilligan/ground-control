# Changelog

All notable changes to Ground Control will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Flight Deck TUI** (`gc fd`) - Multi-project Claude session orchestration dashboard
  - Mission Control view with project list, session status, and activity feed
  - Comms view for messaging active Claude sessions
  - Teleportation (`t`) to jump into active Claude sessions
  - F12 return binding to get back to Flight Deck
  - Session modes: window, pane, headless
- **Altitude system** - Configurable automation levels (Low/Mid/High)
  - Low: Human drives, AI assists (all operations require approval)
  - Mid: Balanced partnership (destructive ops require approval)
  - High: AI drives, human monitors (no approvals required)
- **Sidecar pattern** - `.gc/` directory per project for state management
  - `state.json` - Session state, costs, activity
  - `project.json` - Project configuration
  - `sessions/` - Session history
- **Project Registry** - Centralized tracking of adopted projects
- **File Watcher** - Live TUI updates via fsnotify
- **Cost Tracker** - Token and cost tracking per session
- **Session History** - Query past sessions with stats
- `gc adopt` command for adding external projects to Flight Deck
- `gc self-learn` command for analyzing learning-log.json patterns
- `gc history` command for viewing past orchestration sessions
- `gc sessions` command for session management (list, cleanup, cancel)
- `gc ingest` command for Claude Code log ingestion (AI Matt training)
- Supervised delegation mode with monitor pane (`gc delegate --supervised`)
- Password-protected approvals using SHA256 hash verification
- AI Matt restricted Claude environment (`.claude-ai-matt/`)
- Delegation watchdog for detecting stuck states
- Man page documentation (`man/gc.1`)
- MIT License file
- Context Engineer agent for automatic context bundle generation
- Working directory support for orchestrating external projects (`--add-dir`)
- `gc brainstorm` command for ideation sessions
- `gc delegate` and `gc handoff` commands for cross-agent collaboration
- `gc sprint` command for sprint management
- `gc artifact` command for artifact tracking
- `gc consult` command for consulting specialized agents
- `gc status` command for viewing current session state
- Session management with pause/resume support
- Makefile with install/uninstall targets

### Changed
- `GetDataDir()` now supports running `gc` from any directory
- Pipeline now auto-builds context bundles when missing
- Claude CLI wrapper now uses `--permission-mode` instead of `--dangerously-skip-permissions`
- Delegation monitor now watches both Worker and AI Matt panes for approvals

### Fixed
- Duplicate error messages in CLI (added SilenceErrors)
- Context bundle detection in sanity stage now uses sentinel error
- Watchdog no longer sends Escape key (was interrupting Claude)

## [0.1.0] - 2026-03-05

### Added
- Initial Ground Control CLI scaffold with Go + Cobra
- Core commands: `gc tasks`, `gc dump`, `gc create`, `gc process`, `gc orc`
- TUI interface with Bubble Tea
- Orchestration pipeline: Sanity → Coder → Reviewer → Tester → Commit
- Claude Code CLI integration for AI agent execution
- Task schema with verification requirements
- Brain dump ingestion workflow
- Activity logging for AI-Matt training
- Project and task management
- `gc standup` and `gc complete` commands

[Unreleased]: https://github.com/mmariani/ground-control/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/mmariani/ground-control/releases/tag/v0.1.0
