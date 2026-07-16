// auto_fix.go: automatic code repair triggered by gate evaluation failure.
//
// When a backtest completes and the auto-gate callback detects a gate failure,
// autoFixCode asynchronously calls the LLM with a gate-aware feedback prompt,
// creates a new backtest run with the improved code, and sends notifications.
// The new run will trigger the auto-gate loop again when it completes.

package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	internalai "alphaforge/internal/ai"
	"alphaforge/internal/connect/ai"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/repository"
	systemai "alphaforge/internal/service/systemai"
)

// autoFixCode generates improved strategy code via LLM when a gate evaluation fails,
// creates a new backtest run, and sends a notification. It is designed to be called
// asynchronously (via goroutine) from the auto-gate callback.
func autoFixCode(
	ctx context.Context,
	run *repository.BacktestRun,
	gateResult internalai.PipelineResult,
	aiSvc *systemai.Service,
	backtestRepo *repository.BacktestRunRepository,
	notifSender *notifpubsub.Sender,
	log *zap.Logger,
) {
	if run.StrategyCode == nil || *run.StrategyCode == "" {
		return
	}

	// 1. Build gate-aware feedback prompt.
	builder := internalai.NewStrategyPromptBuilder()
	st := internalai.DetectCodeStrategyType(*run.StrategyCode)
	sysPrompt, userPrompt := builder.BuildFeedbackPrompt(&internalai.FeedbackPromptParams{
		PreviousCode:       *run.StrategyCode,
		FeedbackMessage:    fmt.Sprintf("Gate evaluation failed at: %s. Reason: %s", gateResult.FirstFail, gateResult.Summary),
		FeedbackHints:      "Fix the gate failure by improving the strategy logic. Focus on the specific gate that failed.",
		StrategyType:       st,
		GateFailureReason:  string(gateResult.FirstFail),
		GateFailureDetails: gateResult.Summary,
	})

	// 2. Call LLM to generate improved code.
	msgs := []systemai.ChatMessage{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: userPrompt},
	}
	resp, err := aiSvc.ChatCompletion(ctx, run.UserID, msgs)
	if err != nil {
		log.Warn("auto-fix: LLM call failed",
			zap.String("run_id", run.ID.String()),
			zap.String("gate", string(gateResult.FirstFail)),
			zap.Error(err))
		return
	}

	// 3. Extract code from LLM response.
	code := ai.ExtractCode(resp)
	if code == "" {
		log.Warn("auto-fix: no code extracted from LLM response",
			zap.String("run_id", run.ID.String()))
		return
	}

	// 4. Create a new backtest run with the improved code.
	newRun := &repository.BacktestRun{
		ID:             uuid.New(),
		UserID:         run.UserID,
		Symbol:         run.Symbol,
		Timeframe:      run.Timeframe,
		Mode:           "KLINE_RANGE",
		Status:         "PENDING",
		StrategyCode:   &code,
		InitialCapital: run.InitialCapital,
		Commission:     run.Commission,
		Slippage:       run.Slippage,
		Leverage:       run.Leverage,
		TradeDirection: run.TradeDirection,
		StrictMode:     run.StrictMode,
		FromTs:         run.FromTs,
		ToTs:           run.ToTs,
	}
	if _, err := backtestRepo.Create(ctx, newRun); err != nil {
		log.Warn("auto-fix: create backtest run failed",
			zap.String("original_run_id", run.ID.String()),
			zap.Error(err))
		return
	}

	log.Info("auto-fix: new backtest run created",
		zap.String("original_run_id", run.ID.String()),
		zap.String("new_run_id", newRun.ID.String()),
		zap.String("failed_gate", string(gateResult.FirstFail)))

	// 5. Send notification.
	if notifSender != nil {
		data, _ := structpb.NewStruct(map[string]interface{}{
			"original_run_id": run.ID.String(),
			"new_run_id":      newRun.ID.String(),
			"failed_gate":     string(gateResult.FirstFail),
		})
		_, _ = notifSender.Send(ctx, run.UserID, "auto_fix_started",
			"Auto-Fix: Code Repair Triggered",
			fmt.Sprintf("Gate '%s' failure triggered automatic code repair. A new backtest has started.", gateResult.FirstFail),
			data)
	}
}
