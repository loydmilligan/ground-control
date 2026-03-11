# Flight Deck Test Workflows

End-to-end test scenarios for validating the FD MVP.

## Prerequisites

```bash
# Ensure GC is built
cd /home/mmariani/Projects/ground-control
go build -o gc ./cmd/gc

# Ensure test project is adopted
./gc adopt /home/mmariani/Projects/notifai --force
```

---

## Test 1: New Feature Workflow

**User Story**: Matt wants to add a new feature to notifai.

### Steps

1. **Open Claude session in notifai**
   ```bash
   cd /home/mmariani/Projects/notifai
   claude
   ```

2. **Add feature to roadmap**
   ```
   /roadmap-item "Topic grouping for notifications using AI" --start-now
   ```

3. **Verify roadmap.json updated**
   ```bash
   cat .gc/roadmap.json
   ```
   - Should contain new feature with `feat_*` ID
   - Status should be `in_progress`
   - Should have `created_at` timestamp

4. **Verify state.json updated**
   ```bash
   cat .gc/state.json
   ```
   - Should have `current_focus` set to the feature ID

5. **Update progress**
   ```
   /progress 30 --note "Completed initial research"
   ```

6. **Verify progress saved**
   ```bash
   cat .gc/roadmap.json | jq '.features[0].completion_pct'
   ```
   - Should show `30`

7. **Request commit**
   ```
   /commit "Add topic grouping classifier"
   ```

8. **Verify request queued**
   ```bash
   cat .gc/requests.jsonl
   ```
   - Should contain commit request with status `pending`

9. **Complete the work**
   ```
   /complete --summary "Implemented AI topic grouping with 95% accuracy"
   ```

10. **Verify completion**
    ```bash
    cat .gc/roadmap.json | jq '.features[0].status'
    cat .gc/state.json | jq '.session.current_focus'
    ```
    - Status should be `completed`
    - Current focus should be `null`

11. **Sync from FD**
    ```bash
    cd /home/mmariani/Projects/ground-control
    ./gc sync
    cat ~/.gc/aggregated.json | jq '.projects.notifai'
    ```
    - Should show updated roadmap_pct
    - Should show pending_requests if commit not processed

### Expected Result
- Feature tracked from idea through completion
- All state files updated correctly
- FD can see progress via sync

---

## Test 2: Bug Report Workflow

**User Story**: Matt finds a bug while working and wants to report it.

### Steps

1. **In notifai Claude session, report bug**
   ```
   /issue "Login form doesn't validate email format" --priority high
   ```

2. **Verify issue created**
   ```bash
   cat .gc/issues.json
   ```
   - Should contain issue with `issue_*` ID
   - Priority should be `high`
   - Status should be `open`

3. **Start working on the bug**
   ```
   /start-work issue_xxx
   ```

4. **Verify state updated**
   ```bash
   cat .gc/state.json | jq '.session.current_focus'
   ```
   - Should show the issue ID

5. **Fix and complete**
   ```
   /commit "Fix: validate email format in login form"
   /complete --summary "Added email regex validation"
   ```

6. **Verify issue closed**
   ```bash
   cat .gc/issues.json | jq '.issues[0].status'
   ```
   - Should be `completed`

### Expected Result
- Bug tracked from report through fix
- Commit request queued for FD

---

## Test 3: Learning Capture Workflow

**User Story**: Matt encounters process friction and wants to log it.

### Steps

1. **Log friction**
   ```
   /learn friction "The roadmap-item command doesn't show confirmation of what was added"
   ```

2. **Verify learning captured**
   ```bash
   cat .gc/learning.jsonl
   ```
   - Should contain entry with type `friction`
   - Should have timestamp

3. **Log an idea**
   ```
   /learn idea "Add --verbose flag to show full JSON output"
   ```

4. **Log a success**
   ```
   /learn success "The start-work command loaded all context nicely"
   ```

5. **Sync and check FD**
   ```bash
   cd /home/mmariani/Projects/ground-control
   ./gc sync
   ```

### Expected Result
- All learning types captured in learning.jsonl
- Can be reviewed by FD Claude later

---

## Test 4: FD Dispatch Workflow

**User Story**: FD Claude dispatches work to a project.

### Steps

1. **In FD Claude session**
   ```bash
   cd /home/mmariani/Projects/ground-control/flight-deck
   claude
   ```

2. **Dispatch work**
   ```
   /dispatch --project notifai --type coding "Implement notification batching"
   ```

3. **Verify inbox item created**
   ```bash
   ls /home/mmariani/Projects/notifai/.gc/inbox/
   cat /home/mmariani/Projects/notifai/.gc/inbox/work_*.json
   ```
   - Should contain work item with description

