package runner

import (
	"strings"
	"testing"

	"tasks/internal/models"
)

// Le lancement en session doit résoudre le moteur du projet, pas un moteur par
// défaut : c'est ce qui produisait « binaire agy introuvable » sur une machine
// où seul claude est installé.
func TestInteractiveAgentLaunchUnknownEngineExplainsWhereItLooked(t *testing.T) {
	_, err := InteractiveAgentLaunch(&models.Settings{AIProvider: "agy-qui-nexiste-pas"})
	if err == nil {
		t.Fatal("un moteur inconnu doit être refusé")
	}
	if !strings.Contains(err.Error(), "n'a pas de mode interactif") {
		t.Fatalf("message peu actionnable: %v", err)
	}
}

func TestInteractiveAgentLaunchMissingBinaryNamesTheSearchPath(t *testing.T) {
	_, err := InteractiveAgentLaunch(&models.Settings{AIProvider: "vibe"})
	if err == nil {
		t.Skip("vibe est installé sur cette machine, rien à vérifier ici")
	}
	for _, want := range []string{"PATH", ".local/bin", "homebrew"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("le message doit dire où il a cherché (%s manquant): %v", want, err)
		}
	}
}

// Un moteur personnalisé n'a que son modèle de commande : le binaire est son
// premier mot.
func TestInteractiveAgentLaunchCustomUsesTemplateBinary(t *testing.T) {
	line, err := InteractiveAgentLaunch(&models.Settings{
		AIProvider:        "custom",
		AICommandTemplate: `sh -c "echo {prompt}"`,
	})
	if err != nil {
		t.Fatalf("le premier mot du modèle doit être retenu: %v", err)
	}
	if !strings.Contains(line, "sh") {
		t.Fatalf("binaire attendu sh, obtenu %q", line)
	}
}

func TestSkillCallLineFollowsProjectCommand(t *testing.T) {
	task := &models.Task{Key: "SFE-238", Title: "Titre\navec saut", Source: "jira"}
	got := SkillCallLineWithCommand("clarify-workitem", task, "jira")
	want := "/clarify-workitem SFE-238 (Titre avec saut) suivi dans jira"
	if got != want {
		t.Fatalf("ligne inattendue:\n obtenu %q\n attendu %q", got, want)
	}
}
