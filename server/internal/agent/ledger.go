package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// Fact is one user-attested entry in a conversation's evidence ledger —
// the anti-fabrication backbone (plan §4.1). Every PRD claim should trace
// to one of these, or be explicitly marked [ASSUMPTION].
type Fact struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Text   string `json:"text"`
	Quote  string `json:"quote"`
	Status string `json:"status"` // attested | assumed | needs-validation
}

const (
	FactStatusAttested = "attested"
	FactStatusAssumed  = "assumed"

	maxFactsPerKind = 20
	maxQuoteLen     = 300
)

var factKinds = map[string]bool{
	"problem": true, "segment": true, "alternative": true, "whynow": true,
	"goal": true, "constraint": true, "evidence": true, "decision": true,
}

// ledgerMeta is the shape stored in agent_conversations.meta (jsonb,
// pre-existing column, no migration). Unknown fields in the raw meta
// (e.g. H3's critic_findings) are preserved via rawMeta round-tripping in
// loadLedger/saveLedger below, not modeled here.
type ledgerMeta struct {
	Facts        []Fact              `json:"facts,omitempty"`
	SectionFacts map[string][]string `json:"sectionFacts,omitempty"`
}

func loadLedger(raw json.RawMessage) (ledgerMeta, map[string]json.RawMessage) {
	var extra map[string]json.RawMessage
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &extra)
	}
	var lm ledgerMeta
	if v, ok := extra["facts"]; ok {
		_ = json.Unmarshal(v, &lm.Facts)
		delete(extra, "facts")
	}
	if v, ok := extra["sectionFacts"]; ok {
		_ = json.Unmarshal(v, &lm.SectionFacts)
		delete(extra, "sectionFacts")
	}
	return lm, extra
}

// saveLedger re-marshals the ledger fields alongside whatever other keys
// already lived in meta (extra, from loadLedger) — so the ledger and a
// future H3 critic_findings key can coexist without either clobbering
// the other.
func saveLedger(lm ledgerMeta, extra map[string]json.RawMessage) (json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	for k, v := range extra {
		out[k] = v
	}
	if len(lm.Facts) > 0 {
		b, err := json.Marshal(lm.Facts)
		if err != nil {
			return nil, err
		}
		out["facts"] = b
	}
	if len(lm.SectionFacts) > 0 {
		b, err := json.Marshal(lm.SectionFacts)
		if err != nil {
			return nil, err
		}
		out["sectionFacts"] = b
	}
	return json.Marshal(out)
}

func countByKind(facts []Fact, kind string) int {
	n := 0
	for _, f := range facts {
		if f.Kind == kind {
			n++
		}
	}
	return n
}

func attestedKinds(facts []Fact) map[string]bool {
	out := map[string]bool{}
	for _, f := range facts {
		if f.Status == FactStatusAttested {
			out[f.Kind] = true
		}
	}
	return out
}

