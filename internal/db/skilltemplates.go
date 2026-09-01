package db

import (
	"fmt"
	"path/filepath"
	"strings"

	"tasks/internal/models"
)

// The skills of the agentic workflow, one per step, in stage order.
//
// Before, five templates each lived their own life: the last stage had none and
// ran on a free prompt, and nothing told the agent who moves the ticket. Here a
// single table describes the steps and a single renderer produces the SKILL.md
// files, so the five read alike and carry the same contract.
//
// The files are written in English on purpose: the agents reason on them, and
// the repositories they land in are read by people who do not all speak French.
// The TaskFlow interface stays in French.
type StageSkill struct {
	ID          string // internal id, shared by the catalogue and the job queue
	Name        string
	DirName     string // skill directory, which is also the slash command
	Command     string
	FromStage   string
	ToStage     string
	Scope       string // "task" (default) or "macro"
	Interactive bool
	Description string // shown in the TaskFlow interface, in French
	Icon        string
	Color       string
	Steps       []string // the step list the interface displays

	// Body of the SKILL.md. The renderer adds the header, the stage line and the
	// contract with TaskFlow. title is the English heading of the file, kept
	// apart from Name, which the French interface displays.
	title           string
	frontmatterDesc string
	goal            string
	readFirst       string
	stepsBody       string
	guardTitle      string
	guard           string
	report          string
}

