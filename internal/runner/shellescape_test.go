package runner

import (
	"testing"
)

// TestEscapeForDoubleQuotesNeutralisesInjection locks the fix for command
// injection through task titles: the AI command template is run via 'sh -c',
// and titles come from external trackers.
func TestEscapeForDoubleQuotesNeutralisesInjection(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"double quote breakout", `"; touch /tmp/pwned; echo "`},
		{"command substitution", "$(touch /tmp/pwned)"},
		{"backtick substitution", "`touch /tmp/pwned`"},
		{"variable expansion", "$HOME/secret"},
		{"backslash", `a\b`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			escaped := escapeForDoubleQuotes(c.in)

			// The escaped string, placed inside double quotes, must round-trip
			// back to the original when the shell parses it.
			out, err := (&Runner{}).runCommandForTest(`printf %s "` + escaped + `"`)
			if err != nil {
				t.Fatalf("shell rejected the escaped payload: %v (escaped=%q)", err, escaped)
			}
			if out != c.in {
				t.Fatalf("payload altered by the shell:\n  want %q\n  got  %q\n  escaped %q", c.in, out, escaped)
			}
		})
	}
}
