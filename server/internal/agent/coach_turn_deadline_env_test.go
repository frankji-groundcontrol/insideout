package agent

import (
	"testing"
	"time"
)

// rv4-FULL-09 (residue R3-35): docker-compose.yml passes the knob as
// ${INSIDEOUT_AI_TURN_DEADLINE:-}, so a stack whose .env never sets it hands
// the server an EMPTY value, not an absent one. An empty, unparsable or
// non-positive value must keep the 12m default (the number the environment
// reference and .env.example document), or every compose deployment would cut
// every coach turn short at once. Characterization: it passed before the
// compose line existed and pins what the line relies on.
func TestRvEnvTruthTurnDeadlineEnvKeepsTheDefaultUnlessValid(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"empty, as compose passes an unset variable", "", 12 * time.Minute},
		{"unparsable", "soon", 12 * time.Minute},
		{"zero", "0s", 12 * time.Minute},
		{"negative", "-5m", 12 * time.Minute},
		{"a valid duration wins", "30m", 30 * time.Minute},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("INSIDEOUT_AI_TURN_DEADLINE", tt.value)
			if got := turnDeadlineFromEnv(); got != tt.want {
				t.Errorf("INSIDEOUT_AI_TURN_DEADLINE=%q: turn deadline = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
