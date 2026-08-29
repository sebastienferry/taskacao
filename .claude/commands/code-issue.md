---
name: code-issue
description: Implement the ticket from its specification and prove it works with the project's own build, linters and tests.
---
# Implement Code

Stage: specified -> implemented.

## Goal
Ship the change described by the specification, in code that reads like the code
already there, with the project's checks green.

## Read first
- The specification and its task checklist. It is the contract, follow its order.
- The surrounding code: naming, error handling, comment density, test style. Match it.
- How this project builds and tests. Find the real commands, do not assume them.

## Steps
1. Switch to the ticket's work branch. Never implement on the default branch.
2. Work through the checklist in small steps, each one leaving the tree buildable.
3. Add the tests that cover the new behaviour and its edge cases, not just the
   happy path. A change with no test needs a stated reason.
4. Run build, static analysis and tests. Fix until green, and quote the real output.
5. Re-read your own diff before finishing, as a reviewer would.

## Stop and report instead of pushing through when
- A decision in the specification turns out to be wrong or impossible.
- A test that was already failing before your change blocks the suite.
- The change would require touching a subsystem the specification never mentioned.

## Report
- What changed, file by file, and why.
- The real output of build, linters and tests, remaining failures included.
- What you deliberately left out, and what it would take to finish it.

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `specified` to `implemented`.
- **TaskFlow Local Handler (Recommended)**:
  Call TaskFlow's local transition handler to update SQLite state, record test green checks, and queue tracker synchronization:
  ```bash
  curl -s -X POST "${TASKFLOW_API_URL:-http://127.0.0.1:8090}/api/tasks/${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID}/stage" \
    -H "Content-Type: application/json" \
    -d '{
      "stage": "implemented",
      "note": "<Paste implementation and test output report here>"
    }'
  ```
  *(Or via CLI: `taskflow stage ${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID} implemented`)*
- **Tracker CLI Fallback** (Only if the local TaskFlow server is unreachable):
  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label "implemented" --remove-label "specified"` && `gh issue comment <NUMBER> --body "..."`
  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "implemented" --remove-label "specified"` && `linear issue comment add <ISSUE_KEY> --body "..."`
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).
