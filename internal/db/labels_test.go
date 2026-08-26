package db

import "testing"

// Poser une étape doit retirer toutes les autres côté tracker : Jira ne fait pas
// le remplacement tout seul, et un ticket accumulait clarified, specified,
// implemented… alors que Taskacao n'en montrait qu'un.
func TestStaleWorkflowLabelsExcludesTargetOnly(t *testing.T) {
	stale := StaleWorkflowLabels("implemented")

	for _, label := range stale {
		if label == "implemented" || label == "Implemented" {
			t.Fatalf("le label visé ne doit pas être retiré, trouvé %q", label)
		}
	}

	// Les deux graphies des autres étapes doivent être visées, Jira distinguant
	// la casse.
	for _, want := range []string{"clarified", "Clarified", "reviewed", "Reviewed", "new", "New"} {
		found := false
		for _, label := range stale {
			if label == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q attendu dans les labels à retirer", want)
		}
	}
}

// SetWorkflowLabel, lui, remplace bien en local.
func TestSetWorkflowLabelReplacesLocally(t *testing.T) {
	out := SetWorkflowLabel([]string{"ai:tech:autonomous", "clarified"}, "specified")
	if len(out) != 2 || out[0] != "ai:tech:autonomous" || out[1] != "specified" {
		t.Fatalf("remplacement attendu, obtenu %v", out)
	}
}
