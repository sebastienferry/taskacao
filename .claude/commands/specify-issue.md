---
name: specify-issue
description: Write the executable specification of a ticket in the project's Spec-Driven Design framework, before any code.
---
# Specify Issue (OpenSpec SDD)

Stage: clarified -> specified.

## Goal
Produce a specification another engineer could implement without asking you
anything. Behaviour and acceptance criteria first, implementation choices second,
and the two kept in separate files.

## Read first
- The clarification outcome on the ticket: the decisions are already made, apply them.
- If {sdd_framework} or --framework=<name> is provided, use it. Otherwise, auto-detect:
  - If `openspec/` exists -> use OpenSpec SDD.
  - If `.specify/` or `specs/` exists -> use Spec Kit SDD.
- Ensure the project SDD directory is initialized before writing specifications.

## Steps
1. Create or switch to the work branch, named <KEY>-<title-slug>. Never write on the default branch.
2. Select the SDD framework from {sdd_framework} argument, flag, or project detection:

   **If using OpenSpec SDD:**
   - Create change directory `openspec/changes/<KEY>-<title-slug>/`
   - Write `proposal.md` (problem, value, in/out scope)
   - Write `design.md` (technical decisions, rejected alternatives)
   - Write `tasks.md` (ordered implementation checklist)
   - Write `specs/<capability>/spec.md` (requirements with Given/When/Then)
   - Validate with `openspec validate <change-id> --strict`

   **If using Spec Kit SDD:**
   - Write `specs/<KEY>-<title-slug>/spec.md` (prioritised user stories, functional requirements, Given/When/Then)
   - Write `plan.md` (stack, architecture, data contracts, target files)
   - Write `tasks.md` (ordered implementation checklist with test plan)
   - Use `/speckit.specify`, `/speckit.plan`, `/speckit.tasks` if available.

## Do not
- Do not decide what the clarification left open. Mark it as open and say so.
- Do not describe implementation inside the behaviour file.
- Do not start implementing, even the easy part.

## Report
- The files written, with their paths.
- The work branch.
- Requirements that are still open, and what they block.

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `clarified` to `specified`.
- **TaskFlow Local Handler (Recommended)**:
  Call TaskFlow's local transition handler to update SQLite state, record branch, and queue tracker synchronization:
  ```bash
  curl -s -X POST "${TASKFLOW_API_URL:-http://127.0.0.1:8090}/api/tasks/${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID}/stage" \
    -H "Content-Type: application/json" \
    -d '{
      "stage": "specified",
      "branch": "<KEY>-<title-slug>",
      "note": "<Paste specification report and change ID here>"
    }'
  ```
  *(Or via CLI: `taskflow stage ${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID} specified --branch <KEY>-<title-slug>`)*
- **Tracker CLI Fallback** (Only if the local TaskFlow server is unreachable):
  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label "specified" --remove-label "clarified"` && `gh issue comment <NUMBER> --body "..."`
  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "specified" --remove-label "clarified"` && `linear issue comment add <ISSUE_KEY> --body "..."`
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).
