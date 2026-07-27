package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

func coachTools() []Tool {
	sectionEnum := make([]any, len(store.PrdSectionKeys))
	for i, k := range store.PrdSectionKeys {
		sectionEnum[i] = k
	}
	kindEnum := make([]any, 0, len(factKinds))
	for k := range factKinds {
		kindEnum = append(kindEnum, k)
	}

	return []Tool{
		{
			Name:        "get_prd",
			Description: "Read the PRD's current title and all section contents.",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "update_prd_section",
			Description: "Write (replace) the markdown content of one PRD section. If the section relies on recorded facts, pass their ids in section_facts so the evidence panel can show what grounds it.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"section":       map[string]any{"type": "string", "enum": sectionEnum},
					"markdown":      map[string]any{"type": "string"},
					"section_facts": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "ids of facts (from record_fact) that ground this section's content"},
				},
				"required": []any{"section", "markdown"},
			},
		},
		{
			Name:        "advance_stage",
			Description: "Move the coaching conversation to the next stage.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"next": map[string]any{"type": "string", "enum": []any{StageDraft, StageCritique, StageFinalize}},
				},
				"required": []any{"next"},
			},
		},
		{
			Name:        "record_fact",
			Description: "Record a fact the user actually stated — quote must be their verbatim words. Call this before relying on anything the user told you; never invent a quote.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":  map[string]any{"type": "string", "enum": kindEnum},
					"text":  map[string]any{"type": "string", "description": "the fact, in your own words"},
					"quote": map[string]any{"type": "string", "description": "the user's verbatim words that support this fact"},
				},
				"required": []any{"kind", "text", "quote"},
			},
		},
		{
			Name:        "mark_assumption",
			Description: "Record something you're proposing that the user hasn't confirmed yet. Renders as [ASSUMPTION] in the PRD — use this instead of stating unconfirmed things as fact.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"text": map[string]any{"type": "string"}},
				"required":   []any{"text"},
			},
		},
		{
			Name:        "resolve_finding",
			Description: "Mark a critic finding resolved (you fixed it) or overridden (the user decided to keep it as-is). Call this once the user has responded to a finding you presented.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"status": map[string]any{"type": "string", "enum": []any{FindingResolved, FindingOverridden}},
				},
				"required": []any{"id", "status"},
			},
		},
	}
}

// everyoneLintPhrases blocks a segment fact that names no real audience —
// bilingual since the product is Chinese-first (prompts.go). Deliberately
// short and literal, not NLP: false negatives (a cleverly-worded "everyone"
// getting through) are fine, this is a cheap tripwire, not a classifier.
var everyoneLintPhrases = []string{
	"everyone", "everybody", "all users", "anyone", "any user",
	"所有人", "所有用户", "任何人", "全体用户", "大家",
}