// normalizeForMatch strips whitespace/punctuation and lowercases, so
// quote grounding is robust to spacing/punctuation differences without
// needing a language-specific tokenizer — works the same for CJK and
// Latin text since it operates rune-by-rune, not on word boundaries.
func normalizeForMatch(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// quoteIsGrounded reports whether quote plausibly appears in source: an
// exact normalized substring match, or — to tolerate minor paraphrase/
// reordering — at least 70% of the quote's character bigrams also
// appear in source. Character bigrams (not word n-grams) are what make
// this CJK-aware: Chinese has no whitespace word boundaries, so a
// word-tokenized fuzzy matcher would never work here.
func quoteIsGrounded(quote, source string) bool {
	nq := normalizeForMatch(quote)
	if nq == "" {
		return false
	}
	ns := normalizeForMatch(source)
	if strings.Contains(ns, nq) {
		return true
	}
	return bigramOverlap(nq, ns) >= 0.7
}

func bigramOverlap(quote, source string) float64 {
	qg := bigrams(quote)
	if len(qg) == 0 {
		return 0
	}
	sg := bigrams(source)
	hits := 0
	for g := range qg {
		if sg[g] {
			hits++
		}
	}
	return float64(hits) / float64(len(qg))
}

func bigrams(s string) map[string]bool {
	r := []rune(s)
	out := make(map[string]bool, len(r))
	if len(r) < 2 {
		if len(r) == 1 {
			out[string(r)] = true
		}
		return out
	}
	for i := 0; i+1 < len(r); i++ {
		out[string(r[i:i+2])] = true
	}
	return out
}

// formatLedgerForPrompt renders the ledger for injection into the system
// prompt — it must survive the last-20-message history window, which is
// why it's re-derived from meta on every call rather than relying on the
// model remembering what it recorded earlier in the conversation.
func formatLedgerForPrompt(lm ledgerMeta) string {
	if len(lm.Facts) == 0 {
		return "(尚无已记录的事实 / no facts recorded yet)"
	}
	var b strings.Builder
	for _, f := range lm.Facts {
		fmt.Fprintf(&b, "- [%s/%s] %s（原话：%s）\n", f.Kind, f.Status, f.Text, f.Quote)
	}
	return b.String()
}

// --- toolExecutor methods: record_fact / mark_assumption ---

func (e *toolExecutor) recordFact(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Kind  string `json:"kind"`
		Text  string `json:"text"`
		Quote string `json:"quote"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("agent: invalid record_fact arguments: %w", err)
	}
	if !factKinds[args.Kind] {
		return "", fmt.Errorf("agent: unknown fact kind %q", args.Kind)
	}
	if strings.TrimSpace(args.Text) == "" || strings.TrimSpace(args.Quote) == "" {
		return "", fmt.Errorf("agent: record_fact requires both text and quote")
	}
	grounded := false
	for _, msg := range e.userMessages {
		if quoteIsGrounded(args.Quote, msg) {
			grounded = true
			break
		}
	}
	if !grounded {
		return "", fmt.Errorf("agent: quote %q was not found in anything the user said in this conversation — quote their actual words, don't paraphrase or invent", args.Quote)
	}

	conv, err := e.store.GetConversationForOwner(ctx, e.conversationID, e.actorID)
	if err != nil {
		return "", err
	}
	lm, extra := loadLedger(conv.Meta)
	if countByKind(lm.Facts, args.Kind) >= maxFactsPerKind {
		return "", fmt.Errorf("agent: already have %d facts of kind %q, the most useful ones — don't add more", maxFactsPerKind, args.Kind)
	}
	quote := args.Quote
	if len([]rune(quote)) > maxQuoteLen {
		quote = string([]rune(quote)[:maxQuoteLen])
	}
	fact := Fact{ID: "f" + uuid.NewString()[:8], Kind: args.Kind, Text: args.Text, Quote: quote, Status: FactStatusAttested}
	lm.Facts = append(lm.Facts, fact)

	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return "", err
	}
	if err := e.store.UpdateConversationMeta(ctx, e.actorID, e.conversationID, metaJSON); err != nil {
		return "", err
	}
	if e.onFactRecorded != nil {
		e.onFactRecorded(fact)
	}
	return fmt.Sprintf("fact %s recorded", fact.ID), nil
}

func (e *toolExecutor) markAssumption(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("agent: invalid mark_assumption arguments: %w", err)
	}
	if strings.TrimSpace(args.Text) == "" {
		return "", fmt.Errorf("agent: mark_assumption requires text")
	}

	conv, err := e.store.GetConversationForOwner(ctx, e.conversationID, e.actorID)
	if err != nil {
		return "", err
	}
	lm, extra := loadLedger(conv.Meta)
	fact := Fact{ID: "a" + uuid.NewString()[:8], Kind: "assumption", Text: args.Text, Status: FactStatusAssumed}
	lm.Facts = append(lm.Facts, fact)

	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return "", err
	}
	if err := e.store.UpdateConversationMeta(ctx, e.actorID, e.conversationID, metaJSON); err != nil {
		return "", err
	}
	if e.onFactRecorded != nil {
		e.onFactRecorded(fact)
	}
	return fmt.Sprintf("assumption %s recorded", fact.ID), nil
}

// currentLedger reads the ledger fresh from the store — used by
// coach.go to inject it into the system prompt and by the stage gates.
func currentLedger(ctx context.Context, st *store.Store, actorID, conversationID uuid.UUID) (ledgerMeta, error) {
	conv, err := st.GetConversationForOwner(ctx, conversationID, actorID)
	if err != nil {
		return ledgerMeta{}, err
	}
	lm, _ := loadLedger(conv.Meta)
	return lm, nil
}