// StageSkills is the unified set: one skill per workflow step. The old
// pick-issue auto-pilot is gone, the autonomous button replaced it.
var StageSkills = []StageSkill{
	{
		ID:          "clarify",
		Name:        "Clarify Issue",
		DirName:     models.SkillDirNames["clarify"],
		Command:     "/clarify-issue",
		FromStage:   "new",
		ToStage:     "clarified",
		Interactive: false,
		Description: "Analyse les ambiguïtés techniques et produit 3 à 5 questions de cadrage.",
		Icon:        "HelpCircle",
		Color:       "amber",
		Steps: []string{
			"Lecture du ticket et du code concerné",
			"Détection des ambiguïtés et des dépendances",
			"Questions de cadrage et options recommandées",
			"Label 'clarified' et transition posés par TaskFlow",
		},
		title:           "Clarify Issue",
		frontmatterDesc: "Analyse a ticket against the code, surface what is genuinely undecided, and ask the few questions that unblock specification.",
		goal: `Turn a vague ticket into a decided one. You are looking for the decisions that
would be expensive to reverse later, not for a list of everything unknown.`,
		readFirst: `- The ticket: title, description, comments, parent epic if there is one.
- The code the change would touch. Name the files you actually read.
- Neighbouring features that already solve a similar problem in this codebase.`,
		stepsBody: `1. Restate the request in two sentences, including what you believe is out of scope.
2. List the ambiguities you found, worst first. An ambiguity is worth listing only
   if two readings lead to different code.
3. Name the critical dependencies: other services, other teams, migrations, data
   you do not have.
4. Formulate 3 to 5 numbered questions with your recommended options:
   - **Autonomous execution** (default background / pipeline run): Adopt the recommended options as the settled scope, document the rationale in the report, and advance the ticket.
   - **Interactive TTY session** (when running in an interactive terminal): Ask the questions directly to the user and incorporate their answers.
5. Record the settled scope and advance the ticket locally via TaskFlow handler.`,
		guardTitle: "Do not",
		guard: `- Do not write production code at this stage, and do not start the specification.
- Do not invent an answer to your own question and move on without stating your assumption.
- Do not pad the list to reach five questions.`,
		report: `- Restated request and scope.
- Ambiguities, worst first.
- Critical dependencies.
- Numbered questions with your recommended option.
- Settled scope and assumptions.`,
	},
	{
		ID:          "specify",
		Name:        "Specify Issue",
		DirName:     models.SkillDirNames["specify"],
		Command:     "/specify-issue",
		FromStage:   "clarified",
		ToStage:     "specified",
		Description: "Rédige la spécification technique selon le cadre SDD du projet.",
		Icon:        "FileCode",
		Color:       "blue",
		Steps: []string{
			"Lecture des principes et des specs existantes",
			"Création de la branche de travail",
			"Rédaction de la spécification et de la checklist",
			"Label 'specified' et transition posés par TaskFlow",
		},
		title:           "Specify Issue",
		frontmatterDesc: "Write the executable specification of a ticket in the project's Spec-Driven Design framework, before any code.",
		goal: `Produce a specification another engineer could implement without asking you
anything. Behaviour and acceptance criteria first, implementation choices second,
and the two kept in separate files.`,
		// readFirst et stepsBody dépendent du cadre du projet : voir specifyFrameworkBody.
		guardTitle: "Do not",
		guard: `- Do not decide what the clarification left open. Mark it as open and say so.
- Do not describe implementation inside the behaviour file.
- Do not start implementing, even the easy part.`,
		report: `- The files written, with their paths.
- The work branch.
- Requirements that are still open, and what they block.`,
	},
	{
		ID:          "implement",
		Name:        "Implement Code",
		DirName:     models.SkillDirNames["implement"],
		Command:     "/code-issue",
		FromStage:   "specified",
		ToStage:     "implemented",
		Description: "Exécute le plan d'implémentation et valide par les tests.",
		Icon:        "Flame",
		Color:       "indigo",
		Steps: []string{
			"Lecture de la spécification et des tâches",
			"Implémentation sur la branche du ticket",
			"Construction, analyse statique et tests au vert",
			"Label 'implemented' et transition posés par TaskFlow",
		},
		title:           "Implement Code",
		frontmatterDesc: "Implement the ticket from its specification and prove it works with the project's own build, linters and tests.",
		goal: `Ship the change described by the specification, in code that reads like the code
already there, with the project's checks green.`,
		readFirst: `- The specification and its task checklist. It is the contract, follow its order.
- The surrounding code: naming, error handling, comment density, test style. Match it.
- How this project builds and tests. Find the real commands, do not assume them.`,
		stepsBody: `1. Switch to the ticket's work branch. Never implement on the default branch.
2. Work through the checklist in small steps, each one leaving the tree buildable.
3. Add the tests that cover the new behaviour and its edge cases, not just the
   happy path. A change with no test needs a stated reason.
4. Run build, static analysis and tests. Fix until green, and quote the real output.
5. Re-read your own diff before finishing, as a reviewer would.`,
		guardTitle: "Stop and report instead of pushing through when",
		guard: `- A decision in the specification turns out to be wrong or impossible.
- A test that was already failing before your change blocks the suite.
- The change would require touching a subsystem the specification never mentioned.`,
		report: `- What changed, file by file, and why.
- The real output of build, linters and tests, remaining failures included.
- What you deliberately left out, and what it would take to finish it.`,
	},
	{
		ID:          "create_pr",
		Name:        "Review & Pull Request",
		DirName:     models.SkillDirNames["create_pr"],
		Command:     "/create-pr",
		FromStage:   "implemented",
		ToStage:     "reviewed",
		Description: "Relit le diff, prépare le commit et ouvre la merge request.",
		Icon:        "ShieldCheck",
		Color:       "purple",
		Steps: []string{
			"Relecture du diff complet",
			"Commit conventionnel et poussée de la branche",
			"Ouverture de la merge request, fusion laissée à l'utilisateur",
			"Label 'reviewed' et transition posés par TaskFlow",
		},
		title:           "Review and Pull Request",
		frontmatterDesc: "Review the branch like a peer would, fix what the review finds, then open the merge request and leave the merge to the user.",
		goal: `Hand a reviewer a branch that is already worth reading: the obvious problems
found and fixed, the risky parts pointed out, the test plan written down.`,
		readFirst: `- The full diff of the branch against the default branch. All of it, not the summary.
- The specification, to check that what was asked is what was built.`,
		stepsBody: `1. Review the diff for correctness, side effects, security, and edge cases with no test.
2. Fix what the review finds, now. A known defect belongs in the code, not in the
   description of the merge request.
3. Re-run build, static analysis and tests on the final state.
4. Commit with a conventional message: type, scope, and why the change exists.
5. Push the branch and open the merge request: summary, test plan, and the specific
   places where you want a reviewer's eyes.
6. If the repository has no remote, say so and stop rather than merging locally.`,
		guardTitle: "Do not",
		guard: `- Do not merge, do not approve, do not close the ticket. That is the user's call.
- Do not open a merge request on a red build. Report the failure instead.`,
		report: `- What the review found, and which findings you fixed.
- The merge request URL, or why there is none.
- The test plan a reviewer can replay, as a checklist.`,
	},
	{
		ID:          "handoff",
		Name:        "Handoff & clôture",
		DirName:     models.SkillDirNames["handoff"],
		Command:     "/handoff-issue",
		FromStage:   "reviewed",
		ToStage:     "finished",
		Description: "Rédige le compte-rendu de fin, vérifie la fusion et nettoie l'espace local.",
		Icon:        "CheckCircle2",
		Color:       "emerald",
		Steps: []string{
			"Vérification que la branche est fusionnée",
			"Compte-rendu de passation et vérifications de recette",
			"Nettoyage du worktree et de la branche locale",
			"Label 'finished' et transition posés par TaskFlow",
		},
		title:           "Handoff and Close",
		frontmatterDesc: "Close the ticket properly: confirm the merge, write the handover and the acceptance checklist, then clean the local workspace.",
		goal: `Leave two things behind: a handover a colleague can act on without asking you,
and a local workspace with nothing stale in it.`,
		readFirst: `- The state of the branch against the default branch.
- What the implementation and review steps reported, so the handover matches reality.`,
		stepsBody: `1. Confirm the ticket's branch is actually merged into the default branch. If it is
   not, stop, say so, and clean nothing.
2. Write the handover: what shipped, what changed for the user, what is still open.
3. Write the acceptance checklist as checkboxes, each item something a human can
   verify in the running product.
4. Update the repository documentation when the change makes it wrong, README and
   changelog included.
5. Turn any remaining follow-up into a separate ticket to create, rather than a
   paragraph nobody will read.
6. Clean up locally: remove the ticket's worktree, delete the local branch once the
   merge is confirmed.`,
		guardTitle: "Do not",
		guard: `- Do not delete anything remote: no remote branch, no tag, no release.
- Do not clean up while the merge is unconfirmed.`,
		report: `- The handover.
- The acceptance checklist, as checkboxes.
- What was cleaned locally, and what could not be, with the reason.
- Follow-up tickets worth creating.`,
	},
	{
		ID:          "pickup",
		Name:        "Pickup & Auto-Pilot to PR",
		DirName:     models.SkillDirNames["pickup"],
		Command:     "/pickup-issue",
		FromStage:   "new",
		ToStage:     "reviewed",
		Interactive: false,
		Description: "Exécute en autonomie complète toutes les étapes d'un ticket jusqu'à la création de la Pull Request.",
		Icon:        "Sparkles",
		Color:       "purple",
		Steps: []string{
			"Cadrage des ambiguïtés (Clarify)",
			"Rédaction de la spécification technique SDD (Specify)",
			"Implémentation incrémentale et passage des tests (Code)",
			"Revue du diff, commit et ouverture de la PR/MR (Create PR)",
			"Mise à jour à chaque étape via le handler local TaskFlow",
		},
		title:           "Pickup Issue (Auto-Pilot to PR)",
		frontmatterDesc: "Pick a ticket and autonomously execute all development steps up to Pull Request creation.",
		goal: `Autonomously take a ticket from its current stage through clarification, specification,
implementation, and testing, all the way to opening a clean Pull Request, updating each stage via TaskFlow.`,
		readFirst: `- The ticket: key, title, description, parent macro, and tracker comments.
- The project's code and existing patterns.
- The project SDD framework (OpenSpec or Spec Kit).`,
		stepsBody: `1. **Pick & Inspect**:
   - Identify the ticket key (<KEY>) and target branch name (<KEY>-<title-slug>).
   - Check the current ticket stage and start from where it currently is.
   - Switch or create the work branch ` + tick + `<KEY>-<title-slug>` + tick + `. Never implement on the default branch.

2. **Step 1: Clarification (if not already clarified)**:
   - Restate the requirements and resolve ambiguities with sensible technical choices.
   - Transition ticket locally: ` + tick + `taskflow stage <KEY> clarified` + tick + `

3. **Step 2: Specification (if not already specified)**:
   - Write the formal technical specification in ` + tick + `openspec/changes/<KEY>-<title-slug>/` + tick + ` or ` + tick + `specs/<KEY>-<title-slug>/` + tick + `.
   - Validate the specification structure and checklist.
   - Transition ticket locally: ` + tick + `taskflow stage <KEY> specified` + tick + `

4. **Step 3: Implementation & Validation**:
   - Implement the changes incrementally on the work branch following the spec checklist.
   - Add automated tests covering the new behavior and edge cases.
   - Run the project's build, linters, and test suite until all checks pass (100% green).
   - Transition ticket locally: ` + tick + `taskflow stage <KEY> implemented --branch "<KEY>-<title-slug>"` + tick + `

5. **Step 4: Review, Push & Pull Request**:
   - Review the complete diff against the default branch to ensure cleanliness.
   - Commit all changes with a clean conventional commit message.
   - Push the branch to the remote repository: ` + tick + `git push -u origin <KEY>-<title-slug>` + tick + `
   - Open the Pull Request / Merge Request via GitHub CLI (` + tick + `gh pr create` + tick + `) or GitLab/Linear tooling.
   - Transition ticket locally: ` + tick + `taskflow stage <KEY> reviewed --pr-url "<PR_URL>"` + tick + `

6. **Step 5: Stop before merge**:
   - Report the PR URL, test results, and summary of changes.
   - Do NOT merge into the default branch (merging is strictly reserved for the human user).`,
		guardTitle: "Do not",
		guard: `- Do not merge into the default branch (merging is reserved for the human user).
- Do not push or open a PR if the test suite is failing.
- Do not skip the local handler stage transitions.`,
		report: `- The created Pull Request URL.
- The work branch and files modified.
- The test results demonstrating that build, lint, and tests pass.
- Summary of settled scope and key architectural decisions.`,
	},
	{
		ID:          "rewrite_story",
		Name:        "Rewrite Story",
		DirName:     models.SkillDirNames["rewrite_story"],
		Command:     "/rewrite-story",
		FromStage:   "",
		ToStage:     "",
		Interactive: false,
		Description: "Reformate la description d'une tâche en User Story structurée GFM, avec inclusion facultative des commentaires.",
		Icon:        "Sparkles",
		Color:       "cyan",
		Steps: []string{
			"Analyse du titre, de la description et des commentaires du ticket",
			"Reformulation au format User Story + Contexte + Critères d'acceptation",
			"Aperçu et confirmation par l'utilisateur",
		},
		title:           "Rewrite Story",
		frontmatterDesc: "Reformat a story or task description into structured markdown, optionally incorporating task comments.",
		goal: `Reformat a task's title, description, and optional comments into a clean GitHub-Flavored Markdown specification (User Story: As a..., I want..., So that... + Context + Acceptance Criteria + Notes).`,
		readFirst: `- The task: title, description, and task comments (if requested or passed as context).
- Standard GitHub-Flavored Markdown (GFM) formatting guidelines.`,
		stepsBody: `1. Inspect the task title, raw description, and comments (if provided).
2. Extract the core intent, user value, technical context, and acceptance criteria.
3. Generate a structured GFM document containing:
   - **User Story**: As a <role>, I want <feature>, So that <benefit>.
   - **Context**: Problem background and technical overview.
   - **Acceptance Criteria**: Checkbox list (- [ ]) of verifiable functional & non-functional requirements.
   - **Notes**: Extra technical details or risks mentioned in comments.
4. Output the reformatted markdown directly for preview and user confirmation.`,
		guardTitle: "Do not",
		guard: `- Do not mutate task title, status, priority, assignee, branch, or pull request.
- Do not delete or overwrite task comments.
- Do not invent artificial requirements not implied by the description or comments.`,
		report: `- The reformatted GFM description preview.
- List of comment points integrated into acceptance criteria (if any).`,
	},
	{
		ID:          "refine_macro",
		Name:        "Refine Macro",
		DirName:     models.SkillDirNames["refine_macro"],
		Command:     "/refine-macro",
		FromStage:   "macro",
		ToStage:     "macro",
		Scope:       "macro",
		Interactive: true,
		Description: "Clarifie de manière interactive le cadrage d'une macro et le décompose en TODOs structurés et cartes TaskFlow.",
		Icon:        "ListChecks",
		Color:       "orange",
		Steps: []string{
			"Analyse du titre et du texte de cadrage de la macro",
			"Évaluation de la complétude du cadrage et questions de clarification si nécessaire",
			"Structuration du plan d'action selon le cadre SDD (SpecKit ou OpenSpec)",
			"Génération des items MacroTodo et découpage des tickets TaskFlow prêts à être créés",
		},
		title:           "Refine Macro",
		frontmatterDesc: "Interactively clarify macro framing text with the user and break it down into structured todos and TaskFlow tickets.",
		goal: `Transform high-level macro framing text into an actionable, structured todo list and concrete TaskFlow tickets, interactively clarifying ambiguities with the user when framing text is vague.`,
		guardTitle: "Do not",
		guard: `- Do not generate tasks blindly when framing text is vague without asking clarification questions.
- Do not overwrite existing todos or tasks without user confirmation in the UI.
- Do not mutate external tracker issues directly without user trigger.`,
		report: `- Clarification Q&A summary (if framing was vague).
- Structured list of proposed MacroTodo items.
- Proposed TaskFlow tickets breakdown (Title, IssueType, Description).`,
	},
}

