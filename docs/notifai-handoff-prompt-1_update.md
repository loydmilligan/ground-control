# Ground Control Task Updates Based on Designer Review

**Date:** 2026-03-07
**In response to:** `notifai-handoff-prompt-1_response.md`

---

## Summary

The NotifAI designer Claude reviewed the Phase 1-2 sprint tasks and provided detailed feedback. This document explains how each recommendation was incorporated into the Ground Control tasks.

---

## Changes Applied

### 1. Task Splits

#### Task 1.3 → 1.3a + 1.3b

**Designer feedback:** Position dots should go in the Header component, not standalone. Split recommended.

**Action taken:**
- Renamed `1.3 Navigation Controls` → `1.3a Navigation Controls (Arrows & Keyboard)`
- Created new task `1.3b Header Position Dots`
- 1.3a focuses on arrow buttons and keyboard navigation
- 1.3b focuses on header integration and dot indicators

#### Task 1.4 → Merged into 1.1

**Designer feedback:** Scroll position management is tightly coupled with carousel container.

**Action taken:**
- Merged scroll position requirements into Task 1.1
- Marked Task 1.4 as completed with note "MERGED INTO TASK 1.1"
- Task 1.1 now includes: "Track scroll position per stage with refs"

#### Task 2.4 Simplified (no split needed)

**Designer feedback:** Split API work from UI work.

**Action taken:**
- Since we're using the existing `enabled` field (per API recommendation), no backend changes are needed
- Task 2.4 is now purely frontend: "Channel Search/Filter UI"
- Removed API changes from task description

---

### 2. API Decision: `enabled` vs `is_blocked`

**Designer feedback:** `is_blocked` does NOT exist. Recommend using existing `enabled` field to avoid migration.

**Action taken:**
- Updated Task 2.1 to reference `enabled=false` for blocked state
- Updated Task 2.2 to use:
  - `enabled` field instead of `is_blocked`
  - Existing endpoint: `PATCH /api/channels/{id}` with `{enabled: false}`
  - No migration required
- Removed all references to `is_blocked` from task descriptions

---

### 3. Context Enrichment

**Designer feedback:** Provided list of critical files and patterns.

**Action taken:** Added `context.background` to all tasks with:

| Task | Context Added |
|------|---------------|
| 1.1 | Files: Dashboard.tsx, NotificationCardExpandable.tsx (swipe reference), lib/utils.ts. Patterns: localStorage prefix `notifai-`, swipe threshold 80px |
| 1.2 | Dark theme colors (bg-slate-900, bg-slate-800). Peek clarification: thin sliver NOT preview card |
| 1.3a | Reference NotificationCardExpandable.tsx for swipe patterns. Constraints: min 44px touch targets |
| 1.3b | Files: Header.tsx (dots go here), FeedCarousel.tsx (goToStage method) |
| 1.5 | WARNING: Dashboard.tsx is 754 lines with complex state |
| 2.1 | Use existing `enabled` field. Show ALL channels including 0-notification ones |
| 2.2 | Toast via sonner library. Optimistic UI updates |
| 2.3 | Peek = thin vertical sliver, not minimap |
| 2.4 | Pure frontend task - filter client-side |

---

### 4. Acceptance Criteria Added

**Designer feedback:** Provided detailed acceptance criteria with reject conditions.

**Action taken:** Added `context.requirements` to all tasks based on designer's criteria:

#### Task 1.1 Requirements:
- Horizontal scroll with snap
- Touch swipe works (no jank)
- Mouse drag works
- goToStage(n) scrolls smoothly
- Persist stage to `notifai-carousel-stage`
- Restore stage on load
- Track scroll position per stage with refs

#### Task 1.2 Requirements:
- Each stage fills viewport width minus peek space
- Stage titles visible at top
- Background tint subtle but perceptible
- Peek shows 20-40px of adjacent stages
- Peek is tappable to navigate

#### Task 1.3a Requirements:
- Arrow buttons visible on desktop only
- Arrow click navigates one stage
- Arrow keys work only when carousel focused
- Hidden on mobile

#### Task 1.3b Requirements:
- Dots in header component
- Current stage highlighted
- Dots tappable
- Calls goToStage()

#### Task 1.5 Requirements:
- ViewSwitcher removed from Dashboard
- Carousel renders in Dashboard
- No TypeScript errors
- App builds and runs

#### Task 2.1 Requirements:
- Vertical scrollable list (NOT grid)
- Each row: icon, name, count, toggle
- Blocked channels visually muted
- Styling matches notification cards
- Show ALL channels including those with 0 notifications

#### Task 2.2 Requirements:
- Tap toggles blocked state
- Uses `enabled` field (NOT `is_blocked`)
- Blocked channels move to bottom
- Toast shows channel name and impact count
- API call `PATCH /api/channels/{id}`
- Optimistic UI update

#### Task 2.3 Requirements:
- Peek shows 20-40px of Stage 1
- Notification count visible
- Tapping peek navigates to Stage 1

#### Task 2.4 Requirements:
- Search input at top
- Filters by name case-insensitive
- Optional: filter by channel_type
- UI does not block content

---

### 5. Designer's Open Questions - Resolved

| Question | Designer's Answer | Applied To |
|----------|-------------------|------------|
| Should blocked channels show notification cards? | No - Stage 0 shows channel rows only | Task 2.1 description |
| When blocked, do notifications disappear immediately? | Yes - optimistic UI update | Task 2.2 requirements |
| Should Stage 0 show ALL channels? | Yes - including 0-notification channels | Task 2.1 requirements |

---

### 6. Risks Noted (for implementer awareness)

| Risk | Mitigation |
|------|------------|
| Dashboard complexity (754 lines) | Task 1.5: Keep old components until carousel stable |
| Swipe conflict (card vs carousel) | Carousel swipe only on empty space or headers |
| Mobile performance | Consider lazy-loading non-visible stages |
| Peek misinterpretation | Clarified in all peek-related tasks: "thin sliver NOT preview card" |

---

## Final Sprint State

| Task | Status | Notes |
|------|--------|-------|
| 1.1 Create Carousel Container | Pending | Now includes scroll management |
| 1.2 Create Stage Wrapper | Pending | Peek clarified |
| 1.3a Navigation (Arrows/Keyboard) | Pending | Split from original 1.3 |
| 1.3b Header Position Dots | Pending | New task from split |
| 1.4 Scroll Position | Completed | Merged into 1.1 |
| 1.5 Integrate with Dashboard | Pending | Risk noted |
| 2.1 ChannelBlockList | Pending | Shows ALL channels |
| 2.2 Block/Unblock | Pending | Uses `enabled` field |
| 2.3 Peek Integration | Pending | Thin sliver clarified |
| 2.4 Search/Filter UI | Pending | Frontend only |

**Total: 10 tasks (9 pending, 1 merged/completed)**

---

## Ready for Implementation

All designer feedback has been incorporated. Tasks now include:
- Accurate descriptions matching designer intent
- Rich context with files and patterns
- Clear acceptance criteria
- Resolved API decisions
- Risk awareness

Proceed with orchestration.
