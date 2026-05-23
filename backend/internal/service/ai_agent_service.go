package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"anttrader/internal/repository"
)

// AIAgentDefinition 是 connect 层与 service 层之间的纯数据结构。
// 自 060 起 Agent 与 ai_config_profiles 解耦：模型绑定通过
// (ProviderID, ModelOverride) 直接定位 system_ai_configs 行。
type AIAgentDefinition struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	AgentKey      string
	Type          string
	Name          string
	Identity      string
	InputHint     string
	Enabled       bool
	Position      int32
	ProviderID    string
	ModelOverride string
}

// AIAgentService 提供「列出当前用户的 Agent」「整组替换 Agent」两类操作；
// 并在用户首次访问时自动 seed 8 个默认 Agent。
type AIAgentService struct {
	repo *repository.AIAgentDefinitionRepository
}

func NewAIAgentService(repo *repository.AIAgentDefinitionRepository) *AIAgentService {
	return &AIAgentService{repo: repo}
}

// allowedAgentTypes 列出 ai_agent_definitions.type 的合法取值。
var allowedAgentTypes = map[string]struct{}{
	"style":     {},
	"signals":   {},
	"risk":      {},
	"macro":     {},
	"sentiment": {},
	"portfolio": {},
	"execution": {},
	"code":      {},
	"custom":    {},
}

// defaultAgentSeedOrder 决定首次 seed 时的固定顺序与 agent_key。
var defaultAgentSeedOrder = []struct {
	Key  string
	Type string
}{
	{"default-style", "style"},
	{"default-signals", "signals"},
	{"default-risk", "risk"},
	{"default-macro", "macro"},
	{"default-sentiment", "sentiment"},
	{"default-portfolio", "portfolio"},
	{"default-execution", "execution"},
	{"default-code", "code"},
}

func normalizeAgentType(t string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(t))
	if _, ok := allowedAgentTypes[v]; !ok {
		return "", errors.New("invalid agent type")
	}
	return v, nil
}

// ListAgents 返回该用户的 Agent。库里为空时自动 seed 8 个默认 Agent 并返回。
// locale 用于挑选默认 Agent 的本地化名称 / identity；空字符串走 en fallback。
func (s *AIAgentService) ListAgents(ctx context.Context, userID uuid.UUID, locale string) ([]*AIAgentDefinition, error) {
	if s.repo == nil {
		return []*AIAgentDefinition{}, nil
	}
	rows, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		// 后端 seed：唯一的默认值写入路径，避免前端竞态覆盖。
		seeded, serr := s.seedDefaults(ctx, userID, locale)
		if serr != nil {
			return nil, serr
		}
		return seeded, nil
	}
	return rowsToDefs(rows), nil
}

// SetAgents 用 agents 整体覆盖该用户的 Agent 列表。
func (s *AIAgentService) SetAgents(ctx context.Context, userID uuid.UUID, agents []*AIAgentDefinition) ([]*AIAgentDefinition, error) {
	if s.repo == nil {
		return []*AIAgentDefinition{}, nil
	}
	rows, err := defsToRows(userID, agents)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceByUser(ctx, userID, rows); err != nil {
		return nil, err
	}
	saved, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return rowsToDefs(saved), nil
}

// seedDefaults 写入 8 个默认 Agent。Name / Identity 取自 agent_i18n。
func (s *AIAgentService) seedDefaults(ctx context.Context, userID uuid.UUID, locale string) ([]*AIAgentDefinition, error) {
	rows := make([]*repository.AIAgentDefinitionRow, 0, len(defaultAgentSeedOrder))
	for i, d := range defaultAgentSeedOrder {
		entry, _ := localizedAgentFor(d.Type, locale)
		rows = append(rows, &repository.AIAgentDefinitionRow{
			ID:       uuid.New(),
			UserID:   userID,
			AgentKey: d.Key,
			Type:     d.Type,
			Name:     entry.Name,
			Identity: entry.Identity,
			Enabled:  true,
			Position: int32(i),
		})
	}
	if err := s.repo.ReplaceByUser(ctx, userID, rows); err != nil {
		return nil, err
	}
	return rowsToDefs(rows), nil
}

// defsToRows 校验入参 + 转 row。
func defsToRows(userID uuid.UUID, agents []*AIAgentDefinition) ([]*repository.AIAgentDefinitionRow, error) {
	if len(agents) == 0 {
		return []*repository.AIAgentDefinitionRow{}, nil
	}
	if len(agents) > 64 {
		return nil, errors.New("too many agents for a single user")
	}
	rows := make([]*repository.AIAgentDefinitionRow, 0, len(agents))
	for idx, a := range agents {
		if a == nil {
			continue
		}
		typeNorm, err := normalizeAgentType(a.Type)
		if err != nil {
			return nil, err
		}
		key := strings.TrimSpace(a.AgentKey)
		if key == "" {
			key = typeNorm + "-" + strconv.Itoa(idx)
		}
		name := strings.TrimSpace(a.Name)
		if name == "" {
			return nil, errors.New("agent name is required")
		}
		if r := []rune(name); len(r) > 100 {
			name = string(r[:100])
		}
		identity := strings.TrimSpace(a.Identity)
		if identity == "" {
			return nil, errors.New("agent identity is required")
		}
		if r := []rune(identity); len(r) > 8000 {
			identity = string(r[:8000])
		}
		hint := strings.TrimSpace(a.InputHint)
		if r := []rune(hint); len(r) > 2000 {
			hint = string(r[:2000])
		}
		pos := a.Position
		if pos < 0 {
			pos = int32(idx)
		}
		rows = append(rows, &repository.AIAgentDefinitionRow{
			ID:            a.ID,
			UserID:        userID,
			AgentKey:      key,
			Type:          typeNorm,
			Name:          name,
			Identity:      identity,
			InputHint:     hint,
			Enabled:       a.Enabled,
			Position:      pos,
			ProviderID:    strings.TrimSpace(a.ProviderID),
			ModelOverride: strings.TrimSpace(a.ModelOverride),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Position == rows[j].Position {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Position < rows[j].Position
	})
	return rows, nil
}

func rowsToDefs(rows []*repository.AIAgentDefinitionRow) []*AIAgentDefinition {
	out := make([]*AIAgentDefinition, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		out = append(out, &AIAgentDefinition{
			ID:            r.ID,
			UserID:        r.UserID,
			AgentKey:      r.AgentKey,
			Type:          r.Type,
			Name:          r.Name,
			Identity:      r.Identity,
			InputHint:     r.InputHint,
			Enabled:       r.Enabled,
			Position:      r.Position,
			ProviderID:    r.ProviderID,
			ModelOverride: r.ModelOverride,
		})
	}
	return out
}