// StageSkillByID returns the unified skill for an internal id. "review" is the
// historical alias of create_pr, still used by queued jobs.
func StageSkillByID(skillID string) (StageSkill, bool) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "review" {
		skillID = "create_pr"
	}
	if skillID == "pick" || skillID == "pick_issue" || skillID == "pickup_issue" || skillID == "pickup-issue" || skillID == "pick-issue" {
		skillID = "pickup"
	}
	if skillID == "rewrite" || skillID == "rewrite-story" || skillID == "rewrite_story" {
		skillID = "rewrite_story"
	}
	if skillID == "refine" || skillID == "refine-macro" || skillID == "refine_macro" {
		skillID = "refine_macro"
	}
	for _, s := range StageSkills {
		if s.ID == skillID {
			return s, true
		}
	}
	return StageSkill{}, false
}

const tick = "`"

// specifyFrameworkName is the display name of the specification skill: the SDD
// framework is part of what the step actually is.
func specifyFrameworkName(specFramework string) string {
	if strings.EqualFold(strings.TrimSpace(specFramework), "openspec") {
		return "Specify Issue (OpenSpec SDD)"
	} else if strings.EqualFold(strings.TrimSpace(specFramework), "speckit") {
		return "Specify Issue (Spec Kit SDD)"
	}
	return "Specify Issue (Spec-Driven Design)"
}

