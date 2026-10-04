package character

import "testing"

func TestGradeCountsValidation(t *testing.T) {
	for _, g := range []GradeCounts{{}, {S: 2}, {S: 1, A: 1}, {D: MaxGradeCount}} {
		if err := g.Validate(); err != nil {
			t.Fatalf("%+v: %v", g, err)
		}
	}
	for _, g := range []GradeCounts{{S: -1}, {A: -1}, {B: -1}, {C: -1}, {D: -1}, {S: MaxGradeCount + 1}} {
		if err := g.Validate(); err == nil {
			t.Fatalf("accepted %+v", g)
		}
	}
}
