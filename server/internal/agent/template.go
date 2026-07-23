package agent

import (
	"context"
	"fmt"
)

// templateStreamer is used when no AI provider is configured
// (ANTHROPIC_AUTH_TOKEN unset) — offline dev mode. It never calls a network and
// never fails; the run still records status 'succeeded', matching the old
// edge function's graceful-degradation behavior (see
// docs/plans/2026-07-20-go-rewrite/02-backend-go.md §1).
type templateStreamer struct{}

// NewTemplateStreamer returns the offline-dev-mode ChatStreamer.
func NewTemplateStreamer() ChatStreamer { return templateStreamer{} }

func (templateStreamer) StreamChat(_ context.Context, _ string, msgs []Message, _ []Tool, onDelta func(string)) (Turn, error) {
	var lastUser string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleUser {
			lastUser = msgs[i].Content
			break
		}
	}
	reply := fmt.Sprintf("收到你的消息：%s\n\n已启用模板回复（尚未配置 AI 凭据）。请在环境变量中配置 ANTHROPIC_BASE_URL 和 ANTHROPIC_AUTH_TOKEN。", lastUser)
	if onDelta != nil {
		onDelta(reply)
	}
	return Turn{Text: reply}, nil
}

// StreamChatForcingTool returns an empty tool call with no arguments —
// offline dev mode has no model to critique with, so the critic pass
// degrades to "no findings" rather than erroring.
func (templateStreamer) StreamChatForcingTool(_ context.Context, _ string, _ []Message, tool Tool, _ func(string)) (Turn, error) {
	return Turn{ToolCall: &ToolCallRequest{ID: "template", Name: tool.Name, Arguments: "{}"}}, nil
}

func (templateStreamer) Model() string { return "template" }
