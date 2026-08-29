---
description: Exécute en autonomie complète toutes les étapes d'un ticket jusqu'à la création de la Pull Request.
argument-hint: <TICKET-KEY> [contexte]
---
# Pickup Issue (Auto-Pilot to PR)

Stage: new -> reviewed.

## Goal
Autonomously take a ticket from its current stage through clarification, specification,
implementation, and testing, all the way to opening a clean Pull Request, updating each stage via TaskFlow.

## Read first
- The ticket: key, title, description, parent macro, and tracker comments.
- The project's code and existing patterns.
- The project SDD framework (OpenSpec or Spec Kit).

## Steps
1. **Pick & Inspect**:
   - Identify the ticket key (<KEY>) and target branch name (<KEY>-<title-slug>).
   - Check the current ticket stage and start from where it currently is.
   - Switch or create the work branch `<KEY>-<title-slug>`. Never implement on the default branch.

2. **Step 1: Clarification (if not already clarified)**:
   - Restate the requirements and resolve ambiguities with sensible technical choices.
   - Transition ticket locally: `taskflow stage <KEY> clarified`

3. **Step 2: Specification (if not already specified)**:
   - Write the formal technical specification in `openspec/changes/<KEY>-<title-slug>/` or `specs/<KEY>-<title-slug>/`.
   - Validate the specification structure and checklist.
   - Transition ticket locally: `taskflow stage <KEY> specified`

4. **Step 3: Implementation & Validation**:
   - Implement the changes incrementally on the work branch following the spec checklist.
   - Add automated tests covering the new behavior and edge cases.
   - Run the project's build, linters, and test suite until all checks pass (100% green).
   - Transition ticket locally: `taskflow stage <KEY> implemented --branch "<KEY>-<title-slug>"`

5. **Step 4: Review, Push & Pull Request**:
   - Review the complete diff against the default branch to ensure cleanliness.
   - Commit all changes with a clean conventional commit message.
   - Push the branch to the remote repository: `git push -u origin <KEY>-<title-slug>`
   - Open the Pull Request / Merge Request via GitHub CLI (`gh pr create`) or GitLab/Linear tooling.
   - Transition ticket locally: `taskflow stage <KEY> reviewed --pr-url "<PR_URL>"`

6. **Step 5: Stop before merge**:
   - Report the PR URL, test results, and summary of changes.
   - Do NOT merge into the default branch (merging is strictly reserved for the human user).

## Do not
- Do not merge into the default branch (merging is reserved for the human user).
- Do not push or open a PR if the test suite is failing.
- Do not skip the local handler stage transitions.

## Report
- The created Pull Request URL.
- The work branch and files modified.
- The test results demonstrating that build, lint, and tests pass.
- Summary of settled scope and key architectural decisions.

## Ticket Transition & Autonomous Pipeline Contract
The agent executing the pickup skill is responsible for advancing the ticket through each stage autonomously up to PR creation:
- **Step 1 (Clarify)**: Advance ticket to `clarified` via `taskflow stage <KEY> clarified`
- **Step 2 (Specify)**: Advance ticket to `specified` via `taskflow stage <KEY> specified`
- **Step 3 (Implement)**: Advance ticket to `implemented` via `taskflow stage <KEY> implemented --branch "<KEY>-<title-slug>"`
- **Step 4 (Review & PR)**: Advance ticket to `reviewed` via `taskflow stage <KEY> reviewed --pr-url "<PR_URL>"`
- **HTTP API Alternative** (if CLI not in PATH): `curl -s -X POST http://localhost:8090/api/tasks/stage -H "Content-Type: application/json" -d '{"taskKey":"<KEY>","stage":"<STAGE>"}'`
- **Fallback to Tracker CLI** (only if TaskFlow is unreachable): `gh issue edit <NUMBER> --add-label "<STAGE>"` / `linear issue update <KEY> --add-label "<STAGE>"`
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).

## Ticket
$ARGUMENTS