func refineMacroFrameworkName(specFramework string) string {
	if strings.EqualFold(strings.TrimSpace(specFramework), "openspec") {
		return "Refine Macro (OpenSpec SDD)"
	} else if strings.EqualFold(strings.TrimSpace(specFramework), "speckit") {
		return "Refine Macro (Spec Kit SDD)"
	}
	return "Refine Macro (Spec-Driven Design)"
}

func refineMacroFrameworkBody(specFramework string) (readFirst, steps string) {
	readFirst = `- The macro title and framing description.
- The active project SDD framework (SpecKit or OpenSpec).
- Existing macro todos and child tasks to avoid duplicating completed work.`

	steps = `1. Inspect the macro title and high-level framing description.
2. **Evaluate framing completeness**:
   - If the framing description is empty, under 2 sentences, or lacks clear technical boundaries/acceptance criteria, formulate 3 to 5 numbered clarification questions and ask the user directly in this interactive terminal session before generating tasks.
3. **Decompose & Break Down**:
   - Once answered or if framing text is detailed, group action items according to the selected SDD framework:
     - **SpecKit SDD**: Group into User Stories ([US-x]) and Feature Modules ([FEAT-x]).
     - **OpenSpec SDD**: Group into Capabilities ([CAP-x]) and Change Proposals ([CHANGE-x]).
4. Output the generated checklist of actionable todos AND proposed TaskFlow tickets (Title, IssueType: Story/Task/Bug, Description) for bulk ticket creation.`

	return readFirst, steps
}

