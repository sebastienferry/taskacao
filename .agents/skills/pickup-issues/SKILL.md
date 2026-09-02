---
name: pickup-issues
description: Batch process a list of selected board tickets sequentially in autonomy inside a single dedicated worktree, producing one combined Pull Request covered by tests and lints.
---

# Batch Pickup Issues (Single Worktree & Combined PR)

## Goal
Autonomously process a batch of tickets selected from the board sequentially in the exact order provided. All tickets are processed inside a **single dedicated batch worktree/branch**, superseding individual per-ticket branching/PR instructions, and producing **one single combined Pull Request** at the end covered by full test and lint suites.

## Trigger & Arguments
- Command: `/pickup-issues <ISSUE_1> <ISSUE_2> <ISSUE_3> ...`
- Input: A list of space-separated or comma-separated issue keys/IDs/URLs (e.g., `FRE-101 FRE-102 FRE-103` or `#230 #231 #232`).

## Workflow

### 1. Batch Initialization & Single Worktree Setup
- Parse the input into an ordered queue of tickets: `[TICKET_1, TICKET_2, ..., TICKET_N]`.
- Determine the batch branch name, e.g. `batch-issues-<FIRST_ID>-<LAST_ID>` or `<WORKTREE_BRANCH>`.
- Create or switch to the **single dedicated batch worktree/branch**.
- Verify active tracker configuration (via `.agents/issue-tracker.json` or project tracker defaults).
- Initialize execution tracking log for all tickets in the batch.

### 2. Sequential In-Place Ticket Processing Loop
For each `TICKET_KEY` in the queue (from `1` to `N`), execute in order inside the **same worktree**:

#### Step A: Requirement & Clarification
- Announce: `[Batch Progress: Step X of N] Processing TICKET_KEY...`
- Inspect ticket requirements, comments, and constraints.
- Resolve any unclarified points directly or apply project guidelines.

#### Step B: Specification Generation
- Generate/update the ticket specification under `specs/` (e.g., `proposal.md`, `design.md`, `tasks.md`).

#### Step C: Code & Unit Test Implementation
- Implement code changes and dedicated unit tests corresponding to the ticket spec.
- Commit ticket changes with conventional commit format: `feat(<scope>): <description> (#<TICKET_KEY>)`.
- Update ticket stage/label to `implemented` on the issue tracker.

### 3. Combined Batch Quality & Verification Phase
After all tickets in the queue (`1..N`) have been implemented:
1. **Go Unit & Integration Tests**: Run `go test ./...` — must pass 100% green.
2. **Server Build Validation**: Run `go build -o bin/server ./cmd/server` — must pass.
3. **Battle Test Suite**: Run `go run ./scripts/battletest.go` against a running server — must pass.
4. **Frontend Web App Build**: Run `npm run build` in `web` (or Vite build) — must pass without errors.

### 4. Single Combined Pull Request Creation
- Push the batch branch to remote: `git push origin <BATCH_BRANCH>`.
- Open **ONE single combined Pull Request** (`gh pr create` / Linear CLI) referencing all ticket numbers (`#TICKET_1`, `#TICKET_2`, ...).
- Provide a clear, structured PR description summarizing:
  - List of processed tickets and their implemented features/fixes.
  - Verification evidence (test pass, build pass, security/BOLA checks pass).
- Update all ticket stages/labels to `reviewed` or ready for merge.

## Safety Rules & Constraints
- **Single Worktree**: Do NOT create separate branches or PRs per ticket. All tickets in the batch live on the single batch branch.
- **Sequential Order**: Process tickets strictly one at a time in the exact order specified.
- **No Automatic Merging**: NEVER merge PRs into `main`. Merging is strictly reserved for human review.
- **Test Integrity**: Every single change across all tickets in the batch must be covered by passing tests and clean builds before PR creation.
