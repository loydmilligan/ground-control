# Test Results

**Task**: 5.1 Create FocusedFeed Component
**Iteration**: 1
**Time**: 2026-03-07T03:28:51-08:00
**Status**: PASSED

## Summary

✓ All tests passed

Passed: 24, Failed: 0


## Full Output

```
?   	github.com/mmariani/ground-control/cmd/gc	[no test files]
?   	github.com/mmariani/ground-control/internal/artifact	[no test files]
?   	github.com/mmariani/ground-control/internal/claude	[no test files]
=== RUN   TestFormatTimeAgo
=== RUN   TestFormatTimeAgo/just_now
=== RUN   TestFormatTimeAgo/1_minute_ago
=== RUN   TestFormatTimeAgo/multiple_minutes
=== RUN   TestFormatTimeAgo/1_hour_ago
=== RUN   TestFormatTimeAgo/multiple_hours
=== RUN   TestFormatTimeAgo/1_day_ago
=== RUN   TestFormatTimeAgo/multiple_days
--- PASS: TestFormatTimeAgo (0.00s)
    --- PASS: TestFormatTimeAgo/just_now (0.00s)
    --- PASS: TestFormatTimeAgo/1_minute_ago (0.00s)
    --- PASS: TestFormatTimeAgo/multiple_minutes (0.00s)
    --- PASS: TestFormatTimeAgo/1_hour_ago (0.00s)
    --- PASS: TestFormatTimeAgo/multiple_hours (0.00s)
    --- PASS: TestFormatTimeAgo/1_day_ago (0.00s)
    --- PASS: TestFormatTimeAgo/multiple_days (0.00s)
=== RUN   TestFormatTimeAgoOldDate
--- PASS: TestFormatTimeAgoOldDate (0.00s)
PASS
ok  	github.com/mmariani/ground-control/internal/cmd	0.018s
?   	github.com/mmariani/ground-control/internal/context	[no test files]
?   	github.com/mmariani/ground-control/internal/data	[no test files]
=== RUN   TestAnalyzeDependencies
=== RUN   TestAnalyzeDependencies/empty_tasks
=== RUN   TestAnalyzeDependencies/single_task
=== RUN   TestAnalyzeDependencies/two_independent_tasks
=== RUN   TestAnalyzeDependencies/simple_dependency_chain
=== RUN   TestAnalyzeDependencies/parallel_with_follow-up
=== RUN   TestAnalyzeDependencies/external_dependencies_don't_block
--- PASS: TestAnalyzeDependencies (0.00s)
    --- PASS: TestAnalyzeDependencies/empty_tasks (0.00s)
    --- PASS: TestAnalyzeDependencies/single_task (0.00s)
    --- PASS: TestAnalyzeDependencies/two_independent_tasks (0.00s)
    --- PASS: TestAnalyzeDependencies/simple_dependency_chain (0.00s)
    --- PASS: TestAnalyzeDependencies/parallel_with_follow-up (0.00s)
    --- PASS: TestAnalyzeDependencies/external_dependencies_don't_block (0.00s)
=== RUN   TestCanParallelize
=== RUN   TestCanParallelize/empty_tasks
=== RUN   TestCanParallelize/single_task
=== RUN   TestCanParallelize/two_independent_tasks
=== RUN   TestCanParallelize/simple_dependency_chain
=== RUN   TestCanParallelize/parallel_with_follow-up
--- PASS: TestCanParallelize (0.00s)
    --- PASS: TestCanParallelize/empty_tasks (0.00s)
    --- PASS: TestCanParallelize/single_task (0.00s)
    --- PASS: TestCanParallelize/two_independent_tasks (0.00s)
    --- PASS: TestCanParallelize/simple_dependency_chain (0.00s)
    --- PASS: TestCanParallelize/parallel_with_follow-up (0.00s)
PASS
ok  	github.com/mmariani/ground-control/internal/pipeline	0.036s
?   	github.com/mmariani/ground-control/internal/session	[no test files]
?   	github.com/mmariani/ground-control/internal/tui	[no test files]
?   	github.com/mmariani/ground-control/internal/types	[no test files]
?   	github.com/mmariani/ground-control/internal/verify	[no test files]

```