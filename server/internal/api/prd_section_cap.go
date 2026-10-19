package api

import (
	"fmt"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// maxPrdSectionRunes caps one PRD section's text at write (rv4-COMP-04). A save
// used to check only the section key, so a section could carry the whole 1 MiB
// body and every coach turn then repeated it. It counts characters, not bytes,
// like every other field cap (tooLong). agent.maxPromptSectionRunes is the same
// number on the prompt side; api does not import agent, so the two are kept
// equal by hand and each is pinned by a test that spells the number out.
const maxPrdSectionRunes = 10_000

// sectionLengthError is the 400 message for the first section over the cap, or
// "" when every section fits. It walks the canonical key order, so the answer
// does not depend on map order; the caller has already refused unknown keys.
func sectionLengthError(sections map[string]string) string {
	for _, key := range store.PrdSectionKeys {
		if tooLong(sections[key], maxPrdSectionRunes) {
			return fmt.Sprintf("section %s is too long (max %d characters)", key, maxPrdSectionRunes)
		}
	}
	return ""
}
