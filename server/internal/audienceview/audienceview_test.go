package audienceview

import "testing"

func TestGetAllAudiences(t *testing.T) {
	for _, a := range Audiences() {
		p, err := Get(a)
		if err != nil {
			t.Fatalf("Get(%s): %v", a, err)
		}
		if len(p.Sections) == 0 || p.Title == "" || p.Purpose == "" {
			t.Errorf("projection %s incomplete: %+v", a, p)
		}
		for _, s := range p.Sections {
			if s.Key == "" || s.Why == "" {
				t.Errorf("projection %s has a pick without key/why: %+v", a, s)
			}
		}
	}
}

func TestUnknownAudience(t *testing.T) {
	if Valid("boss") {
		t.Error("boss accepted as audience")
	}
	if _, err := Get("boss"); err == nil {
		t.Error("Get(boss) should error")
	}
}

func TestEveryPickIsARealSection(t *testing.T) {
	// The fixed template keys (store.PrdSectionKeys), stated locally so
	// this package stays import-clean.
	valid := map[string]bool{
		"background": true, "constraints": true, "goals": true, "nonGoals": true,
		"requirements": true, "risks": true, "stories": true, "users": true,
	}
	for _, a := range Audiences() {
		p, _ := Get(a)
		for _, s := range p.Sections {
			if !valid[s.Key] {
				t.Errorf("%s references unknown section %q", a, s.Key)
			}
		}
	}
}
