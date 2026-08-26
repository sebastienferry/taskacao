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
// The Taskacao interface stays in French.
type StageSkill struct {
	ID          string // internal id, shared by the catalogue and the job queue
	Name        string
	DirName     string // skill directory, which is also the slash command
	Command     string
	FromStage   string
	ToStage     string
	Interactive bool
	Description string // shown in the Taskacao interface, in French
	Icon        string
	Color       string
	Steps       []string // the step list the interface displays

	// Body of the SKILL.md. The renderer adds the header, the stage line and the
	// contract with Taskacao. title is the English heading of the file, kept
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
		Interactive: true,
		Description: "Analyse les ambiguïtés techniques et produit 3 à 5 questions de cadrage.",
		Icon:        "HelpCircle",
		Color:       "amber",
		Steps: []string{
			"Lecture du ticket et du code concerné",
			"Détection des ambiguïtés et des dépendances",
			"Questions de cadrage posées en session interactive",
			"Label 'clarified' et transition posés par Taskacao",
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
4. Ask 3 to 5 numbered questions. Prefer closed questions. For each one, state the
   option you recommend and why, so silence still leaves a usable default.
5. Wait for the answers. Then write the settled scope: what was decided, what was
   explicitly dropped, what stays open on purpose.`,
		guardTitle: "Do not",
		guard: `- Do not write production code at this stage, and do not start the specification.
- Do not invent an answer to your own question and move on.
- Do not pad the list to reach five questions.`,
		report: `- Restated request and scope.
- Ambiguities, worst first.
- Critical dependencies.
- Numbered questions with your recommended option.
- After the answers: the settled scope.`,
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
			"Label 'specified' et transition posés par Taskacao",
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
			"Label 'implemented' et transition posés par Taskacao",
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
			"Label 'reviewed' et transition posés par Taskacao",
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
			"Label 'finished' et transition posés par Taskacao",
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
}

// StageSkillByID returns the unified skill for an internal id. "review" is the
// historical alias of create_pr, still used by queued jobs.
func StageSkillByID(skillID string) (StageSkill, bool) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "review" {
		skillID = "create_pr"
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
	}
	return "Specify Issue (Spec Kit SDD)"
}

// specifyFrameworkBody is the one body that depends on the project: the SDD
// framework decides which files the specification is written into. The exact
// paths are spelled out on purpose, so the agent never has to guess the layout.
func specifyFrameworkBody(specFramework string) (readFirst, steps string) {
	if strings.EqualFold(strings.TrimSpace(specFramework), "openspec") {
		readFirst = `- The clarification outcome on the ticket: the decisions are already made, apply them.
- ` + tick + `openspec/project.md` + tick + `, plus the specs already written under ` + tick + `openspec/specs/` + tick + `,
  so you know which capabilities are described already.
- This project must be initialised with OpenSpec. If ` + tick + `openspec/` + tick + ` is missing, say so
  and stop: install OpenSpec from the Taskacao project settings, or run
  ` + tick + `openspec init` + tick + ` at the repository root.`
		steps = `1. Create or switch to the work branch, named <KEY>-<title-slug>. Never write on the
   default branch.
2. Create the change folder ` + tick + `openspec/changes/<KEY>-<title-slug>/` + tick + ` and write:
   - ` + tick + `proposal.md` + tick + `: the why. Problem, value, what is in scope and what is not.
   - ` + tick + `design.md` + tick + `: the technical decisions, and the alternatives you rejected.
   - ` + tick + `tasks.md` + tick + `: the ordered, individually verifiable implementation checklist.
   - ` + tick + `specs/<capability>/spec.md` + tick + `: the behaviour deltas, as ` + tick + `## ADDED` + tick + ` /
     ` + tick + `## MODIFIED` + tick + ` / ` + tick + `## REMOVED` + tick + ` requirements with Given / When / Then
     scenarios.
3. Validate with ` + tick + `openspec validate <change-id> --strict` + tick + ` and fix everything it reports.
4. Use the agent's OpenSpec commands if it exposes any, rather than writing the
   files by hand.`
		return readFirst, steps
	}

	readFirst = `- The clarification outcome on the ticket: the decisions are already made, apply them.
- ` + tick + `.specify/memory/constitution.md` + tick + `, so the specification respects the project's
  own principles, plus the specifications already written under ` + tick + `specs/` + tick + `.
- This project must be initialised with Spec Kit. If ` + tick + `.specify/` + tick + ` is missing, say so
  and stop: install Spec Kit from the Taskacao project settings, or run
  ` + tick + `specify init --here` + tick + ` at the repository root.`
	steps = `1. Create or switch to the work branch, named <KEY>-<title-slug>. Never write on the
   default branch.
2. Write ` + tick + `specs/<KEY>-<title-slug>/spec.md` + tick + `: the what and the why, with no
   implementation choices. Context, prioritised user stories, explicit out of scope,
   numbered functional requirements, and Given / When / Then acceptance criteria.
3. Write ` + tick + `plan.md` + tick + `: the how. Stack, architecture, data contracts, target files,
   and a Mermaid flow diagram when the flow is not obvious from the text.
4. Write ` + tick + `tasks.md` + tick + `: the ordered, individually verifiable implementation
   checklist, with the test plan attached to each item.
5. Use ` + tick + `/speckit.specify` + tick + `, ` + tick + `/speckit.plan` + tick + ` then ` + tick + `/speckit.tasks` + tick + ` if the
   agent exposes the Spec Kit commands, rather than writing the files by hand.`
	return readFirst, steps
}

// taskacaoContract is appended to every skill. It is the point of the
// unification: the agent produces the work and the report, Taskacao owns the
// ticket. Without it, a skill either tried to move the ticket itself or assumed
// somebody else would, and nobody did.
const taskacaoContract = `## Contract with Taskacao
Taskacao owns the ticket. Do not change its status, labels, or parent, and do not
post comments on it: the report above is published by Taskacao when the step ends.
Work in the current directory, which Taskacao already picked for this ticket.
Never delete anything remote and never merge a branch unless asked. Report
faithfully: a failing check is reported with its output, a skipped step is stated.`

// RenderSkillContent builds the SKILL.md of one skill. Same sections in the same
// order for the five, which is what makes them reviewable side by side.
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
	b.WriteString(taskacaoContract)
	b.WriteString("\n")
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
// invocable par « /<nom> ». Taskacao appelle ses étapes par leur commande, donc
// il faut écrire les deux, sinon « claude -p "/clarify-issue SFE-331" » se
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