// specifyFrameworkBody produces the specification instructions, supporting explicit
// {sdd_framework} parameterization (openspec / speckit) as well as auto-detection.
func specifyFrameworkBody(specFramework string) (readFirst, steps string) {
	readFirst = `- The clarification outcome on the ticket: the decisions are already made, apply them.
- If {sdd_framework} or --framework=<name> is provided, use it. Otherwise, auto-detect:
  - If ` + tick + `openspec/` + tick + ` exists -> use OpenSpec SDD.
  - If ` + tick + `.specify/` + tick + ` or ` + tick + `specs/` + tick + ` exists -> use Spec Kit SDD.
- Ensure the project SDD directory is initialized before writing specifications.`

	steps = `1. Create or switch to the work branch, named <KEY>-<title-slug>. Never write on the default branch.
2. Select the SDD framework from {sdd_framework} argument, flag, or project detection:

   **If using OpenSpec SDD:**
   - Create change directory ` + tick + `openspec/changes/<KEY>-<title-slug>/` + tick + `
   - Write ` + tick + `proposal.md` + tick + ` (problem, value, in/out scope)
   - Write ` + tick + `design.md` + tick + ` (technical decisions, rejected alternatives)
   - Write ` + tick + `tasks.md` + tick + ` (ordered implementation checklist)
   - Write ` + tick + `specs/<capability>/spec.md` + tick + ` (requirements with Given/When/Then)
   - Validate with ` + tick + `openspec validate <change-id> --strict` + tick + `

   **If using Spec Kit SDD:**
   - Write ` + tick + `specs/<KEY>-<title-slug>/spec.md` + tick + ` (prioritised user stories, functional requirements, Given/When/Then)
   - Write ` + tick + `plan.md` + tick + ` (stack, architecture, data contracts, target files)
   - Write ` + tick + `tasks.md` + tick + ` (ordered implementation checklist with test plan)
   - Use ` + tick + `/speckit.specify` + tick + `, ` + tick + `/speckit.plan` + tick + `, ` + tick + `/speckit.tasks` + tick + ` if available.`

	return readFirst, steps
}

