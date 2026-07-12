package agent

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// HookEvent identifies a lifecycle hook point (ADR-0025 §8).
type HookEvent string

const (
	HookPreStrategySubmit HookEvent = "pre_strategy_submit"
	HookPostBacktest      HookEvent = "post_backtest"
	HookPreLiveDeploy     HookEvent = "pre_live_deploy"
	HookPostStrategyGen   HookEvent = "post_strategy_generation"
)

// HookContext carries data to lifecycle hooks.
type HookContext struct {
	Event          HookEvent
	UserID         uuid.UUID
	StrategyID     string
	Source         string
	BacktestResult *antv1.AgentBacktestResult
	Profile        *antv1.StrategyProfile
	Analysis       *antv1.BacktestAnalysis
	Error error
}

// HookType is the execution type of a hook (ADR-0025 §8).
type HookType string

const (
	HookTypeCommand  HookType = "command"  // shell command, stdin=JSON, exit code 2 = abort
	HookTypeWebhook  HookType = "webhook"  // HTTP POST, {"allow": false} = abort
)

// HookConfig describes a registered hook with its type and execution params.
type HookConfig struct {
	Type     HookType
	Event    HookEvent
	Command  string        // for HookTypeCommand
	WebhookURL string      // for HookTypeWebhook
	Timeout  time.Duration // default 10s
}

// HookResult is the outcome of a hook execution.
type HookResult struct {
	Abort   bool   // if true, abort the operation
	Reason  string // abort reason
}

// HookHandler is a function called at a lifecycle hook point.
type HookHandler func(ctx context.Context, hc *HookContext) HookResult

// HookEngine manages lifecycle hooks (ADR-0025 §8).
// Supports three hook types: internal (Go function), command (shell), webhook (HTTP POST).
type HookEngine struct {
	handlers map[HookEvent][]HookHandler   // internal handlers
	configs  map[HookEvent][]HookConfig     // command/webhook configs
	log      *zap.Logger
}

// NewHookEngine creates a hook engine.
func NewHookEngine(log *zap.Logger) *HookEngine {
	return &HookEngine{
		handlers: make(map[HookEvent][]HookHandler),
		configs:  make(map[HookEvent][]HookConfig),
		log:      log,
	}
}

// RegisterConfig adds a command or webhook hook for a lifecycle event.
func (e *HookEngine) RegisterConfig(cfg HookConfig) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	e.configs[cfg.Event] = append(e.configs[cfg.Event], cfg)
}

// Fire executes all handlers and configs for an event. If any returns Abort=true,
// subsequent handlers are skipped and the abort result is returned.
func (e *HookEngine) Fire(ctx context.Context, hc *HookContext) HookResult {
	// Internal handlers first
	for _, h := range e.handlers[hc.Event] {
		result := h(ctx, hc)
		if result.Abort {
			e.log.Info("hook: operation aborted",
				zap.String("event", string(hc.Event)),
				zap.String("reason", result.Reason))
			return result
		}
	}

	// Then command/webhook configs
	for _, cfg := range e.configs[hc.Event] {
		result := e.execConfig(ctx, cfg, hc)
		if result.Abort {
			e.log.Info("hook: operation aborted",
				zap.String("event", string(hc.Event)),
				zap.String("type", string(cfg.Type)),
				zap.String("reason", result.Reason))
			return result
		}
	}

	return HookResult{}
}

// execConfig executes a single command or webhook hook.
func (e *HookEngine) execConfig(ctx context.Context, cfg HookConfig, hc *HookContext) HookResult {
	switch cfg.Type {
	case HookTypeCommand:
		return e.execCommand(ctx, cfg, hc)
	case HookTypeWebhook:
		return e.execWebhook(ctx, cfg, hc)
	default:
		return HookResult{}
	}
}

// execCommand runs a shell command with hook context as stdin JSON.
// Exit code 2 = abort (ADR-0025 §8).
func (e *HookEngine) execCommand(ctx context.Context, cfg HookConfig, hc *HookContext) HookResult {
	hookCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(hookCtx, "sh", "-c", cfg.Command)
	stdin := buildHookJSON(hc)
	cmd.Stdin = bytes.NewReader([]byte(stdin))

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 2 {
				return HookResult{Abort: true, Reason: string(output)}
			}
		}
		e.log.Warn("hook: command failed",
			zap.String("event", string(hc.Event)),
			zap.String("command", cfg.Command),
			zap.Error(err),
			zap.String("output", string(output)))
		return HookResult{}
	}
	return HookResult{}
}

// execWebhook sends an HTTP POST with hook context as JSON body.
// Response {"allow": false, "reason": "..."} = abort (ADR-0025 §8).
func (e *HookEngine) execWebhook(ctx context.Context, cfg HookConfig, hc *HookContext) HookResult {
	hookCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	body := buildHookJSON(hc)
	req, err := http.NewRequestWithContext(hookCtx, "POST", cfg.WebhookURL, bytes.NewReader([]byte(body)))
	if err != nil {
		e.log.Warn("hook: webhook request creation failed", zap.Error(err))
		return HookResult{}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.log.Warn("hook: webhook call failed", zap.Error(err))
		return HookResult{}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e.log.Warn("hook: webhook returned error status", zap.Int("status", resp.StatusCode))
		return HookResult{Abort: true, Reason: fmt.Sprintf("webhook returned %d", resp.StatusCode)}
	}

	// Parse response: {"allow": false, "reason": "..."}
	// Simple check — avoid json.Unmarshal per project rules.
	respBody := make([]byte, 1024)
	n, _ := resp.Body.Read(respBody)
	respStr := string(respBody[:n])
	if strings.Contains(respStr, `"allow":false`) || strings.Contains(respStr, `"allow": false`) {
		reason := extractJSONValue(respStr, "reason")
		if reason == "" {
			reason = "webhook denied"
		}
		return HookResult{Abort: true, Reason: reason}
	}
	return HookResult{}
}

// HasHandlers returns true if any handlers or configs are registered for the event.
func (e *HookEngine) HasHandlers(event HookEvent) bool {
	return len(e.handlers[event]) > 0 || len(e.configs[event]) > 0
}

// buildHookJSON serializes hook context as a simple JSON string for command/webhook hooks.
func buildHookJSON(hc *HookContext) string {
	return fmt.Sprintf(`{"event":"%s","user_id":"%s","strategy_id":"%s","source":"%s"}`,
		hc.Event, hc.UserID, hc.StrategyID, escapeJSON(hc.Source))
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func extractJSONValue(json, key string) string {
	search := fmt.Sprintf(`"%s":"`, key)
	idx := strings.Index(json, search)
	if idx < 0 {
		return ""
	}
	start := idx + len(search)
	end := strings.Index(json[start:], `"`)
	if end < 0 {
		return ""
	}
	return json[start : start+end]
}
