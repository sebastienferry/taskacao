---
description: Analyse les ambiguïtés techniques et produit 3 à 5 questions de cadrage.
argument-hint: <TICKET-KEY> [contexte]
---
# Clarify Issue

Stage: new -> clarified.

## Goal
Turn a vague ticket into a decided one. You are looking for the decisions that
would be expensive to reverse later, not for a list of everything unknown.

## Read first
- The ticket: title, description, comments, parent epic if there is one.
- The code the change would touch. Name the files you actually read.
- Neighbouring features that already solve a similar problem in this codebase.

## Steps
1. Restate the request in two sentences, including what you believe is out of scope.
2. List the ambiguities you found, worst first. An ambiguity is worth listing only
   if two readings lead to different code.
3. Name the critical dependencies: other services, other teams, migrations, data
   you do not have.
4. Formulate 3 to 5 numbered questions with your recommended options:
   - **Autonomous execution** (default background / pipeline run): Adopt the recommended options as the settled scope, document the rationale in the report, and advance the ticket.
   - **Interactive TTY session** (when running in an interactive terminal): Ask the questions directly to the user and incorporate their answers.
5. Record the settled scope and advance the ticket locally via TaskFlow handler.

## Do not
- Do not write production code at this stage, and do not start the specification.
- Do not invent an answer to your own question and move on without stating your assumption.
- Do not pad the list to reach five questions.

## Report
- Restated request and scope.
- Ambiguities, worst first.
- Critical dependencies.
- Numbered questions with your recommended option.
- Settled scope and assumptions.

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `new` to `clarified`.
- **Step 1: Check and use Local Handler (Recommended if TaskFlow is running)**:
  Call TaskFlow's local transition handler to update local state, record branch/PR, and automatically queue two-way synchronization to GitHub/Linear:
  - **Via TaskFlow CLI**:
    ```bash
    taskflow stage <KEY> clarified ["<optional summary note>"]
    ```
  - **Via HTTP API** (port 8090 or 8080):
    ```bash
    curl -s -X POST http://localhost:8090/api/tasks/stage -H "Content-Type: application/json" -d '{"taskKey": "<KEY>", "stage": "clarified"}' || curl -s -X POST http://localhost:8080/api/tasks/stage -H "Content-Type: application/json" -d '{"taskKey": "<KEY>", "stage": "clarified"}'
    ```
- **Step 2: Fallback to Direct Tracker CLI (Only if local TaskFlow handler is unreachable)**:
  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label "clarified" --remove-label "new"`
  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "clarified" --remove-label "new"`
- **Comments**: Post the stage summary report as a comment on the ticket via `taskflow stage <KEY> clarified "<REPORT_NOTE>"` or `gh issue comment <NUMBER> --body "..."` / `linear issue comment add <ISSUE_KEY> --body "..."`.
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).

## Ticket
$ARGUMENTS
