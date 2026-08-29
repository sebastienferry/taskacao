---
description: Write the executable specification of a ticket in the project's Spec-Driven Design framework, before any code.
argument-hint: <TICKET-KEY> [contexte]
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
- **GitHub CLI**: `gh issue edit <NUMBER> --add-label "specified" --remove-label "clarified"`
- **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "specified" --remove-label "clarified"`
- **Comments**: Post the stage summary report as a comment on the ticket via `gh issue comment <NUMBER> --body "..."` or `linear issue comment add <ISSUE_KEY> --body "..."`.
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).

## Ticket
$ARGUMENTS
