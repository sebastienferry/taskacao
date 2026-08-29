---
name: handoff-issue
description: Close the ticket properly: confirm the merge, write the handover and the acceptance checklist, then clean the local workspace.
---
# Handoff and Close

Stage: reviewed -> finished.

## Goal
Leave two things behind: a handover a colleague can act on without asking you,
and a local workspace with nothing stale in it.

## Read first
- The state of the branch against the default branch.
- What the implementation and review steps reported, so the handover matches reality.

## Steps
1. Confirm the ticket's branch is actually merged into the default branch. If it is
   not, stop, say so, and clean nothing.
2. Write the handover: what shipped, what changed for the user, what is still open.
3. Write the acceptance checklist as checkboxes, each item something a human can
   verify in the running product.
4. Update the repository documentation when the change makes it wrong, README and
   changelog included.
5. Turn any remaining follow-up into a separate ticket to create, rather than a
   paragraph nobody will read.
6. Clean up locally: remove the ticket's worktree, delete the local branch once the
   merge is confirmed.

## Do not
- Do not delete anything remote: no remote branch, no tag, no release.
- Do not clean up while the merge is unconfirmed.

## Report
- The handover.
- The acceptance checklist, as checkboxes.
- What was cleaned locally, and what could not be, with the reason.
- Follow-up tickets worth creating.

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `reviewed` to `finished`.
- **GitHub CLI**: `gh issue edit <NUMBER> --add-label "finished" --remove-label "reviewed"` then `gh issue close <NUMBER>`
- **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "finished" --remove-label "reviewed" --state "Done"`
- **Comments**: Post the stage summary report as a comment on the ticket via `gh issue comment <NUMBER> --body "..."` or `linear issue comment add <ISSUE_KEY> --body "..."`.
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).
