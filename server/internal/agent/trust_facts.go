package agent

import (
	"fmt"
	"unicode"
)

// The numbers below are tunables I chose, not measured limits: change them
// together with the pinning test (trust_facts_bounds_test.go).
const (
	// minAttestedQuoteRunes is the shortest quote that may attest a fact
	// (rv4-AIDEV-07). It counts the normalized quote (spaces and punctuation
	// dropped) in Latin-letter equivalents: a Han, kana or hangul rune counts
	// twice, as it carries about two letters of content. 12 turns a bare "we want
	// it now" (11) away and lets a clause such as "before the next cohort" in.
	minAttestedQuoteRunes = 12

	// minTextSupport is the least share of a fact's text, as character n-grams,
	// that its quote must also hold. In the 39 hand-written pairs I calibrated on,
	// text about something else shared at most 0.16 and honest restatements 0.27 or
	// more, bar one heavy paraphrase at 0.14, so 0.2 sits in the gap (the figures
	// and the trade-off: the 2026-10-18 rv4 w3-coach-trust changelog; a subset of
	// the pairs: trust_facts_bounds_test.go).
	minTextSupport = 0.2

	// minLongTextSupport is the least share of a text, as the same n-grams, that its
	// quote must hold once the text is long: more than longTextRatio times the quote
	// (rv4-FR-F90). The 0.2 above asks only that the quote be the same subject, so a
	// verbatim quote could carry a much longer invented claim: the review's probe, a
	// 39-letter quote plus a 108-letter invented price decision, scored 0.27 and was
	// stored as attested. 0.6 is the review's own number, not a measurement of mine.
	// Support counts distinct n-grams, so a long text that is mostly the quote said
	// again holds more than 0.6 and still passes: a ceiling of a lexical check.
	minLongTextSupport = 0.6

	// longTextRatio is where "long" starts for the bar above: a text more than this
	// many times as long as its quote, in the same letter equivalents. The review
	// began the bar at the quote's own length (or cut text off at 1.5 times); I begin
	// it at two, because the payload fact that tools_fence_test.go records is 1.6
	// times its quote at a support of 0.49 and an earlier start refuses it. The
	// honest longer texts I wrote are at most 1.4 times and every padded probe 2.2
	// times or more, so two sits between them. A text up to twice its quote can
	// therefore still carry an invented addition as long as the quote itself. Where
	// to begin is the owner's call.
	longTextRatio = 2
)

// checkFactEvidence refuses a fact its quote cannot stand behind. quoteIsGrounded
// only proves the user said the quote; this asks whether the quote is enough to
// say anything (length), whether it is about the same thing as the text (support)
// and, for a text far longer than the quote, whether the quote holds most of it.
// Support is lexical, as far as a server without a model call can go: it catches
// text about something else and long invented additions, not a short addition or
// text that merely sounds like the quote, and the model can still pass it by
// copying the quote into the text.
func checkFactEvidence(text, quote string) error {
	nq := normalizeForMatch(quote)
	if quoteWeight(nq) < minAttestedQuoteRunes {
		return fmt.Errorf("agent: the quote is too short to attest a fact — quote the user's fuller words, a whole clause rather than a word or two")
	}
	nt := normalizeForMatch(text)
	support := textSupport(nt, nq)
	if support < minTextSupport {
		return fmt.Errorf("agent: the fact's text is not supported by its quote — restate the fact in the words of the quote, and don't add claims the user didn't make")
	}
	if quoteWeight(nt) > longTextRatio*quoteWeight(nq) && support < minLongTextSupport {
		return fmt.Errorf("agent: the fact's text is not supported by its quote — it is more than %d times as long as the quote and most of it is not in the quote; cut it back to what the quote says, and record anything else the user said as its own fact with its own quote", longTextRatio)
	}
	return nil
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

// quoteWeight is the length of a normalized string in Latin-letter equivalents.
func quoteWeight(nq string) int {
	w := 0
	for _, r := range nq {
		w++
		if isCJK(r) {
			w++
		}
	}
	return w
}

// textSupport is the share of the text's distinct character n-grams that the
// quote also holds: trigrams for alphabetic text, bigrams once the text has any
// CJK rune (a CJK bigram is about as distinctive as a Latin trigram; Latin
// bigrams are not, "the" and "he" overlap almost everything). Both arguments are
// normalizeForMatch output. Text too short to have an n-gram has nothing to check.
func textSupport(text, quote string) float64 {
	n := 3
	for _, r := range text {
		if isCJK(r) {
			n = 2
			break
		}
	}
	tg := runeGrams(text, n)
	if len(tg) == 0 {
		return 1
	}
	qg := runeGrams(quote, n)
	hits := 0
	for g := range tg {
		if qg[g] {
			hits++
		}
	}
	return float64(hits) / float64(len(tg))
}

func runeGrams(s string, n int) map[string]bool {
	r := []rune(s)
	out := make(map[string]bool, len(r))
	for i := 0; i+n <= len(r); i++ {
		out[string(r[i:i+n])] = true
	}
	return out
}