// renderTicketTransitionContract generates the autonomous ticket transition instructions
// for the skill based on its from/to stages in the sequence:
// new -> clarified -> specified -> implemented -> reviewed -> finished
func renderTicketTransitionContract(s StageSkill) string {
	var b strings.Builder
	if s.ID == "pickup" {
		b.WriteString("## Ticket Transition & Autonomous Pipeline Contract\n")
		b.WriteString("The agent executing the pickup skill is responsible for advancing the ticket through each stage autonomously up to PR creation:\n")
		b.WriteString("- **Step 1 (Clarify)**: Advance ticket to `clarified` via `taskflow stage <KEY> clarified`\n")
		b.WriteString("- **Step 2 (Specify)**: Advance ticket to `specified` via `taskflow stage <KEY> specified`\n")
		b.WriteString("- **Step 3 (Implement)**: Advance ticket to `implemented` via `taskflow stage <KEY> implemented --branch \"<KEY>-<title-slug>\"`\n")
		b.WriteString("- **Step 4 (Review & PR)**: Advance ticket to `reviewed` via `taskflow stage <KEY> reviewed --pr-url \"<PR_URL>\"`\n")
		b.WriteString("- **HTTP API Alternative** (if CLI not in PATH): `curl -s -X POST http://localhost:8090/api/tasks/stage -H \"Content-Type: application/json\" -d '{\"taskKey\":\"<KEY>\",\"stage\":\"<STAGE>\"}'`\n")
		b.WriteString("- **Fallback to Tracker CLI** (only if TaskFlow is unreachable): `gh issue edit <NUMBER> --add-label \"<STAGE>\"` / `linear issue update <KEY> --add-label \"<STAGE>\"`\n")
		b.WriteString("- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).\n")
		return b.String()
	}

	b.WriteString("## Ticket Transition & Status Update\n")
	b.WriteString("The agent executing this skill is responsible for advancing the ticket to the next agentic status upon completion:\n")
	fmt.Fprintf(&b, "- **Stage Transition**: Advance ticket from `%s` to `%s`.\n", s.FromStage, s.ToStage)
	b.WriteString("- **Step 1: Check and use Local Handler (Recommended if TaskFlow is running)**:\n")
	b.WriteString("  Call TaskFlow's local transition handler to update local state, record branch/PR, and automatically queue two-way synchronization to GitHub/Linear:\n")
	b.WriteString("  - **Via TaskFlow CLI**:\n")
	if s.ID == "code" {
		fmt.Fprintf(&b, "    ```bash\n    taskflow stage <KEY> %s --branch \"<KEY>-<title-slug>\" [\"<optional summary note>\"]\n    ```\n", s.ToStage)
	} else if s.ID == "create_pr" {
		fmt.Fprintf(&b, "    ```bash\n    taskflow stage <KEY> %s --pr-url \"<PR_URL>\" [\"<optional summary note>\"]\n    ```\n", s.ToStage)
	} else {
		fmt.Fprintf(&b, "    ```bash\n    taskflow stage <KEY> %s [\"<optional summary note>\"]\n    ```\n", s.ToStage)
	}
	b.WriteString("  - **Via HTTP API** (port 8090 or 8080):\n")
	b.WriteString("    ```bash\n")
	if s.ID == "code" {
		fmt.Fprintf(&b, "    curl -s -X POST http://localhost:8090/api/tasks/stage -H \"Content-Type: application/json\" -d '{\"taskKey\": \"<KEY>\", \"stage\": \"%s\", \"branch\": \"<KEY>-<title-slug>\"}' || curl -s -X POST http://localhost:8080/api/tasks/stage -H \"Content-Type: application/json\" -d '{\"taskKey\": \"<KEY>\", \"stage\": \"%s\", \"branch\": \"<KEY>-<title-slug>\"}'\n", s.ToStage, s.ToStage)
	} else if s.ID == "create_pr" {
		fmt.Fprintf(&b, "    curl -s -X POST http://localhost:8090/api/tasks/stage -H \"Content-Type: application/json\" -d '{\"taskKey\": \"<KEY>\", \"stage\": \"%s\", \"prUrl\": \"<PR_URL>\"}' || curl -s -X POST http://localhost:8080/api/tasks/stage -H \"Content-Type: application/json\" -d '{\"taskKey\": \"<KEY>\", \"stage\": \"%s\", \"prUrl\": \"<PR_URL>\"}'\n", s.ToStage, s.ToStage)
	} else {
		fmt.Fprintf(&b, "    curl -s -X POST http://localhost:8090/api/tasks/stage -H \"Content-Type: application/json\" -d '{\"taskKey\": \"<KEY>\", \"stage\": \"%s\"}' || curl -s -X POST http://localhost:8080/api/tasks/stage -H \"Content-Type: application/json\" -d '{\"taskKey\": \"<KEY>\", \"stage\": \"%s\"}'\n", s.ToStage, s.ToStage)
	}
	b.WriteString("    ```\n")
	b.WriteString("- **Step 2: Fallback to Direct Tracker CLI (Only if local TaskFlow handler is unreachable)**:\n")
	if s.ToStage == "finished" {
		fmt.Fprintf(&b, "  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label \"%s\" --remove-label \"%s\"` then `gh issue close <NUMBER>`\n", s.ToStage, s.FromStage)
		fmt.Fprintf(&b, "  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label \"%s\" --remove-label \"%s\" --state \"Done\"`\n", s.ToStage, s.FromStage)
	} else {
		fmt.Fprintf(&b, "  - **GitHub CLI**: `gh issue edit <NUMBER> --add-label \"%s\" --remove-label \"%s\"`\n", s.ToStage, s.FromStage)
		fmt.Fprintf(&b, "  - **Linear CLI**: `linear issue update <ISSUE_KEY> --add-label \"%s\" --remove-label \"%s\"`\n", s.ToStage, s.FromStage)
	}
	b.WriteString("- **Comments**: Post the stage summary report as a comment on the ticket via `taskflow stage <KEY> " + s.ToStage + " \"<REPORT_NOTE>\"` or `gh issue comment <NUMBER> --body \"...\"` / `linear issue comment add <ISSUE_KEY> --body \"...\"`.\n")
	b.WriteString("- **Safety Rules**: Always work on the ticket branch (`<KEY>-<title-slug>`). Never delete anything remote and never merge into the default branch (merging is strictly reserved for the human user).\n")
	return b.String()
}

