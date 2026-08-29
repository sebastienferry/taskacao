---
name: create-pr
description: Review the branch like a peer would, fix what the review finds, then open the merge request and leave the merge to the user.
---
# Review and Pull Request

Stage: implemented -> reviewed.

## Goal
Hand a reviewer a branch that is already worth reading: the obvious problems
found and fixed, the risky parts pointed out, the test plan written down.

## Read first
- The full diff of the branch against the default branch. All of it, not the summary.
- The specification, to check that what was asked is what was built.

## Steps
1. Review the diff for correctness, side effects, security, and edge cases with no test.
2. Fix what the review finds, now. A known defect belongs in the code, not in the
   description of the merge request.
3. Re-run build, static analysis and tests on the final state.
4. Commit with a conventional message: type, scope, and why the change exists.
5. Push the branch and open the merge request: summary, test plan, and the specific
   places where you want a reviewer's eyes.
6. If the repository has no remote, say so and stop rather than merging locally.

## Do not
- Do not merge, do not approve, do not close the ticket. That is the user's call.
- Do not open a merge request on a red build. Report the failure instead.

## Report
- What the review found, and which findings you fixed.
- The merge request URL, or why there is none.
- The test plan a reviewer can replay, as a checklist.

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `implemented` to `reviewed`.
- **TaskFlow Local Handler (Recommended)**:
  Call TaskFlow's local transition handler to record the PR URL, review report, and queue tracker synchronization:
  ```bash
  curl -s -X POST "${TASKFLOW_API_URL:-http://127.0.0.1:8090}/api/tasks/${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID}/stage" \
    -H "Content-Type: application/json" \
    -d '{
      "stage": "reviewed",
      "prUrl": "<PR_OR_MR_URL>",
      "note": "<Paste review findings and PR link report here>"
    }'
  ```
  *(Or via CLI: `taskflow stage ${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID} reviewed --pr-url "<PR_OR_MR_URL>"`)*
- **Tracker CLI Fallback** (Only if the local TaskFlow server is unreachable):
  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label "reviewed" --remove-label "implemented"` && `gh issue comment <NUMBER> --body "..."`
  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "reviewed" --remove-label "implemented"` && `linear issue comment add <ISSUE_KEY> --body "..."`
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).
