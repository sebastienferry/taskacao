package runner

import "testing"

// Un ticket dont ni le titre ni la description ni les labels n'ont bougé ne doit
// produire aucun appel à acli : c'est ce qui protège la mise en forme Jira, que
// notre copie aplatie écraserait. Le test le prouve sans réseau — si un appel
// était fait, il échouerait faute de ticket réel.
func TestUpdateJiraIssueSkipsUntouchedFields(t *testing.T) {
	r := NewRunner()
	if err := r.UpdateJiraIssue("ZZZ-000", "", nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("aucun champ édité : aucun appel attendu, erreur reçue: %v", err)
	}
}

// jiraCLIFailure doit reconnaître le refus qu'acli imprime en sortant en 0.
func TestJiraCLIFailureDetectsRefusal(t *testing.T) {
	cases := map[string]bool{
		"✓ Work item PE-1 has been successfully transitioned to Open":                    false,
		"✗ Failure: PE-1 can't be transitioned: No allowed transitions found for status": true,
		"": false,
	}
	for output, wantFailure := range cases {
		got := jiraCLIFailure(output) != ""
		if got != wantFailure {
			t.Errorf("jiraCLIFailure(%q) = %v, attendu %v", output, got, wantFailure)
		}
	}
}
