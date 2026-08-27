package runner

import (
	"encoding/json"
	"testing"
)

func TestParseJiraTeamValue(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		wantID   string
		wantName string
	}{
		{
			name:     "atlassian team object",
			raw:      `{"id":"6aafbf90-c8b2-4ed3-a196-01d2dccd837a","name":"Transversal - PE DevEx","isVerified":false}`,
			wantID:   "6aafbf90-c8b2-4ed3-a196-01d2dccd837a",
			wantName: "Transversal - PE DevEx",
		},
		{
			// Une équipe rattachée au site porte un suffixe qui fait partie de
			// l'identifiant : le tronquer donne un 404 sur ses membres.
			name:     "site scoped team keeps its suffix",
			raw:      `{"id":"53fb2415-9173-4eaa-9357-a735bb6a42e8-67","name":"Transversal - ARC"}`,
			wantID:   "53fb2415-9173-4eaa-9357-a735bb6a42e8-67",
			wantName: "Transversal - ARC",
		},
		{
			name:     "label under title",
			raw:      `{"id":"abc","title":"QE"}`,
			wantID:   "abc",
			wantName: "QE",
		},
		{
			name:     "plain string field",
			raw:      `"Transversal - PE Core"`,
			wantID:   "",
			wantName: "Transversal - PE Core",
		},
		{
			name:     "empty field",
			raw:      `null`,
			wantID:   "",
			wantName: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, name := parseJiraTeamValue(json.RawMessage(tc.raw))
			if id != tc.wantID {
				t.Errorf("id = %q, attendu %q", id, tc.wantID)
			}
			if name != tc.wantName {
				t.Errorf("name = %q, attendu %q", name, tc.wantName)
			}
		})
	}
}

func TestNormalizeIssueTypes(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "vide retombe sur les types par défaut", in: nil, want: []string{"Task", "Story"}},
		{name: "liste vide aussi", in: []string{"", "  "}, want: []string{"Task", "Story"}},
		{
			// Un projet dont le tracker n'expose que son propre type doit pouvoir
			// le nommer : sans cela sa synchro ne ramène rien du tout.
			name: "type propre au projet",
			in:   []string{"Platform Feedback"},
			want: []string{"Platform Feedback"},
		},
		{name: "doublons et espaces retirés", in: []string{" Task ", "task", "Story"}, want: []string{"Task", "Story"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeIssueTypes(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
