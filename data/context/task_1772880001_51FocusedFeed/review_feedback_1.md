# Code Review Feedback

**Task**: 5.1 Create FocusedFeed Component
**Iteration**: 1
**Time**: 2026-03-07T03:25:08-08:00
**Decision**: NEEDS_REVISION

## Feedback

**: The `onAction` prop is accepted but not used.

**Specific issues:**

1. **Unused `onAction` prop** (line 9): The component accepts `onAction` as a prop but renames it to `_onAction` and never passes it to `NotificationFeed`. Looking at `NotificationFeed.tsx`, it doesn't accept an `onAction` prop either - it relies on internal `NotificationCard` action handling.

   **Fix options:**
   - If `NotificationFeed` should support external action handlers, add the prop there and wire it through
   - If action handling is fully internal (via `useNotificationAction` hook as mentioned), remove `onAction` from `FocusedFeedProps` entirely since it's misleading API surface

2. **Minor**: The requirements say "Clean notification cards with actions" - need to verify `NotificationCard` actually renders action buttons. The current implementation relies on `NotificationCard` having built-in action handling, but without seeing that component I can't confirm actions are visible.

**Recommendation:** Either remove the unused `onAction` prop (if actions are fully internal), or wire it through to the notification cards. An unused prop creates a confusing API contract where consumers might pass it expecting it to work.