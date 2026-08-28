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
- **GitHub CLI**: `gh issue edit <NUMBER> --add-label "implemented" --remove-label "specified"`
- **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "implemented" --remove-label "specified"`
- **Comments**: Post the stage summary report as a comment on the ticket via `gh issue comment <NUMBER> --body "..."` or `linear issue comment add <ISSUE_KEY> --body "..."`.
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).
