package agent

// mergeRankedCritique folds a restated ranked list into the stored
// findings (AIDEV-01, review 3). The stored list is what the finalize
// gate reads, so a restatement may never shrink it: stored findings with
// no restated line survive (an omitted open blocking finding stays), a
// restated line that matches a stored finding by (section, severity,
// issue) keeps the stored entry — id and recorded status included — and
// only genuinely new lines are added, open. The issue comparison goes
// through normalizeForMatch so a restated line still matches its stored
// original when punctuation or spacing differs; a paraphrase simply
// counts as a new line, which fails safe (the original survives too).
//
// There is deliberately no "full replace on the first ranked reply"
// branch: the independent critic pass can already have stored findings
// before the coach's first ranked reply (it runs on any critique-stage
// turn, and a reply that isn't a ranked list captures nothing), and a
// replace in that window is exactly the erasure this merge exists to
// prevent. On an empty stored list, merge and replace coincide.
func mergeRankedCritique(stored, restated []CriticFinding) []CriticFinding {
	out := append([]CriticFinding(nil), stored...)
	for _, r := range restated {
		matched := false
		for i := range out {
			if out[i].Section == r.Section && out[i].Severity == r.Severity &&
				normalizeForMatch(out[i].Issue) == normalizeForMatch(r.Issue) {
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, r)
		}
	}
	return out
}
