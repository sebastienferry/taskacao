---
description: Reformate la description d'une tâche en User Story structurée GFM, avec inclusion facultative des commentaires.
argument-hint: <TICKET-KEY> [contexte]
---
# Rewrite Story

Stage:  -> .

## Goal
Reformat a task's title, description, and optional comments into a clean GitHub-Flavored Markdown specification (User Story: As a..., I want..., So that... + Context + Acceptance Criteria + Notes).

## Read first
- The task: title, description, and task comments (if requested or passed as context).
- Standard GitHub-Flavored Markdown (GFM) formatting guidelines.

## Steps
1. Inspect the task title, raw description, and comments (if provided).
2. Extract the core intent, user value, technical context, and acceptance criteria.
3. Generate a structured GFM document containing:
   - **User Story**: As a <role>, I want <feature>, So that <benefit>.
   - **Context**: Problem background and technical overview.
   - **Acceptance Criteria**: Checkbox list (- [ ]) of verifiable functional & non-functional requirements.
   - **Notes**: Extra technical details or risks mentioned in comments.
4. Output the reformatted markdown directly for preview and user confirmation.

## Do not
- Do not mutate task title, status, priority, assignee, branch, or pull request.
- Do not delete or overwrite task comments.
- Do not invent artificial requirements not implied by the description or comments.

## Report
- The reformatted GFM description preview.
- List of comment points integrated into acceptance criteria (if any).

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `` to ``.
- **Step 1: Check and use Local Handler (Recommended if TaskFlow is running)**:
  Call TaskFlow's local transition handler to update local state, record branch/PR, and automatically queue two-way synchronization to GitHub/Linear:
  - **Via TaskFlow CLI**:
    ```bash
    taskflow stage <KEY>  ["<optional summary note>"]
    ```
  - **Via HTTP API** (port 8090 or 8080):
    ```bash
    curl -s -X POST http://localhost:8090/api/tasks/stage -H "Content-Type: application/json" -d '{"taskKey": "<KEY>", "stage": ""}' || curl -s -X POST http://localhost:8080/api/tasks/stage -H "Content-Type: application/json" -d '{"taskKey": "<KEY>", "stage": ""}'
    ```
- **Step 2: Fallback to Direct Tracker CLI (Only if local TaskFlow handler is unreachable)**:
  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label "" --remove-label ""`
  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "" --remove-label ""`
- **Comments**: Post the stage summary report as a comment on the ticket via `taskflow stage <KEY>  "<REPORT_NOTE>"` or `gh issue comment <NUMBER> --body "..."` / `linear issue comment add <ISSUE_KEY> --body "..."`.
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).

## Ticket
$ARGUMENTS
