---
name: clarify-issue
description: Analyse a ticket against the code, surface what is genuinely undecided, and ask the few questions that unblock specification.
---
# Clarify Issue

Stage: new -> clarified. Interactive: the user answers in the terminal.

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
4. Ask 3 to 5 numbered questions. Prefer closed questions. For each one, state the
   option you recommend and why, so silence still leaves a usable default.
5. Wait for the answers. Then write the settled scope: what was decided, what was
   explicitly dropped, what stays open on purpose.

## Do not
- Do not write production code at this stage, and do not start the specification.
- Do not invent an answer to your own question and move on.
- Do not pad the list to reach five questions.

## Report
- Restated request and scope.
- Ambiguities, worst first.
- Critical dependencies.
- Numbered questions with your recommended option.
- After the answers: the settled scope.

## Ticket Transition & Status Update
The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:
- **Stage Transition**: Advance ticket from `new` to `clarified`.
- **GitHub CLI**: `gh issue edit <NUMBER> --add-label "clarified" --remove-label "new"`
- **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label "clarified" --remove-label "new"`
- **Comments**: Post the stage summary report as a comment on the ticket via `gh issue comment <NUMBER> --body "..."` or `linear issue comment add <ISSUE_KEY> --body "..."`.
- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).
