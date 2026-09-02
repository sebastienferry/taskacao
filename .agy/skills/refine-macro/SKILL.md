---
name: refine-macro
description: Interactively clarify macro framing text with the user and break it down into structured todos and TaskFlow tickets.
---
# Refine Macro (OpenSpec SDD)

Stage: macro -> macro. Interactive: the user answers in the terminal.

## Goal
Transform high-level macro framing text into an actionable, structured todo list and concrete TaskFlow tickets, interactively clarifying ambiguities with the user when framing text is vague.

## Read first
- The macro title and framing description.
- The active project SDD framework (SpecKit or OpenSpec).
- Existing macro todos and child tasks to avoid duplicating completed work.

## Steps
1. Inspect the macro title and high-level framing description.
2. **Evaluate framing completeness**:
   - If the framing description is empty, under 2 sentences, or lacks clear technical boundaries/acceptance criteria, formulate 3 to 5 numbered clarification questions and ask the user directly in this interactive terminal session before generating tasks.
3. **Decompose & Break Down**:
   - Once answered or if framing text is detailed, group action items according to the selected SDD framework:
     - **SpecKit SDD**: Group into User Stories ([US-x]) and Feature Modules ([FEAT-x]).
     - **OpenSpec SDD**: Group into Capabilities ([CAP-x]) and Change Proposals ([CHANGE-x]).
4. Output the generated checklist of actionable todos AND proposed TaskFlow tickets (Title, IssueType: Story/Task/Bug, Description) for bulk ticket creation.

## Do not
- Do not generate tasks blindly when framing text is vague without asking clarification questions.
- Do not overwrite existing todos or tasks without user confirmation in the UI.
- Do not mutate external tracker issues directly without user trigger.

## Report
- Clarification Q&A summary (if framing was vague).
- Structured list of proposed MacroTodo items.
- Proposed TaskFlow tickets breakdown (Title, IssueType, Description).

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `macro` to `macro`.
- **Step 1: Check and use Local Handler (Recommended if TaskFlow is running)**:
  Call TaskFlow's local transition handler to update local state, record branch/PR, and automatically queue two-way synchronization to GitHub/Linear:
  - **Via TaskFlow CLI**:
    ```bash
    taskflow stage <KEY> macro ["<optional summary note>"]
    ```
  - **Via HTTP API** (port 8090 or 8080):
    ```bash
    curl -s -X POST http://localhost:8090/api/tasks/stage -H "Content-Type: application/json" -d '{"taskKey": "<KEY>", "stage": "macro"}' || curl -s -X POST http://localhost:8080/api/tasks/stage -H "Content-Type: application/json" -d '{"taskKey": "<KEY>", "stage": "macro"}'
    ```
- **Step 2: Fallback to Direct Tracker CLI (Only if local TaskFlow handler is unreachable)**:
  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label "macro" --remove-label "macro"`
  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "macro" --remove-label "macro"`
- **Comments**: Post the stage summary report as a comment on the ticket via `taskflow stage <KEY> macro "<REPORT_NOTE>"` or `gh issue comment <NUMBER> --body "..."` / `linear issue comment add <ISSUE_KEY> --body "..."`.
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).
