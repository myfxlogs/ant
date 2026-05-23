package service

// debate_v2_helpers.go
// V1 debate_service.go 已删除（Phase 1 重构），但其中三个共享的小工具仍被
// V2 服务用到，挪到这里集中管理，避免 V2 主文件继续膨胀。

// validTurnTypes 列出 debate_turns 表 type 列的允许取值。RPC 入口校验用。
var validTurnTypes = map[string]struct{}{
	"user_intent":      {},
	"clarify_question": {},
	"clarify_answer":   {},
	"intent_spec":      {},
	"agent_opinion":    {},
	"user_feedback":    {},
	"consensus":        {},
	"code_proposal":    {},
	"system_note":      {},
}

// sanitizeAgents 去重 + 去空。
func sanitizeAgents(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, a := range in {
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

// strPtr 用于把字符串包成 *string，便于配合 nullable 列。
func strPtr(v string) *string { return &v }
