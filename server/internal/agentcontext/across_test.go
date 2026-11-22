package agentcontext

import (
	"reflect"
	"testing"
)

// acrossNodes: Root → {Pay → {Card, Wallet}, Ship, Done}. Card waits on Ship
// and on Done (finished); Ship waits on Wallet; Wallet waits on Card (inside).
func acrossNodes() []NodeIn {
	return []NodeIn{
		{ID: "root", Title: "Root", Status: "pending"},
		{ID: "pay", ParentID: "root", Title: "Pay", Status: "in_progress", Position: 0},
		{ID: "ship", ParentID: "root", Title: "Ship", Status: "pending", Position: 1, DependsOn: []string{"wallet"}},
		{ID: "done", ParentID: "root", Title: "Done", Status: "done", Position: 2},
		{ID: "card", ParentID: "pay", Title: "Card", Status: "pending", Position: 0, DependsOn: []string{"ship", "done"}},
		{ID: "wallet", ParentID: "pay", Title: "Wallet", Status: "in_progress", Position: 1, DependsOn: []string{"card"}},
	}
}

// The focus says what its whole subtree waits on outside it, what outside
// waits on it, and its own steps in order; finished work drops out.
func TestTheFocusSaysWhatItWaitsOnAndBlocksAcrossItsSteps(t *testing.T) {
	focus := func(id string) map[string]any {
		return Assemble(Inputs{Mode: "implementation", FocusNodeID: id, Nodes: acrossNodes()})["focus"].(map[string]any)
	}
	pay := focus("pay")
	if !reflect.DeepEqual(pay["waitsOn"], []string{"Ship"}) || !reflect.DeepEqual(pay["blocks"], []string{"Ship"}) {
		t.Errorf("Pay waitsOn %v blocks %v, want Ship both ways", pay["waitsOn"], pay["blocks"])
	}
	// Wallet waits on Card: one step after it, inside Pay.
	wantSteps := []map[string]any{{"id": "card", "title": "Card", "stage": 0}, {"id": "wallet", "title": "Wallet", "stage": 1}}
	if !reflect.DeepEqual(pay["steps"], wantSteps) {
		t.Errorf("Pay steps = %v, want %v", pay["steps"], wantSteps)
	}
	root := focus("root")
	if _, has := root["waitsOn"]; has {
		t.Errorf("Root waits on %v, but nothing lies outside it", root["waitsOn"])
	}
	// Pay and Ship wait on each other through their work: one loop, one
	// stage, after Done (finished work still orders the level).
	if got := root["steps"]; !reflect.DeepEqual(got, []map[string]any{
		{"id": "done", "title": "Done", "stage": 0}, {"id": "pay", "title": "Pay", "stage": 1}, {"id": "ship", "title": "Ship", "stage": 1},
	}) {
		t.Errorf("Root steps = %v", got)
	}
	if card := focus("card"); card["steps"] != nil || !reflect.DeepEqual(card["blocks"], []string{"Wallet"}) {
		t.Errorf("Card (a leaf) steps %v blocks %v, want no steps and Wallet blocked", card["steps"], card["blocks"])
	}
}

// With no focus, a person in charge of several branches is briefed on those
// branches only, and told which they are.
func TestABriefingWithoutAFocusKeepsToWhatTheHumanIsInChargeOf(t *testing.T) {
	out := Assemble(Inputs{Mode: "implementation", InCharge: []string{"pay", "ship"}, Nodes: acrossNodes()})
	if got := out["inCharge"]; !reflect.DeepEqual(got, []map[string]string{{"id": "pay", "title": "Pay"}, {"id": "ship", "title": "Ship"}}) {
		t.Errorf("inCharge = %v", got)
	}
	var leaves []string
	for _, l := range out["leaves"].([]map[string]any) {
		leaves = append(leaves, l["id"].(string))
	}
	if !reflect.DeepEqual(leaves, []string{"ship", "card", "wallet"}) {
		t.Errorf("leaves = %v, want Ship, Card and Wallet, not Done", leaves)
	}
	if _, has := out["focus"]; has {
		t.Error("several roots in charge still set a focus")
	}
}
