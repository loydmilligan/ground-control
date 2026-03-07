# Design Review: Ground Control Sprint Phases 1-2

**Reviewer:** Claude (Designer instance)
**Date:** 2026-03-07
**In response to:** `notifai-handoff-prompt-1.md`

---

## 1. Task Accuracy

**Overall:** The tasks accurately capture the design intent. A few refinements:

### Task 1.2 (CarouselStage) - Clarification needed
- "Peek of adjacent stages" should show a **thin vertical sliver** (~20-40px) of the next/previous stage content, not a preview card
- Peek should be **tappable** to navigate (mentioned in 2.3 but not here)

### Task 1.3 (Navigation) - Missing detail
- Position dots should go **in the header component**, not be standalone
- Keyboard navigation should only activate when carousel container is focused (not globally)

### Task 2.2 (Block/Unblock) - API field issue
**`is_blocked` does NOT exist** on Channel model. Current equivalents:
- `enabled` (boolean) - toggles channel on/off
- `status` ('active' | 'pending' | 'ignored')

**Decision needed:** Either:
- A) Repurpose `enabled=false` as "blocked" (simpler)
- B) Add new `is_blocked` field (more semantic but requires migration)

I recommend **Option A** for Phase 2 - treat `enabled=false` as blocked.

### Task 2.4 - Should be split
API work should be a separate task from UI work. See granularity below.

---

## 2. Task Granularity

### Recommended splits:

**Task 1.3 should split into:**
- 1.3a: Arrow buttons and keyboard navigation
- 1.3b: Header position dots (touches Header.tsx)

**Task 2.4 should split into:**
- 2.4a: Channel search/filter UI
- 2.4b: Backend API changes (if adding `is_blocked`)

### Recommended combines:
- Tasks 1.4 (scroll management) could merge into 1.1 (carousel container) since they're tightly coupled

### Task ordering is logical
1.1 → 1.2 → 1.3 → 1.4 → 1.5 is correct dependency order.

---

## 3. Missing Context

### Files the implementer MUST know about:

| File | Why |
|------|-----|
| `Dashboard.tsx` | Main integration point - 754 lines, complex state |
| `ViewSwitcher.tsx` | Will be replaced by carousel dots |
| `NotificationFeed.tsx` | Modes: 'focus' \| 'all' \| 'digest' \| 'timeline' |
| `NotificationCardExpandable.tsx` | Has existing swipe implementation (reference pattern) |
| `Header.tsx` | Position dots go here |
| `lib/api.ts` | Channel interface definition |
| `lib/utils.ts` | Has `cn()` for className merging |

### Existing patterns to follow:

1. **Swipe detection** - `NotificationCardExpandable.tsx` already has touch handlers with 80px threshold, vertical movement detection, and 300ms animations

2. **Toast system** - Use `sonner` library:
   ```typescript
   import { toast } from 'sonner';
   toast.success('Channel blocked');
   ```

3. **localStorage keys** - Prefix with `notifai-`:
   ```typescript
   localStorage.setItem('notifai-carousel-stage', '0');
   ```

4. **Scroll preservation** - RawInfoView/CleanView use refs to save/restore positions

### Design constraints not in tasks:

- **Minimum 44px touch targets** (WCAG requirement, see CLAUDE.md)
- **Dark theme colors**: bg-slate-900 (base), bg-slate-800 (cards)
- **Stage background tints** should be VERY subtle (5-10% opacity difference)

---

## 4. Acceptance Criteria

### Task 1.1 (FeedCarousel)
- [ ] Horizontal scroll with CSS `scroll-snap-type: x mandatory`
- [ ] Touch swipe works on mobile (no jank)
- [ ] Mouse drag works on desktop
- [ ] `goToStage(n)` scrolls smoothly to stage
- [ ] Current stage persists to `notifai-carousel-stage` in localStorage
- [ ] Restores stage on page load

**Reject if:** Scroll is janky, snap doesn't work, swipe fights with card swipe

### Task 1.2 (CarouselStage)
- [ ] Each stage fills viewport width minus peek space
- [ ] Stage titles visible at top (Blocked Channels, Channel Rules, AI Rules, Focused)
- [ ] Background tint is subtle but perceptible
- [ ] Peek shows ~20-40px of adjacent stages

**Reject if:** Stage titles missing, backgrounds are garish, peek is too wide

### Task 1.3 (Navigation)
- [ ] Arrows visible on desktop, hidden on mobile
- [ ] Arrow click navigates one stage
- [ ] Arrow keys work when carousel focused
- [ ] Position dots in header are tappable
- [ ] Current stage dot is highlighted

**Reject if:** Mobile shows arrows, dots are not in header, no visual indicator of current stage