4. **In notifai session, check inbox**
   ```bash
   ls .gc/inbox/
   ```

5. **Pick up the work**
   ```
   # Review inbox item, then convert to roadmap item
   /roadmap-item "Notification batching" --start-now
   ```

### Expected Result
- FD can dispatch work to projects
- Projects can see and pick up dispatched work

---

## Test 5: Sync and Hangar View

**User Story**: Matt wants to see all project statuses in Flight Deck.

### Steps

1. **Add some data to notifai**
   - Create a feature, update progress to 50%
   - Create 2 issues, mark 1 as high priority

2. **Run sync**
   ```bash
   cd /home/mmariani/Projects/ground-control
   ./gc sync
   ```

3. **Check aggregated state**
   ```bash
   cat ~/.gc/aggregated.json | jq '.projects.notifai'
   ```
   - Should show `issues_count`, `open_bugs`
   - Should show `roadmap_pct`
   - Should show `pending_requests`

4. **Launch Flight Deck**
   ```bash
   ./gc fd
   ```

5. **Verify Hangar display**
   - Should show notifai with phase, %, bugs, flags columns
   - Numbers should match what's in aggregated.json

### Expected Result
- Sync aggregates all project data
- Hangar displays project details correctly

---

## Test 6: Full Cycle - Feature from Idea to Completion

**User Story**: Complete lifecycle of a feature.

### Steps

1. **Capture idea** (in notifai)
   ```
   /roadmap-item "Smart notification grouping by topic"
   ```

2. **Check FD** (run sync, check aggregated.json)

3. **Start work**
   ```
   /start-work feat_xxx
   ```

4. **Progress updates**
   ```
   /progress 25 --note "Designed algorithm"
   /progress 50 --note "Implemented core logic"
   /progress 75 --note "Added tests"
   ```

5. **Encounter bug, report it**
   ```
   /issue "Edge case: empty notification list crashes" --blocks feat_xxx
   ```

6. **Fix bug**
   ```
   /start-work issue_xxx
   /commit "Fix: handle empty notification list"
   /complete
   ```

7. **Continue feature**
   ```
   /start-work feat_xxx
   /progress 90 --note "Fixed blocking bug"
   ```

8. **Final commit and complete**
   ```
   /commit "Add smart notification grouping"
   /complete --summary "Feature complete with tests"
   ```

9. **Sync and verify in FD**
   ```bash
   ./gc sync
   cat ~/.gc/aggregated.json
   ```

### Expected Result
- Full feature lifecycle tracked
- Bug as side issue during development
- All states visible in FD

---

## Test 7: Re-adoption with Backup

**User Story**: Re-adopt a project and verify backup works.

### Steps

1. **Check existing .gc files**
   ```bash
   ls -la /home/mmariani/Projects/notifai/.gc/
   ```

2. **Re-adopt**
   ```bash
   ./gc adopt /home/mmariani/Projects/notifai --force
   ```

3. **Verify backup created**
   ```bash
   ls /home/mmariani/Projects/notifai/.gc/backup_*/
   ```
   - Should contain previous files

4. **Verify new files generated**
   ```bash
   cat /home/mmariani/Projects/notifai/.gc/CLAUDE.md | grep "Flight Deck"
   ```
   - Should have FD integration section

### Expected Result
- Backup preserves previous state
- New adoption includes latest FD features

---

## Cleanup

After testing, you can reset notifai's .gc state:

```bash
# Remove test data (keep backup)
cd /home/mmariani/Projects/notifai/.gc
rm -f roadmap.json issues.json requests.jsonl learning.jsonl
rm -rf inbox/*

# Recreate empty files
echo '{"features":[],"milestones":[]}' > roadmap.json
echo '{"issues":[]}' > issues.json
touch requests.jsonl learning.jsonl
```

Or restore from backup:
```bash
cp .gc/backup_*/roadmap.json .gc/
cp .gc/backup_*/issues.json .gc/
# etc.
```

---

## Test Results Checklist

| Test | Status | Notes |
|------|--------|-------|
| Test 1: New Feature | [ ] | |
| Test 2: Bug Report | [ ] | |
| Test 3: Learning Capture | [ ] | |
| Test 4: FD Dispatch | [ ] | |
| Test 5: Sync & Hangar | [ ] | |
| Test 6: Full Cycle | [ ] | |
| Test 7: Re-adoption | [ ] | |

---

## Known Limitations (MVP)

1. **Slash commands are prompt-based** - Claude reads the prompt file and follows it, not automated parsing
2. **No auto-sync** - Must run `gc sync` manually before checking FD
3. **No real-time updates** - State only updates when Claude acts on it
4. **Inbox pickup is manual** - Project Claude doesn't auto-check inbox on start
