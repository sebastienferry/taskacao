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
			raw:      `{"id":"11111111-2222-3333-4444-555555555555","name":"Platform Team","isVerified":false}`,
			wantID:   "11111111-2222-3333-4444-555555555555",
			wantName: "Platform Team",
		},
		{
			// Une équipe rattachée au site porte un suffixe qui fait partie de
			// l'identifiant : le tronquer donne un 404 sur ses membres.
			name:     "site scoped team keeps its suffix",
			raw:      `{"id":"66666666-7777-8888-9999-000000000000-42","name":"Site Scoped Team"}`,
			wantID:   "66666666-7777-8888-9999-000000000000-42",
			wantName: "Site Scoped Team",
		},
		{
			name:     "label under title",
			raw:      `{"id":"abc","title":"Quality"}`,
			wantID:   "abc",
			wantName: "Quality",
		},
		{
			name:     "plain string field",
			raw:      `"Plain Text Team"`,
			wantID:   "",
			wantName: "Plain Text Team",
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
			in:   []string{"Customer Feedback"},
			want: []string{"Customer Feedback"},
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