// RenderSkillContent builds the SKILL.md of one skill.
func RenderSkillContent(s StageSkill, specFramework string) string {
	name := s.title
	if name == "" {
		name = s.Name
	}
	readFirst := s.readFirst
	steps := s.stepsBody
	if s.ID == "specify" {
		name = specifyFrameworkName(specFramework)
		readFirst, steps = specifyFrameworkBody(specFramework)
	}
	if s.ID == "refine_macro" {
		name = refineMacroFrameworkName(specFramework)
		readFirst, steps = refineMacroFrameworkBody(specFramework)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "---\nname: %s\ndescription: %s\n---\n", s.DirName, s.frontmatterDesc)
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "Stage: %s -> %s.", s.FromStage, s.ToStage)
	if s.Interactive {
		b.WriteString(" Interactive: the user answers in the terminal.")
	}
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "## Goal\n%s\n\n", s.goal)
	if readFirst != "" {
		fmt.Fprintf(&b, "## Read first\n%s\n\n", readFirst)
	}
	if steps != "" {
		fmt.Fprintf(&b, "## Steps\n%s\n\n", steps)
	}
	if s.guard != "" {
		fmt.Fprintf(&b, "## %s\n%s\n\n", s.guardTitle, s.guard)
	}
	fmt.Fprintf(&b, "## Report\n%s\n\n", s.report)
	b.WriteString(renderTicketTransitionContract(s))
	return b.String()
}

