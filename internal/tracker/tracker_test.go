package tracker

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCapabilitiesDifferPerTracker(t *testing.T) {
	cases := []struct {
		name   string
		writer Writer
		has    []Capability
		lacks  []Capability
	}{
		{
			name:   "linear",
			writer: NewLinearWriter(),
			has:    []Capability{CapTransition, CapLabels},
			// Sprint et équipe sont exactement ce que les autres trackers n'ont
			// pas : c'est la raison d'être de cette interface.
			lacks: []Capability{CapSprint, CapTeam, CapEpic},
		},
		{
			name:   "github",
			writer: NewGithubWriter(),
			has:    []Capability{CapTransition, CapAssign},
			lacks:  []Capability{CapSprint, CapTeam, CapEpic},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.has {
				if !tc.writer.Supports(c) {
					t.Errorf("%s devrait prendre en charge %q", tc.name, c)
				}
			}
			for _, c := range tc.lacks {
				if tc.writer.Supports(c) {
					t.Errorf("%s ne devrait pas prendre en charge %q", tc.name, c)
				}
			}
		})
	}
}

func TestUnsupportedNamesWhatIsMissing(t *testing.T) {
	err := NewLinearWriter().SetSprint(context.Background(), "42", []string{"PROJ-1"})
	if err == nil {
		t.Fatal("un tracker sans sprint doit refuser")
	}
	if !IsUnsupported(err) {
		t.Errorf("le refus doit être reconnaissable comme une capacité absente, pas comme une panne")
	}
	// Le message doit nommer l'opération : « ça n'a pas marché » n'apprend rien à
	// qui vient de cliquer.
	if !strings.Contains(err.Error(), "sprint") {
		t.Errorf("message = %q, il devrait nommer le sprint", err.Error())
	}
}

func TestIsUnsupportedIgnoresOtherErrors(t *testing.T) {
	if IsUnsupported(errors.New("le réseau a coupé")) {
		t.Error("une panne réseau n'est pas une capacité absente")
	}
	if IsUnsupported(nil) {
		t.Error("nil n'est pas un refus")
	}
}
