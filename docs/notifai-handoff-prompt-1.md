# Ground Control Integration Test: NotifAI Carousel UI

## Context

I'm testing a task orchestration system called **Ground Control** by using it to manage the implementation of the NotifAI Carousel UI redesign (Phases 1 & 2). You (the Claude instance that designed the carousel) will serve as the domain expert and quality reviewer.

A separate Claude Code instance (the "implementer") will execute the tasks using Ground Control's orchestration. This document is a handoff to you for review before implementation begins.

---

## What is Ground Control?

Ground Control is a CLI/TUI task management system where:
- Tasks are defined with structured context, requirements, and verification criteria
- An orchestration pipeline executes tasks through stages (planning, coding, testing, review)
- Work is verified before being marked complete
- A learning log captures issues for continuous improvement

The goal of this test is to validate that Ground Control can successfully orchestrate work on a real project (NotifAI) that was designed by a different Claude instance.

---

## Sprint: NotifAI Carousel UI Phase 1-2

**Goal:** Build carousel foundation and blocked channels stage

### Phase 1: Foundation & Carousel Shell (5 tasks)

| Task ID | Title | Description |
|---------|-------|-------------|
| 1.1 | Create Carousel Container (FeedCarousel.tsx) | Create FeedCarousel.tsx component with horizontal scroll, snap points, touch swipe and mouse drag support. Track current stage in state and persist to localStorage. Expose goToStage(n) method. |
| 1.2 | Create Stage Wrapper (CarouselStage.tsx) | Create CarouselStage.tsx component accepting stage prop (0-3). Apply stage-specific backgrounds and display stage titles. Show peek of adjacent stages and support expand/collapse. |
| 1.3 | Navigation Controls | Left/right arrow buttons (desktop only), swipe gestures (mobile), keyboard navigation. Position indicator dots in header. |
| 1.4 | Scroll Position Management | Track and restore scroll position per stage independently using refs. |
| 1.5 | Integrate Carousel with Dashboard | Replace ViewSwitcher with FeedCarousel in Dashboard. Remove old RawInfoView, CleanView usage. Update routing if needed. |

### Phase 2: Blocked Channels Stage (4 tasks)

| Task ID | Title | Description |
|---------|-------|-------------|
| 2.1 | Create ChannelBlockList Component | Create ChannelBlockList.tsx displaying channels as scrollable list. Each row shows icon, name, notification count, block toggle. Blocked channels have muted/opaque styling with notification-card-like design. |
| 2.2 | Block/Unblock Interaction | Tap to toggle blocked state, blocked channels slide to bottom. Show toast confirmation with impact count. API call to update channel is_blocked flag. |
| 2.3 | Peek Integration for Blocked Channels | Right edge shows peek of Channel Rules stage with notification count. Tapping peek navigates to Stage 1. |
| 2.4 | Channel Search/Filter | Search box at top to filter channel list. Filter by channel type (email, chat, system, etc.). API: Add is_blocked field to Channel model, endpoint to toggle block status. |

---

## Our Process

1. **Task Extraction**: I read your TASKS_CAROUSEL_UI.md document and created Ground Control tasks from Phases 1 & 2
2. **Sprint Creation**: Tasks are grouped into a sprint with a clear goal
3. **Orchestration**: The implementer Claude will execute each task through Ground Control's pipeline
4. **Verification**: Each task has verification criteria (human approval in this case)
5. **Learning Capture**: Any issues, rework, or deviations will be logged

---

## Request for Your Review

Before implementation begins, please review the following and provide feedback:

### 1. Task Accuracy
- Do the task descriptions accurately capture what you designed?
- Are there any missing requirements or constraints?
- Are there any dependencies between tasks that should be explicit?

### 2. Task Granularity
- Are the tasks appropriately sized?
- Should any be split further or combined?
- Is the ordering logical?

### 3. Missing Context
- What existing components/patterns should the implementer know about?
- Are there design decisions or constraints not captured in the task descriptions?
- What files/components will these tasks need to interact with?

### 4. Acceptance Criteria
- For each task, what would YOU check to verify it's done correctly?
- Are there edge cases or specific behaviors that must work?
- What would make you reject an implementation?

### 5. Risks or Concerns
- Do you see any risks with this approach?
- Are there parts of the design that might be misinterpreted?
- What questions would you want to ask before starting?

### 6. API/Backend Requirements
- Task 2.2 and 2.4 mention API changes (is_blocked field, toggle endpoint)
- Are these already implemented, or do they need to be created?
- Are there existing patterns for channel mutations we should follow?

---

## What Happens Next

Based on your feedback:
1. I will update the Ground Control tasks with any corrections or additional context
2. The implementer Claude will execute the tasks
3. After implementation, I will send you a second prompt with:
   - Summary of what was implemented
   - Any issues encountered
   - Any deviations from the plan
   - Request for you to verify the implementation against YOUR original design

The goal is to test whether Ground Control can successfully bridge the gap between a designer Claude and an implementer Claude, with the human (Matt) orchestrating via the CLI.

---

**Please provide your review and any adjustments needed before we proceed with implementation.**
