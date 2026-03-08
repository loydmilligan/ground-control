# Code Review Feedback

**Task**: 5.2 Notification Cards in Focused Mode
**Iteration**: 1
**Time**: 2026-03-07T03:48:01-08:00
**Decision**: NEEDS_REVISION

## Feedback

**: The implementation has two issues that need to be addressed:

1. **Reply/Snooze buttons will render but be non-functional**: When `mode="clean"` is used, the `NotificationCardExpandable` component renders Reply and Snooze buttons (lines 574-579). Since `FocusedFeed.tsx` doesn't pass `onReply` or `onSnooze` handlers, clicking these buttons will do nothing (silent failure). 

   **Fix options:**
   - Option A: Pass no-op handlers that log a message or show a toast indicating "Coming in a future release"
   - Option B: Check if `NotificationCardExpandable` can be modified to conditionally hide these buttons when handlers aren't provided
   - Option C: Add a new mode like `"clean-minimal"` that hides Reply/Snooze

2. **`data-notification-card` attribute is ineffective**: The attribute passed to `NotificationCardExpandable` won't reach the DOM because the component doesn't spread props to its root element. If this attribute is needed to prevent carousel swipe conflicts (as mentioned in implementation notes), it needs to be handled differently.

   **Fix options:**
   - Add a `className` or `containerProps` prop to `NotificationCardExpandable` that forwards attributes
   - Or handle the swipe conflict prevention at a different level (e.g., wrapping div in `FocusedFeed`)

**Action required**: Address the Reply/Snooze button issue - users will see these buttons but they won't work, which is a poor UX. The `data-notification-card` issue should be verified if it's actually needed for swipe conflict prevention.