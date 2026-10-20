package store

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// requireDoneEvidenceOrReason enforces the completion rule on the one
// transition that marks work finished: in_progress → done needs at least
// one linked evidence row, or a one-sentence reason stored on this write.
// Every other transition is left alone, and a reason sent beside one of
// those is not written. A reason already stored from an earlier completion
// does not excuse a later one.
func requireDoneEvidenceOrReason(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID, from string, f *RoadmapNodeFields) error {
	if f.Status == nil || from == *f.Status || from != "in_progress" || *f.Status != "done" {
		f.CompletionReason = nil
		return nil
	}
	reason := ""
	if f.CompletionReason != nil {
		reason = strings.TrimSpace(*f.CompletionReason)
	}
	var evidenced bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM insideout.roadmap_evidence WHERE node_id = $1)`,
		nodeID).Scan(&evidenced); err != nil {
		return err
	}
	if evidenced && reason == "" {
		f.CompletionReason = nil
		return nil
	}
	if !oneSentence(reason) {
		return fmt.Errorf("%w: moving in_progress to done needs linked evidence or a one-sentence completion reason", ErrValidation)
	}
	f.CompletionReason = &reason
	return nil
}

// oneSentence is one clause that ends in a single sentence terminator.
// A Latin fragment ("Done.") is not enough; two sentences are not one.
// A decimal point inside a token (v1.2) is not a second sentence.
func oneSentence(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	rs := []rune(s)
	if !sentenceTerminator(rs[len(rs)-1]) {
		return false
	}
	n := 0
	for i, r := range rs {
		if !sentenceTerminator(r) {
			continue
		}
		if i == len(rs)-1 || unicode.IsSpace(rs[i+1]) {
			n++
		}
	}
	if n != 1 {
		return false
	}
	body := strings.TrimSpace(string(rs[:len(rs)-1]))
	if body == "" {
		return false
	}
	if len(strings.Fields(body)) >= 2 {
		return true
	}
	han := 0
	for _, r := range body {
		if unicode.Is(unicode.Han, r) {
			han++
		}
	}
	return han >= 4 && utf8.RuneCountInString(body) == han
}

func sentenceTerminator(r rune) bool {
	switch r {
	case '.', '!', '?', '。', '！', '？':
		return true
	default:
		return false
	}
}