### Task 1.4 (Scroll Position)
- [ ] Each stage maintains independent scroll position
- [ ] Switching stages and back restores position
- [ ] Uses refs (not localStorage)

**Reject if:** Scroll resets when navigating between stages

### Task 1.5 (Integration)
- [ ] ViewSwitcher removed from Dashboard
- [ ] Carousel renders in Dashboard
- [ ] No TypeScript errors
- [ ] App builds and runs

**Reject if:** Old ViewSwitcher still visible, build fails

### Task 2.1 (ChannelBlockList)
- [ ] Channels displayed as vertical scrollable list
- [ ] Each row: icon, name, notification count, block toggle
- [ ] Blocked channels visually muted (opacity 0.5-0.7)
- [ ] Styling matches notification cards (rounded corners, dark bg)

**Reject if:** Grid layout used, no visual distinction for blocked

### Task 2.2 (Block/Unblock)
- [ ] Tap toggles blocked state
- [ ] Blocked channels move to bottom (or separate section)
- [ ] Toast shows: "Gmail blocked - X notifications hidden"
- [ ] API call updates channel (using `enabled` field)

**Reject if:** No toast, no visual feedback, API not called

### Task 2.3 (Peek Integration)
- [ ] Right edge shows peek of Stage 1
- [ ] Peek displays notification count
- [ ] Tapping peek navigates to Stage 1

**Reject if:** Peek not tappable, no count shown

### Task 2.4 (Search/Filter)
- [ ] Search input at top of channel list
- [ ] Filters by name (case-insensitive)
- [ ] Optional: filter by channel_type

**Reject if:** Search doesn't filter, UI blocks content

---

## 5. Risks & Concerns

### High Risk: Dashboard complexity
Dashboard.tsx is 754 lines with complex state management. Task 1.5 (integration) needs careful execution. **Suggestion:** Keep old components until carousel is stable, then remove.

### Medium Risk: Swipe conflict
Card swipe (left=archive) vs. carousel swipe (left=next stage) could conflict. **Mitigation:** Carousel swipe should only work on empty space or stage headers, not on cards.

### Medium Risk: Mobile performance
Four full-width stages could be memory-heavy. **Mitigation:** Consider lazy-loading stages that aren't visible.

### Low Risk: Design misinterpretation
"Peek views" could be interpreted as full previews. **Clarification:** Peek = thin sliver showing edge of adjacent stage, NOT a minimap or preview card.

### Questions I would ask:
1. Should blocked channels in Stage 0 show their notification cards, or just the channel row?
2. When a channel is blocked, do its notifications immediately disappear from Stage 3, or on next refresh?
3. Should Stage 0 show ALL channels or only channels with notifications?

---

## 6. API/Backend Requirements

### Current state:
- `is_blocked` does **NOT** exist on Channel model
- `enabled` (boolean) and `status` (enum: active/pending/ignored) exist

### Recommendation for Phase 2:

**Use existing `enabled` field** rather than adding `is_blocked`. Rationale:
- No migration needed
- `enabled=false` already hides notifications from processing
- Semantically similar to "blocked"

**API endpoint already exists:**
```
PATCH /api/channels/{id}
Body: { "enabled": false }
```

### If you want separate `is_blocked` field (optional):

Backend changes needed:
1. Add `is_blocked = Column(Boolean, default=False)` to `models/channel.py`
2. Add to `ChannelUpdate` Pydantic schema
3. Migration: `alembic revision --autogenerate -m "add is_blocked to channels"`
4. Frontend: Add to `Channel` interface in `lib/api.ts`

**My recommendation:** Use `enabled` for Phase 2, consider `is_blocked` for Phase 3+ if semantic distinction matters.

---

## Summary

| Section | Status | Key Action |
|---------|--------|------------|
| Task Accuracy | Good with caveats | Clarify peek behavior, resolve `is_blocked` question |
| Granularity | Minor adjustments | Split 1.3 and 2.4 |
| Missing Context | Provided above | Share file list with implementer |
| Acceptance Criteria | Defined above | Use as verification checklist |
| Risks | Identified | Watch Dashboard integration and swipe conflicts |
| API Requirements | Clarified | Use `enabled` field, no migration needed |

**Ready to proceed** after resolving the `is_blocked` vs `enabled` decision.

---

## Answers to Open Questions (from my perspective as designer)

1. **Should blocked channels show notification cards?**
   No - Stage 0 should show channel rows only, not individual notifications. The point is to manage channels at the channel level.

2. **When a channel is blocked, do notifications disappear immediately?**
   Yes - the filter should be applied client-side immediately. The API call persists it, but the UI should react optimistically.

3. **Should Stage 0 show ALL channels or only those with notifications?**
   Show ALL channels. Users may want to preemptively block a channel before notifications arrive. Show notification count as "0" for empty channels.
