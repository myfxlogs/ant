package marketplace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/pglisten"
)

// BatchGenerator manages the auto-generation task queue.
// Producer: EnqueueBatch creates tasks for symbol×timeframe×type combinations.
// Consumer: PG NOTIFY-driven worker picks up pending tasks and runs generation.
type BatchGenerator struct {
	pg       *pgxpool.Pool
	log      *zap.Logger
	gen      BatchAgentGenerator
	pgListen *pglisten.Listener
	quality  QualityValidator
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// BatchAgentGenerator is the interface for AI generation within batch context.
type BatchAgentGenerator interface {
	Generate(ctx context.Context, userID uuid.UUID, msg *antv1.AgentGenerateStrategyRequest, stream func(*antv1.AgentGenerateStrategyChunk) error) error
}

// TaskPublisher is the interface for publishing approved batch tasks.
type TaskPublisher interface {
	Publish(ctx context.Context, params PublishParams) (string, error)
}

// QualityValidator is the interface for validating backtest quality.
type QualityValidator interface {
	ValidateBacktestQuality(ctx context.Context, snapshotProto []byte, strategyID string) ([]QualityViolation, error)
}

// AutoGenTask represents a row in auto_generation_tasks.
type AutoGenTask struct {
	ID             uuid.UUID
	Symbol         string
	Timeframe      string
	StrategyType   string
	RiskLevel      string
	Status         string
	StrategyID     *uuid.UUID
	QualityPassed  *bool
	ErrorMessage   *string
	SourceCode     string
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

const notifyChannel = "auto_generation_task_ready"

// NewBatchGenerator creates the batch generator.
func NewBatchGenerator(pg *pgxpool.Pool, log *zap.Logger, gen BatchAgentGenerator, pgListen *pglisten.Listener, quality QualityValidator) *BatchGenerator {
	return &BatchGenerator{
		pg:       pg,
		log:      log,
		gen:      gen,
		pgListen: pgListen,
		quality:  quality,
		stopCh:   make(chan struct{}),
	}
}

// EnqueueBatch creates tasks for the cartesian product of symbols×timeframes×types.
// Skips combinations that failed 3+ times in the last 7 days.
// Uses a single multi-row INSERT for efficiency.
func (b *BatchGenerator) EnqueueBatch(ctx context.Context, symbols, timeframes, strategyTypes []string, riskLevel string) (int, error) {
	// Build values and args for a single batch INSERT.
	var valueParts []string
	var args []any
	argIdx := 1
	inserted := 0

	for _, sym := range symbols {
		for _, tf := range timeframes {
			for _, st := range strategyTypes {
				skip, err := b.shouldSkip(ctx, sym, tf, st)
				if err != nil {
					b.log.Warn("batch: skip-check failed", zap.Error(err))
				}
				if skip {
					continue
				}
				valueParts = append(valueParts, fmt.Sprintf("($%d, $%d, $%d, $%d, 'pending')", argIdx, argIdx+1, argIdx+2, argIdx+3))
				args = append(args, sym, tf, st, riskLevel)
				argIdx += 4
				inserted++
			}
		}
	}

	if inserted == 0 {
		b.log.Info("batch: no tasks to enqueue (all skipped)")
		return 0, nil
	}

	query := `INSERT INTO auto_generation_tasks (symbol, timeframe, strategy_type, risk_level, status) VALUES ` +
		strings.Join(valueParts, ", ")
	_, err := b.pg.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("batch: batch insert tasks: %w", err)
	}

	pglisten.Notify(ctx, b.pg, notifyChannel, "")
	b.log.Info("batch: enqueued tasks", zap.Int("count", inserted))
	return inserted, nil
}

// shouldSkip returns true if the same combination failed 3+ times in last 7 days.
func (b *BatchGenerator) shouldSkip(ctx context.Context, symbol, timeframe, strategyType string) (bool, error) {
	var failCount int
	err := b.pg.QueryRow(ctx,
		`SELECT count(*) FROM auto_generation_tasks
		 WHERE symbol=$1 AND timeframe=$2 AND strategy_type=$3
		   AND status='rejected' AND created_at > now() - interval '7 days'`,
		symbol, timeframe, strategyType).Scan(&failCount)
	if err != nil {
		return false, err
	}
	return failCount >= 3, nil
}

// Start launches the PG NOTIFY-driven consumer goroutine.
func (b *BatchGenerator) Start(ctx context.Context) {
	b.wg.Add(1)
	go b.consumeLoop(ctx)
}

// Stop gracefully shuts down the consumer.
func (b *BatchGenerator) Stop() {
	close(b.stopCh)
	b.wg.Wait()
}

func (b *BatchGenerator) consumeLoop(ctx context.Context) {
	defer b.wg.Done()

	// Subscribe to NOTIFY channel.
	notifyCh, cancelListen, err := b.pgListen.Listen(ctx, notifyChannel)
	if err != nil {
		b.log.Error("batch: LISTEN failed", zap.Error(err))
		return
	}
	defer cancelListen()

	b.log.Info("batch: consumer started, listening for tasks")
	// Process any existing pending tasks on startup.
	b.drainPending(ctx)

	for {
		select {
		case <-b.stopCh:
			b.log.Info("batch: consumer stopping")
			return
		case <-ctx.Done():
			b.log.Info("batch: consumer context cancelled")
			return
		case <-notifyCh:
			b.drainPending(ctx)
		}
	}
}

// drainPending processes all pending tasks until none remain.
func (b *BatchGenerator) drainPending(ctx context.Context) {
	for {
		task, err := b.claimNextTask(ctx)
		if err != nil {
			b.log.Warn("batch: claim task failed", zap.Error(err))
			return
		}
		if task == nil {
			return // no more pending tasks
		}
		b.processTask(ctx, task)
	}
}

func (b *BatchGenerator) claimNextTask(ctx context.Context) (*AutoGenTask, error) {
	row := b.pg.QueryRow(ctx,
		`UPDATE auto_generation_tasks
		 SET status='generating', started_at=now()
		 WHERE id = (
		   SELECT id FROM auto_generation_tasks
		   WHERE status='pending'
		   ORDER BY created_at
		   LIMIT 1
		   FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, symbol, timeframe, strategy_type, risk_level, status`)

	var t AutoGenTask
	err := row.Scan(&t.ID, &t.Symbol, &t.Timeframe, &t.StrategyType, &t.RiskLevel, &t.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (b *BatchGenerator) processTask(ctx context.Context, task *AutoGenTask) {
	b.log.Info("batch: processing task",
		zap.String("symbol", task.Symbol),
		zap.String("timeframe", task.Timeframe),
		zap.String("type", task.StrategyType))

	desc := fmt.Sprintf("Generate a %s strategy for %s on %s timeframe. Risk level: %s.",
		task.StrategyType, task.Symbol, task.Timeframe, task.RiskLevel)

	agentReq := &antv1.AgentGenerateStrategyRequest{
		Message:   desc,
		Symbol:    task.Symbol,
		Timeframe: task.Timeframe,
		BacktestConfig: &antv1.AgentBacktestConfig{
			Symbol:    task.Symbol,
			Timeframe: task.Timeframe,
		},
	}

	var finalResult *antv1.AgentBacktestResult
	var finalSource string
	var genErr error

	genStream := func(chunk *antv1.AgentGenerateStrategyChunk) error {
		if chunk.PythonSource != "" {
			finalSource = chunk.PythonSource
		}
		if chunk.Result != nil {
			finalResult = chunk.Result
		}
		if chunk.CompileError != "" {
			genErr = fmt.Errorf("compile: %s", chunk.CompileError)
		}
		if chunk.BacktestError != "" {
			genErr = fmt.Errorf("backtest: %s", chunk.BacktestError)
		}
		if chunk.Error != "" {
			genErr = fmt.Errorf("%s", chunk.Error)
		}
		return nil
	}

	systemUserID := uuid.Nil
	if err := b.gen.Generate(ctx, systemUserID, agentReq, genStream); err != nil {
		genErr = err
	}

	if genErr != nil {
		b.failTask(ctx, task.ID, genErr.Error())
		return
	}

	if finalSource == "" {
		b.failTask(ctx, task.ID, "no source code generated")
		return
	}

	// Build snapshot proto.
	var snapshotBytes []byte
	if finalResult != nil {
		snap := &antv1.BacktestSnapshot{
			TotalReturn:  finalResult.TotalReturn,
			AnnualReturn: finalResult.AnnualReturn,
			MaxDrawdown:  finalResult.MaxDrawdown,
			SharpeRatio:  finalResult.SharpeRatio,
			WinRate:      finalResult.WinRate,
			TotalTrades:  finalResult.TotalTrades,
		}
		snapshotBytes, _ = proto.Marshal(snap)
	}

	// Validate quality if a validator is configured.
	qualityPassed := true
	if b.quality != nil && len(snapshotBytes) > 0 {
		violations, qErr := b.quality.ValidateBacktestQuality(ctx, snapshotBytes, "")
		if qErr != nil {
			b.log.Warn("batch: quality validation error", zap.Error(qErr))
		} else {
			qualityPassed = len(violations) == 0
			if !qualityPassed {
				b.log.Info("batch: task failed quality gate",
					zap.String("task", task.ID.String()),
					zap.Int("violations", len(violations)))
			}
		}
	}

	b.completeTask(ctx, task.ID, snapshotBytes, qualityPassed, finalSource)
}

func (b *BatchGenerator) failTask(ctx context.Context, taskID uuid.UUID, errMsg string) {
	_, err := b.pg.Exec(ctx,
		`UPDATE auto_generation_tasks
		 SET status='rejected', error_message=$2, finished_at=now()
		 WHERE id=$1`,
		taskID, errMsg)
	if err != nil {
		b.log.Error("batch: failTask DB error", zap.Error(err))
	}
	b.log.Warn("batch: task failed", zap.String("task", taskID.String()), zap.String("error", errMsg))
}

func (b *BatchGenerator) completeTask(ctx context.Context, taskID uuid.UUID, snapshot []byte, qualityPassed bool, sourceCode string) {
	_, err := b.pg.Exec(ctx,
		`UPDATE auto_generation_tasks
		 SET status='awaiting_review', result_backtest_snapshot=$2, quality_passed=$3, source_code=$4, finished_at=now()
		 WHERE id=$1`,
		taskID, snapshot, qualityPassed, sourceCode)
	if err != nil {
		b.log.Error("batch: completeTask DB error", zap.Error(err))
	}
	b.log.Info("batch: task completed, awaiting review",
		zap.String("task", taskID.String()),
		zap.Bool("quality_passed", qualityPassed))
}