// ProjectSkillTemplates returns the unified set ready to be written, with the
// specification skill resolved for the project's SDD framework.
func ProjectSkillTemplates(specFramework string) []ProjectSkillTemplate {
	out := make([]ProjectSkillTemplate, 0, len(StageSkills))
	for _, s := range StageSkills {
		name := s.Name
		if s.ID == "specify" {
			name = specifyFrameworkName(specFramework)
		}
		if s.ID == "refine_macro" {
			name = refineMacroFrameworkName(specFramework)
		}
		out = append(out, ProjectSkillTemplate{
			ID:          s.ID,
			Name:        name,
			DirName:     s.DirName,
			Description: s.Description,
			Content:     RenderSkillContent(s, specFramework),
		})
	}
	return out
}

// SkillDirsFor returns the skill directories of one skill inside a checkout,
// one per agent CLI. The same file is written to all of them: one source,
// several readers.
func SkillDirsFor(root, dirName string) []string {
	dirs := make([]string, 0, len(models.SkillAgentDirs))
	for _, agent := range models.SkillAgentDirs {
		if agent == "" {
			// La convention agnostique, a la racine : .skills/<nom>
			dirs = append(dirs, filepath.Join(root, ".skills", dirName))
			continue
		}
		dirs = append(dirs, filepath.Join(root, agent, "skills", dirName))
	}
	return dirs
}

// SkillCommandPath is where a skill's slash command lives for Claude Code.
//
// Une skill et une commande ne sont pas la même chose : une skill sous
// .claude/skills/<nom>/SKILL.md est choisie par le modèle quand il la juge
// pertinente, alors qu'une commande sous .claude/commands/<nom>.md est
// invocable par « /<nom> ». TaskFlow appelle ses étapes par leur commande, donc
// il faut écrire les deux, sinon « claude -p "/clarify-issue PROJ-123" » se
// contente de recopier le texte.
func SkillCommandPath(root, dirName string) string {
	return filepath.Join(root, ".claude", "commands", dirName+".md")
}

// RenderSkillCommand turns a rendered SKILL.md into its slash command: same
// instructions, a command frontmatter, and the ticket passed as $ARGUMENTS.
func RenderSkillCommand(s StageSkill, specFramework string) string {
	body := RenderSkillContent(s, specFramework)

	// Le frontmatter d'une skill porte name et description ; celui d'une commande
	// porte description et argument-hint. On retire le premier pour écrire le bon.
	if strings.HasPrefix(body, "---\n") {
		if end := strings.Index(body[4:], "\n---\n"); end >= 0 {
			body = body[4+end+5:]
		}
	}

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: %s\n", s.frontmatterDesc)
	b.WriteString("argument-hint: <TICKET-KEY> [contexte]\n")
	b.WriteString("---\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n\n## Ticket\n$ARGUMENTS\n")
	return b.String()
}

// CommandContentFor returns the slash command content of a skill id, with the
// project's specification framework resolved.
func CommandContentFor(skillID, specFramework string) (string, bool) {
	stage, ok := StageSkillByID(skillID)
	if !ok {
		return "", false
	}
	return RenderSkillCommand(stage, specFramework), true
}
