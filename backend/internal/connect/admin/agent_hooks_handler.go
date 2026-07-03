package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/agent"
)

// AgentHooksServer implements AgentHooksService (ADR-0025 §8).
// Manages persistent hook configurations (command/webhook) in the database.
type AgentHooksServer struct {
	pool   *pgxpool.Pool
	hooks  *agent.HookEngine
	log    *zap.Logger
}

var _ antv1c.AgentHooksServiceHandler = (*AgentHooksServer)(nil)

func NewAgentHooksServer(pool *pgxpool.Pool, hooks *agent.HookEngine, log *zap.Logger) *AgentHooksServer {
	return &AgentHooksServer{pool: pool, hooks: hooks, log: log}
}

type hookConfigRow struct {
	ID              string
	Event           string
	Type            string
	Command         string
	WebhookURL      string
	TimeoutSeconds  int
	Enabled         bool
}

func (s *AgentHooksServer) ListHookConfigs(
	ctx context.Context,
	_ *connect.Request[antv1.ListHookConfigsRequest],
) (*connect.Response[antv1.ListHookConfigsResponse], error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, event, type, command, webhook_url, timeout_seconds, enabled
		 FROM agent_hook_configs ORDER BY event, created_at`)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var configs []*antv1.HookConfigEntry
	for rows.Next() {
		var r hookConfigRow
		if err := rows.Scan(&r.ID, &r.Event, &r.Type, &r.Command, &r.WebhookURL, &r.TimeoutSeconds, &r.Enabled); err != nil {
			continue
		}
		configs = append(configs, rowToProto(r))
	}
	return connect.NewResponse(&antv1.ListHookConfigsResponse{Configs: configs}), nil
}

func (s *AgentHooksServer) SetHookConfig(
	ctx context.Context,
	req *connect.Request[antv1.SetHookConfigRequest],
) (*connect.Response[antv1.SetHookConfigResponse], error) {
	if req.Msg.Event == antv1.HookEventType_HOOK_EVENT_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("event is required"))
	}
	if req.Msg.Type == antv1.HookExecutionType_HOOK_EXEC_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("type is required"))
	}

	eventStr := hookEventTypeToString(req.Msg.Event)
	typeStr := hookExecTypeToString(req.Msg.Type)
	timeout := int(req.Msg.TimeoutSeconds)
	if timeout <= 0 {
		timeout = 10
	}

	id := uuid.New().String()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO agent_hook_configs (id, event, type, command, webhook_url, timeout_seconds, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, eventStr, typeStr, req.Msg.Command, req.Msg.WebhookUrl, timeout, req.Msg.Enabled)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.log.Info("admin: hook config set",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("event", eventStr),
		zap.String("type", typeStr),
		zap.String("id", id))

	return connect.NewResponse(&antv1.SetHookConfigResponse{Id: id, Success: true}), nil
}

func (s *AgentHooksServer) DeleteHookConfig(
	ctx context.Context,
	req *connect.Request[antv1.DeleteHookConfigRequest],
) (*connect.Response[antv1.DeleteHookConfigResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM agent_hook_configs WHERE id = $1`, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: hook config deleted",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("id", req.Msg.Id))
	return connect.NewResponse(&antv1.DeleteHookConfigResponse{Success: true}), nil
}

// LoadHookConfigsFromDB loads enabled hook configs from DB and registers them in the HookEngine.
// Called at startup to restore persisted hook configurations.
func LoadHookConfigsFromDB(ctx context.Context, pool *pgxpool.Pool, engine *agent.HookEngine) error {
	if pool == nil || engine == nil {
		return nil
	}
	rows, err := pool.Query(ctx,
		`SELECT event, type, command, webhook_url, timeout_seconds
		 FROM agent_hook_configs WHERE enabled = true`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var r hookConfigRow
		if err := rows.Scan(&r.Event, &r.Type, &r.Command, &r.WebhookURL, &r.TimeoutSeconds); err != nil {
			continue
		}
		engine.RegisterConfig(agent.HookConfig{
			Event:      agent.HookEvent(r.Event),
			Type:       agent.HookType(r.Type),
			Command:    r.Command,
			WebhookURL: r.WebhookURL,
			Timeout:    time.Duration(r.TimeoutSeconds) * time.Second,
		})
	}
	return nil
}

func rowToProto(r hookConfigRow) *antv1.HookConfigEntry {
	return &antv1.HookConfigEntry{
		Id:             r.ID,
		Event:          stringToHookEventType(r.Event),
		Type:           stringToHookExecType(r.Type),
		Command:        r.Command,
		WebhookUrl:     r.WebhookURL,
		TimeoutSeconds: int32(r.TimeoutSeconds),
		Enabled:        r.Enabled,
	}
}

func hookEventTypeToString(e antv1.HookEventType) string {
	switch e {
	case antv1.HookEventType_HOOK_EVENT_PRE_STRATEGY_SUBMIT:
		return "pre_strategy_submit"
	case antv1.HookEventType_HOOK_EVENT_POST_BACKTEST:
		return "post_backtest"
	case antv1.HookEventType_HOOK_EVENT_PRE_LIVE_DEPLOY:
		return "pre_live_deploy"
	case antv1.HookEventType_HOOK_EVENT_DEGRADATION_ALERT:
		return "degradation_alert"
	case antv1.HookEventType_HOOK_EVENT_POST_STRATEGY_GENERATION:
		return "post_strategy_generation"
	default:
		return ""
	}
}

func stringToHookEventType(s string) antv1.HookEventType {
	switch s {
	case "pre_strategy_submit":
		return antv1.HookEventType_HOOK_EVENT_PRE_STRATEGY_SUBMIT
	case "post_backtest":
		return antv1.HookEventType_HOOK_EVENT_POST_BACKTEST
	case "pre_live_deploy":
		return antv1.HookEventType_HOOK_EVENT_PRE_LIVE_DEPLOY
	case "degradation_alert":
		return antv1.HookEventType_HOOK_EVENT_DEGRADATION_ALERT
	case "post_strategy_generation":
		return antv1.HookEventType_HOOK_EVENT_POST_STRATEGY_GENERATION
	default:
		return antv1.HookEventType_HOOK_EVENT_UNSPECIFIED
	}
}

func hookExecTypeToString(t antv1.HookExecutionType) string {
	switch t {
	case antv1.HookExecutionType_HOOK_EXEC_COMMAND:
		return "command"
	case antv1.HookExecutionType_HOOK_EXEC_WEBHOOK:
		return "webhook"
	case antv1.HookExecutionType_HOOK_EXEC_INTERNAL:
		return "internal"
	default:
		return ""
	}
}

func stringToHookExecType(s string) antv1.HookExecutionType {
	switch s {
	case "command":
		return antv1.HookExecutionType_HOOK_EXEC_COMMAND
	case "webhook":
		return antv1.HookExecutionType_HOOK_EXEC_WEBHOOK
	case "internal":
		return antv1.HookExecutionType_HOOK_EXEC_INTERNAL
	default:
		return antv1.HookExecutionType_HOOK_EXEC_UNSPECIFIED
	}
}
