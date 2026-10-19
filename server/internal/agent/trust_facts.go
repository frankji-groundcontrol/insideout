package agent

import (
	"fmt"
	"unicode"
)

// Both numbers below are tunables I chose, not measured limits: change them
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
)

// checkFactEvidence refuses a fact its quote cannot stand behind. quoteIsGrounded
// only proves the user said the quote; this asks whether the quote is enough to
// say anything (length) and whether it is about the same thing as the text
// (support). Support is lexical, as far as a server without a model call can go:
// it catches text about something else, not text that merely sounds like the
// quote, and the model can still pass it by copying the quote into the text.
func checkFactEvidence(text, quote string) error {
	nq := normalizeForMatch(quote)
	if quoteWeight(nq) < minAttestedQuoteRunes {
		return fmt.Errorf("agent: the quote is too short to attest a fact — quote the user's fuller words, a whole clause rather than a word or two")
	}
	if textSupport(normalizeForMatch(text), nq) < minTextSupport {
		return fmt.Errorf("agent: the fact's text is not supported by its quote — restate the fact in the words of the quote, and don't add claims the user didn't make")
	}
	return nil
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

// quoteWeight is the length of a normalized quote in Latin-letter equivalents.
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