func segmentFailsEveryoneLint(text string) bool {
	lower := strings.ToLower(text)
	for _, p := range everyoneLintPhrases {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// toolExecutor applies a single tool call against the store and reports
// side effects (section writes, stage changes, facts) via callbacks so
// coach.go can emit the matching SSE events.
type toolExecutor struct {
	store          *store.Store
	actorID        uuid.UUID
	prdID          uuid.UUID
	conversationID uuid.UUID
	currentStage   string
	userMessages   []string // this conversation's user turns, for record_fact quote grounding
	onSectionWrite func(section string)
	onStageChange  func(stage string)
	onFactRecorded func(f Fact)
}

func (e *toolExecutor) Execute(ctx context.Context, call ToolCallRequest) (string, error) {
	switch call.Name {
	case "get_prd":
		return e.getPrd(ctx)
	case "update_prd_section":
		return e.updateSection(ctx, call.Arguments)
	case "advance_stage":
		return e.advanceStage(ctx, call.Arguments)
	case "record_fact":
		return e.recordFact(ctx, call.Arguments)
	case "mark_assumption":
		return e.markAssumption(ctx, call.Arguments)
	case "resolve_finding":
		return e.resolveFinding(ctx, call.Arguments)
	default:
		return "", fmt.Errorf("agent: unknown tool %q", call.Name)
	}
}

func (e *toolExecutor) getPrd(ctx context.Context) (string, error) {
	prd, err := e.store.GetPrdForMember(ctx, e.prdID, e.actorID)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(map[string]any{"title": prd.Title, "sections": prd.Sections})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (e *toolExecutor) updateSection(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Section      string   `json:"section"`
		Markdown     string   `json:"markdown"`
		SectionFacts []string `json:"section_facts"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("agent: invalid update_prd_section arguments: %w", err)
	}
	valid := false
	for _, k := range store.PrdSectionKeys {
		if k == args.Section {
			valid = true
			break
		}
	}
	if !valid {
		return "", fmt.Errorf("agent: unknown section %q", args.Section)
	}

	prd, err := e.store.GetPrdForMember(ctx, e.prdID, e.actorID)
	if err != nil {
		return "", err
	}
	// title=nil: a section edit never rewrites the title (COALESCE leaves it
	// untouched) — the coach only owns section content, not the PRD title.
	if _, err := e.store.UpdateSections(ctx, e.actorID, e.prdID, nil, map[string]string{args.Section: args.Markdown}, &prd.UpdatedAt); err != nil {
		if err == store.ErrConflict {
			return "", fmt.Errorf("agent: this section was edited by someone else since you last read it — call get_prd again to see the current content before retrying")
		}
		return "", err
	}
	// Best-effort: the section write itself already succeeded; the
	// evidence mapping is a display aid, not the source of truth.
	_ = e.recordSectionFacts(ctx, args.Section, args.SectionFacts)
	if e.onSectionWrite != nil {
		e.onSectionWrite(args.Section)
	}
	return fmt.Sprintf("section %q updated", args.Section), nil
}

// recordSectionFacts persists which fact ids the model says ground a
// section, out-of-band in conversation meta (never embedded in the
// section markdown itself — see plan §4.3) so the evidence panel can
// render it without any export-time stripping.
func (e *toolExecutor) recordSectionFacts(ctx context.Context, section string, factIDs []string) error {
	if len(factIDs) == 0 {
		return nil
	}
	conv, err := e.store.GetConversationForOwner(ctx, e.conversationID, e.actorID)
	if err != nil {
		return err
	}
	lm, extra := loadLedger(conv.Meta)
	if lm.SectionFacts == nil {
		lm.SectionFacts = map[string][]string{}
	}
	lm.SectionFacts[section] = factIDs
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return err
	}
	return e.store.UpdateConversationMeta(ctx, e.actorID, e.conversationID, metaJSON)
}

func (e *toolExecutor) advanceStage(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Next string `json:"next"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("agent: invalid advance_stage arguments: %w", err)
	}
	if nextStage(e.currentStage) != args.Next {
		return "", fmt.Errorf("agent: cannot advance from %q to %q", e.currentStage, args.Next)
	}
	if err := e.checkStageGate(ctx, args.Next); err != nil {
		return "", err
	}
	if err := e.store.UpdateConversationStage(ctx, e.actorID, e.conversationID, args.Next); err != nil {
		return "", err
	}
	e.currentStage = args.Next
	if e.onStageChange != nil {
		e.onStageChange(args.Next)
	}
	return fmt.Sprintf("stage advanced to %q", args.Next), nil
}

// checkStageGate is the mechanical, server-side gate (plan §4.2) — it
// does not trust the model's say-so that clarify/draft is "done".
func (e *toolExecutor) checkStageGate(ctx context.Context, next string) error {
	switch next {
	case StageDraft:
		lm, err := currentLedger(ctx, e.store, e.actorID, e.conversationID)
		if err != nil {
			return err
		}
		attested := attestedKinds(lm.Facts)
		var missing []string
		for _, k := range []string{"problem", "segment", "alternative", "whynow"} {
			if !attested[k] {
				missing = append(missing, k)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("agent: cannot draft yet — no attested fact recorded for: %s. Ask the user and call record_fact, or if they want to skip, record a decision fact saying so", strings.Join(missing, ", "))
		}
		for _, f := range lm.Facts {
			if f.Kind == "segment" && f.Status == FactStatusAttested && segmentFailsEveryoneLint(f.Text) {
				return fmt.Errorf("agent: the segment fact %q reads like \"everyone\" — press for a narrower, named audience before drafting", f.Text)
			}
		}

	case StageCritique:
		prd, err := e.store.GetPrdForMember(ctx, e.prdID, e.actorID)
		if err != nil {
			return err
		}
		var empty []string
		for _, k := range store.PrdSectionKeys {
			if strings.TrimSpace(prd.Sections[k]) == "" {
				empty = append(empty, k)
			}
		}
		if len(empty) > 0 {
			return fmt.Errorf("agent: cannot critique yet — these sections are still empty: %s", strings.Join(empty, ", "))
		}

	case StageFinalize:
		conv, err := e.store.GetConversationForOwner(ctx, e.conversationID, e.actorID)
		if err != nil {
			return err
		}
		_, extra := loadLedger(conv.Meta)
		cs := loadCriticState(extra)
		if cs.RoundCount == 0 && cs.Skipped == "" {
			return fmt.Errorf("agent: cannot finalize yet — no critic pass has run against this PRD")
		}
		if hasOpenBlockingFindings(loadCriticFindings(extra)) {
			return fmt.Errorf("agent: cannot finalize yet — there are unresolved blocking findings; call resolve_finding on each first")
		}
	}
	return nil
}

func (e *toolExecutor) resolveFinding(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("agent: invalid resolve_finding arguments: %w", err)
	}
	if args.Status != FindingResolved && args.Status != FindingOverridden {
		return "", fmt.Errorf("agent: status must be %q or %q", FindingResolved, FindingOverridden)
	}

	conv, err := e.store.GetConversationForOwner(ctx, e.conversationID, e.actorID)
	if err != nil {
		return "", err
	}
	lm, extra := loadLedger(conv.Meta)
	findings := loadCriticFindings(extra)
	found := false
	for i := range findings {
		if findings[i].ID == args.ID {
			findings[i].Status = args.Status
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("agent: unknown finding id %q", args.ID)
	}
	saveCriticFindings(extra, findings)
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return "", err
	}
	if err := e.store.UpdateConversationMeta(ctx, e.actorID, e.conversationID, metaJSON); err != nil {
		return "", err
	}
	return fmt.Sprintf("finding %s marked %s", args.ID, args.Status), nil
}
