package ai

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	internalai "alphaforge/internal/ai"
	systemai "alphaforge/internal/service/systemai"
)

// ── Diagnose: analysis only, no code generation ──

func (s *StrategyPlanServer) Diagnose(
	ctx context.Context,
	req *connect.Request[antv1.DiagnoseRequest],
	stream *connect.ServerStream[antv1.AnalyzePlanChunk],
) error {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return err
	}
	m := req.Msg
	lang := LangFromAccept(req.Header().Get("Accept-Language"))

	sysPrompt := internalai.AgentPrompt(lang) + "\n\n## 当前任务：诊断问题，提出建议\n分析用户的反馈和回测数据，给出具体的修改建议。每行一个建议，用 - 开头。不要生成代码，只输出诊断分析和建议。"
	userPrompt := fmt.Sprintf(
		"## 执行计划\n%s\n\n## 当前代码\n```go\n%s\n```\n\n## 回测数据\n%s\n\n## 用户反馈\n%s\n\n请诊断问题并给出修改建议。每行一个建议，用 - 开头。",
		m.Plan, m.CurrentCode, formatBacktestMetrics(m.BacktestMetrics), m.FeedbackMessage,
	)

	var fullBuf strings.Builder
	err = s.systemSvc.ChatCompletionStream(ctx, userID,
		[]systemai.ChatMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: userPrompt}},
		func(chunk systemai.ChatStreamChunk) error {
			fullBuf.WriteString(chunk.Content)
			send := &antv1.AnalyzePlanChunk{Phase: "analyzing", Delta: chunk.Content}
			if chunk.Done {
				send.Phase = "plan_ready"
				send.Plan = fullBuf.String()
			}
			return stream.Send(send)
		})
	if err != nil {
		return stream.Send(&antv1.AnalyzePlanChunk{Phase: "error", Error: systemai.FriendlyError(err)})
	}

	s.persistDiagnose(ctx, userID, m.ConversationId, m.FeedbackMessage, fullBuf.String())
	return nil
}

// formatBacktestMetrics formats a BacktestMetricsMsg into a human-readable string for LLM prompts.
func formatBacktestMetrics(m *antv1.BacktestMetricsMsg) string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("total_return=%s sharpe=%s max_drawdown=%s win_rate=%s profit_factor=%s trades=%d",
		m.TotalReturn, m.SharpeRatio, m.MaxDrawdown, m.WinRate, m.ProfitFactor, m.TotalTrades)
}
