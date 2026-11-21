package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
	"github.com/google/uuid"
)

// maybeRunCritic implements the fresh-context critic pass (plan §4.4):
// it runs at critique-stage entry and after one applied revision round,
// bounded at maxCriticRounds, and never sees the chat history — only the
// PRD and the fact ledger. Under queue pressure (waited) it degrades
// instead of running, recording a critic-skipped marker so the finalize
// gate isn't hostage to load (plan §5.1). The critic inherits this
// turn's already-held concurrency permit; it never acquires its own.
func (c *Coach) maybeRunCritic(ctx context.Context, actorID, prdID, conversationID, runID uuid.UUID, stage, reply string, sectionUpdated, waited bool, sse *sseWriter) {
	c.captureRankedCritique(ctx, actorID, conversationID, stage, reply, sse)
	if stage != StageCritique {
		return
	}
	conv, err := c.store.GetConversationForOwner(ctx, conversationID, actorID)
	if err != nil {
		return
	}
	lm, extra := loadLedger(conv.Meta)
	cs := loadCriticState(extra)
	if cs.RoundCount >= maxCriticRounds {
		return
	}
	if !(cs.RoundCount == 0 || (cs.RoundCount == 1 && sectionUpdated)) {
		return
	}

	markSkipped := func(reason string) {
		c.recordCriticSkip(ctx, actorID, conversationID, lm, extra, cs, reason, sse)
	}

	if waited {
		markSkipped("contention")
		return
	}

	prd, err := c.store.GetPrdForMember(ctx, prdID, actorID)
	if err != nil {
		return
	}
	var focus []string
	if cs.RoundCount == 1 {
		for _, f := range loadCriticFindings(extra) {
			if f.Status == FindingOpen {
				focus = append(focus, f.Section)
			}
		}
	}
	started := time.Now()
	findings, usage, ok := runCritic(ctx, c.streamer, conv.Locale, prd.Title, prd.Sections, lm, focus)
	var callErr error
	if !ok {
		callErr = fmt.Errorf("critic output failure")
	}
	c.recordTelemetry(ctx, runID, "critic", started, usage, callErr)
	if !ok {
		markSkipped("critic-output-failure")
		return
	}

	merged := append(loadCriticFindings(extra), findings...)
	saveCriticFindings(extra, merged)
	cs.RoundCount++
	cs.Skipped = "" // a round that ran supersedes an earlier skip: the card reads this marker
	saveCriticState(extra, cs)
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		c.log.Error("save critic findings", "error", err)
		return
	}
	if err := c.store.UpdateConversationMeta(ctx, actorID, conversationID, metaJSON); err != nil {
		c.log.Error("persist critic findings", "error", err)
		return
	}
	sse.criticFindings(findings)
}

// rankedCritiqueFindings is what a critique-stage reply becomes for the
// panel, or nil when the reply isn't the ranked list. A later turn that
// talks through one finding therefore doesn't wipe the list.
func rankedCritiqueFindings(stage, reply string) []CriticFinding {
	if stage != StageCritique {
		return nil
	}
	findings := parseRankedCritique(reply)
	if len(findings) == 0 {
		return nil
	}
	return findings
}

// captureRankedCritique turns a critique-stage reply into the findings the
// panel renders: it merges them into what's stored (a restatement may not
// erase or reset stored findings — AIDEV-01) and emits the merged list.
func (c *Coach) captureRankedCritique(ctx context.Context, actorID, conversationID uuid.UUID, stage, reply string, sse *sseWriter) {
	findings := rankedCritiqueFindings(stage, reply)
	if findings == nil {
		return
	}
	conv, err := c.store.GetConversationForOwner(ctx, conversationID, actorID)
	if err != nil {
		return
	}
	lm, extra := loadLedger(conv.Meta)
	merged := mergeRankedCritique(loadCriticFindings(extra), findings)
	saveCriticFindings(extra, merged)
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return
	}
	if err := c.store.UpdateConversationMeta(ctx, actorID, conversationID, metaJSON); err != nil {
		c.log.Error("persist ranked critique", "error", err)
		return
	}
	_ = sse.criticFindings(merged)
}

func hasOpenBlockingFindings(findings []CriticFinding) bool {
	for _, f := range findings {
		if f.Status == FindingOpen && f.Severity == "blocking" {
			return true
		}
	}
	return false
}

// runCritic makes one fresh-context critic call (no chat history — only
// the PRD and the ledger, per plan §4.4) and returns verified findings.
// The prompt language follows the conversation's locale (zh prompts here,
// en variants in prompts_en.go). Output-shape failures (unparseable/empty)
// get one re-prompt retry with the error appended; a second failure
// returns ok=false so the caller can degrade to single-context critique
// rather than fail the turn.
func runCritic(ctx context.Context, streamer llm.ChatStreamer, locale, prdTitle string, sections map[string]string, lm ledgerMeta, focusSections []string) (findings []CriticFinding, usage llm.Usage, ok bool) {
	tmpl, focusPrefix, userDirective, retryFormat := criticSystemPromptTemplate,
		"本轮只需重新评审以下此前被标记过的章节：",
		"请按系统提示中的标准独立评审这份 PRD，并通过 report_findings 工具报告全部发现。",
		"上一次调用失败：%s。请重新输出，严格符合 report_findings 的参数格式。"
	if locale == localeEN {
		tmpl, focusPrefix, userDirective, retryFormat = criticSystemPromptTemplateEN,
			criticFocusPrefixEN, criticUserDirectiveEN, criticRetryPrefixFormat
	}
	focus := ""
	if len(focusSections) > 0 {
		// The names come from stored findings whose section is model output, so
		// they ride in a fence like the rest of the PRD-derived text.
		focus = "\n\n" + focusPrefix + fenceData(strings.Join(focusSections, ", "))
	}
	system := fmt.Sprintf(tmpl, focus, dataFenceClause,
		fenceData(prdTitle), fenceData(formatSectionsForPrompt(sections)), fenceData(formatLedgerForPrompt(lm)))
	tool := reportFindingsTool()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		sys := system
		if lastErr != nil {
			sys += fmt.Sprintf("\n\n"+retryFormat, lastErr.Error())
		}
		// AIDEV-09: the Messages API rejects an empty "messages" array with
		// a 400 on every call — the PRD itself rides in the system prompt,
		// so this turn only needs to direct the forced tool call.
		user := userDirective
		turn, err := streamer.StreamChatForcingTool(ctx, sys, []llm.Message{{Role: llm.RoleUser, Content: user}}, tool, nil)
		usage = turn.Usage
		if err != nil {
			lastErr = err
			continue
		}
		if turn.ToolCall == nil || turn.ToolCall.Name != tool.Name {
			lastErr = fmt.Errorf("model did not call report_findings")
			continue
		}
		parsed, perr := parseCriticFindings(turn.ToolCall.Arguments, sections)
		if perr != nil {
			lastErr = perr
			continue
		}
		return parsed, usage, true
	}
	return nil, usage, false
}
