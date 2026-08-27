package db

import (
	"strings"
	"testing"
)

func TestDigestAgendaPromptUsesTheCustomOne(t *testing.T) {
	// Quelqu'un qui a déjà sa propre commande de brief la met ici : le prompt
	// devient cette ligne, sans que le texte d'origine s'y ajoute.
	got := digestAgendaPrompt("SFE", "2026-08-27", "/daily-brief {date}")
	if got != "/daily-brief 2026-08-27" {
		t.Errorf("prompt = %q, attendu la commande substituée", got)
	}
}

func TestDigestAgendaPromptFallsBackToDefault(t *testing.T) {
	for _, custom := range []string{"", "   "} {
		got := digestAgendaPrompt("SFE", "2026-08-27", custom)
		if !strings.Contains(got, "Agenda du jour") {
			t.Errorf("un réglage vide doit rendre le prompt d'origine, obtenu %q", got)
		}
		// Le défaut porte les mêmes marqueurs : ils doivent être remplacés, sinon
		// l'agent recevrait « le projet {project} ».
		if strings.Contains(got, "{project}") || strings.Contains(got, "{date}") {
			t.Errorf("marqueur non substitué dans %q", got)
		}
		if !strings.Contains(got, "SFE") || !strings.Contains(got, "2026-08-27") {
			t.Errorf("projet ou date absents de %q", got)
		}
	}
}
