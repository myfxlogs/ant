package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

type RiskChecker interface {
	CheckRiskLimits(ctx context.Context, req *model.RiskCheckRequest) (*model.RiskCheckResult, error)
}

type ExecutionGateway struct {
	engine        MatchingEngine
	accountRepo   accountRepository
	risk          RiskChecker
	logSvc        *LogService
	store         ExecutionIdempotencyStore
	storeTTL      time.Duration
	sendTimeout   time.Duration
	closeTimeout  time.Duration
	limitMu       sync.Mutex
	limiters      map[string]chan struct{}
	maxConcurrent int
	keyMode       ExecutionGatewayKeyMode
	cb            *CircuitBreaker
	metrics       *ExecutionMetrics
}

var ErrExecutionLimited = errors.New("execution limited")
var ErrorReasonCircuitOpen = "circuit_open"

type ExecutionGatewayKeyMode int

const (
	ExecutionGatewayKeyModeAccount ExecutionGatewayKeyMode = iota
	ExecutionGatewayKeyModeAccountSymbol
)

type accountRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.MTAccount, error)
}

func NewExecutionGateway(engine MatchingEngine, accountRepo accountRepository, risk RiskChecker, logSvc *LogService, store ExecutionIdempotencyStore) *ExecutionGateway {
	return &ExecutionGateway{engine: engine, accountRepo: accountRepo, risk: risk, logSvc: logSvc, store: store, storeTTL: 24 * time.Hour, limiters: map[string]chan struct{}{}, keyMode: ExecutionGatewayKeyModeAccount}
}

func (g *ExecutionGateway) SetRiskChecker(risk RiskChecker) {
	if g == nil {
		return
	}
	g.risk = risk
}

func (g *ExecutionGateway) SetTimeouts(sendTimeout, closeTimeout time.Duration) {
	if g == nil {
		return
	}
	g.sendTimeout = sendTimeout
	g.closeTimeout = closeTimeout
}

func (g *ExecutionGateway) SetConcurrencyLimit(maxConcurrent int) {
	if g == nil {
		return
	}
	g.limitMu.Lock()
	defer g.limitMu.Unlock()
	g.maxConcurrent = maxConcurrent
}

func (g *ExecutionGateway) SetCircuitBreaker(cb *CircuitBreaker) {
	if g == nil {
		return
	}
	g.cb = cb
}

func (g *ExecutionGateway) CircuitBreaker() *CircuitBreaker {
	if g == nil {
		return nil
	}
	return g.cb
}

func (g *ExecutionGateway) SetMetrics(m *ExecutionMetrics) {
	if g == nil {
		return
	}
	g.metrics = m
}

func (g *ExecutionGateway) Metrics() *ExecutionMetrics {
	if g == nil {
		return nil
	}
	return g.metrics
}

func (g *ExecutionGateway) checkCircuitBreaker(op *model.SystemOperationLog) error {
	if g.cb != nil && !g.cb.Allow() {
		if op != nil {
			op.Status = model.OperationStatusFailed
			op.ErrorMessage = ErrCircuitOpen.Error()
			setErrorReason(op, ErrorReasonCircuitOpen)
		}
		if g.metrics != nil {
			g.metrics.RecordCall()
			g.metrics.RecordCircuitOpen()
		}
		return ErrCircuitOpen
	}
	if g.metrics != nil {
		g.metrics.RecordCall()
	}
	return nil
}

func (g *ExecutionGateway) recordEngineResult(err error, latencyMs int64) {
	if g.cb != nil {
		if err != nil && !errors.Is(err, context.Canceled) {
			g.cb.RecordFailure()
		} else {
			g.cb.RecordSuccess()
		}
	}
	if g.metrics != nil {
		g.metrics.RecordLatency(latencyMs)
		if err == nil {
			g.metrics.RecordSuccess()
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			g.metrics.RecordTimeout()
		} else {
			g.metrics.RecordEngineError()
		}
	}
}

func (g *ExecutionGateway) SetConcurrencyKeyMode(mode ExecutionGatewayKeyMode) {
	if g == nil {
		return
	}
	g.limitMu.Lock()
	defer g.limitMu.Unlock()
	g.keyMode = mode
}

const (
	ErrorReasonRiskRejected  = "risk_rejected"
	ErrorReasonEngineError   = "engine_error"
	ErrorReasonTimeout       = "timeout"
	ErrorReasonLimitExceeded = "limit_exceeded"
)

func opNewValueMap(op *model.SystemOperationLog) map[string]interface{} {
	if op == nil {
		return nil
	}
	m, _ := op.NewValue.(map[string]interface{})
	if m == nil {
		m = map[string]interface{}{}
		op.NewValue = m
	}
	return m
}

func setErrorReason(op *model.SystemOperationLog, reason string) {
	if m := opNewValueMap(op); m != nil {
		m["error_reason"] = reason
	}
}

func setRiskDecision(op *model.SystemOperationLog, decision *model.RiskDecision) {
	if m := opNewValueMap(op); m != nil && decision != nil {
		m["risk_decision"] = decision
	}
}

func setEngineLatency(op *model.SystemOperationLog, d time.Duration) {
	if m := opNewValueMap(op); m != nil {
		m["engine_latency_ms"] = d.Milliseconds()
	}
}

func classifyEngineError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ErrorReasonTimeout
	}
	return ErrorReasonEngineError
}

func (g *ExecutionGateway) limiterKey(accountID string, symbol string) string {
	g.limitMu.Lock()
	mode := g.keyMode
	g.limitMu.Unlock()
	if mode == ExecutionGatewayKeyModeAccountSymbol {
		return accountID + ":" + symbol
	}
	return accountID
}

func (g *ExecutionGateway) tryAcquire(key string) (func(), bool) {
	if g == nil {
		return func() {}, true
	}
	g.limitMu.Lock()
	max := g.maxConcurrent
	if max <= 0 {
		g.limitMu.Unlock()
		return func() {}, true
	}
	ch, ok := g.limiters[key]
	if !ok || cap(ch) != max {
		ch = make(chan struct{}, max)
		g.limiters[key] = ch
	}
	g.limitMu.Unlock()

	select {
	case ch <- struct{}{}:
		return func() { <-ch }, true
	default:
		return func() {}, false
	}
}

func (g *ExecutionGateway) idempotencyKey(op *model.SystemOperationLog, accountID string, action string, payload string) string {
	if op == nil || op.ResourceID == uuid.Nil {
		return ""
	}
	sum := sha256.Sum256([]byte(op.Module + ":" + op.Action + ":" + op.ResourceType + ":" + op.ResourceID.String() + ":" + accountID + ":" + action + ":" + payload))
	return fmt.Sprintf("exec:idemp:%s", hex.EncodeToString(sum[:]))
}

func (g *ExecutionGateway) OrderSend(ctx context.Context, userID uuid.UUID, req *OrderSendRequest, op *model.SystemOperationLog) (*OrderResponse, error) {
	if g == nil || g.engine == nil {
		return nil, errors.New("engine not available")
	}
	if req == nil {
		return nil, errors.New("invalid request")
	}
	if err := g.checkCircuitBreaker(op); err != nil {
		return nil, err
	}

	rel, ok := g.tryAcquire(g.limiterKey(req.AccountID, req.Symbol))
	if !ok {
		if op != nil {
			op.Status = model.OperationStatusFailed
			op.ErrorMessage = ErrExecutionLimited.Error()
			m := opNewValueMap(op)
			m["limit"] = map[string]interface{}{"hit": true, "max_concurrent": g.maxConcurrent}
			setErrorReason(op, ErrorReasonLimitExceeded)
			if g.logSvc != nil {
				_ = g.logSvc.LogOperation(ctx, op)
			}
		}
		return nil, ErrExecutionLimited
	}
	defer rel()

	key := ""
	if g.store != nil {
		payload := fmt.Sprintf("%s:%s:%f:%f:%f:%f:%d", req.Symbol, req.Type, req.Volume, req.Price, req.StopLoss, req.TakeProfit, req.Magic)
		key = g.idempotencyKey(op, req.AccountID, "order_send", payload)
		if key != "" {
			if cached, ok, _ := g.store.GetOrderResponse(ctx, key); ok {
				if op != nil {
					m := opNewValueMap(op)
					m["idempotency"] = map[string]interface{}{"hit": true, "key_hash": key}
				}
				return cached, nil
			}
		}
	}
	if op != nil && key != "" {
		m := opNewValueMap(op)
		m["idempotency"] = map[string]interface{}{"hit": false, "key_hash": key}
	}

	if op != nil && g.logSvc != nil {
		_ = g.logSvc.LogOperation(ctx, op)
	}

	if g.risk != nil && g.accountRepo != nil {
		accID, err := uuid.Parse(req.AccountID)
		if err == nil {
			acc, aerr := g.accountRepo.GetByID(ctx, accID)
			if aerr == nil && acc != nil {
				positions, _ := g.engine.GetPositions(ctx, userID, accID)
				riskRes, rerr := g.risk.CheckRiskLimits(ctx, &model.RiskCheckRequest{
					AccountID:      accID,
					Symbol:         req.Symbol,
					Volume:         req.Volume,
					CurrentBalance: acc.Balance,
					CurrentEquity:  acc.Equity,
					OpenPositions:  len(positions),
				})
				if rerr != nil {
					if op != nil {
						op.Status = model.OperationStatusFailed
						op.ErrorMessage = rerr.Error()
						setRiskDecision(op, RiskDecisionFromError(rerr, model.RiskDecisionSourceAuto))
						setErrorReason(op, ErrorReasonRiskRejected)
						if g.logSvc != nil {
							_ = g.logSvc.LogOperation(ctx, op)
						}
					}
					return nil, rerr
				}
				if riskRes != nil && !riskRes.Allowed {
					decision := riskRes.Decision
					if decision == nil {
						decision = model.RejectRiskDecision(model.RiskDecisionSourceAuto, ErrorReasonRiskRejected, riskRes.Reason, false)
					}
					err := errors.New(decision.Reason)
					if op != nil {
						op.Status = model.OperationStatusFailed
						op.ErrorMessage = err.Error()
						setRiskDecision(op, decision)
						setErrorReason(op, ErrorReasonRiskRejected)
						if g.logSvc != nil {
							_ = g.logSvc.LogOperation(ctx, op)
						}
					}
					return nil, err
				}
			}
		}
	}

	callCtx := ctx
	cancel := func() {}
	if g.sendTimeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, g.sendTimeout)
		defer cancel()
	}
	engStart := time.Now()
	resp, err := g.engine.OrderSend(callCtx, userID, req)
	engDur := time.Since(engStart)
	g.recordEngineResult(err, engDur.Milliseconds())
	if err == nil && resp != nil && key != "" && g.store != nil {
		_ = g.store.SetOrderResponse(ctx, key, resp, g.storeTTL)
	}
	if op != nil {
		setEngineLatency(op, engDur)
		if err != nil {
			op.Status = model.OperationStatusFailed
			op.ErrorMessage = err.Error()
			setErrorReason(op, classifyEngineError(err))
		} else {
			op.Status = model.OperationStatusCompleted
		}
		op.DurationMs = time.Since(op.CreatedAt).Milliseconds()
		if g.logSvc != nil {
			_ = g.logSvc.LogOperation(ctx, op)
		}
	}
	return resp, err
}

func (g *ExecutionGateway) OrderModify(ctx context.Context, userID uuid.UUID, req *OrderModifyRequest, op *model.SystemOperationLog) (*OrderResponse, error) {
	if g == nil || g.engine == nil {
		return nil, errors.New("engine not available")
	}
	if req == nil {
		return nil, errors.New("invalid request")
	}
	if err := g.checkCircuitBreaker(op); err != nil {
		return nil, err
	}

	rel, ok := g.tryAcquire(g.limiterKey(req.AccountID, ""))
	if !ok {
		if op != nil {
			op.Status = model.OperationStatusFailed
			op.ErrorMessage = ErrExecutionLimited.Error()
			m := opNewValueMap(op)
			m["limit"] = map[string]interface{}{"hit": true, "max_concurrent": g.maxConcurrent}
			setErrorReason(op, ErrorReasonLimitExceeded)
			if g.logSvc != nil {
				_ = g.logSvc.LogOperation(ctx, op)
			}
		}
		return nil, ErrExecutionLimited
	}
	defer rel()

	key := ""
	if g.store != nil {
		payload := fmt.Sprintf("%d:%f:%f:%f", req.Ticket, req.StopLoss, req.TakeProfit, req.Price)
		key = g.idempotencyKey(op, req.AccountID, "order_modify", payload)
		if key != "" {
			if cached, ok, _ := g.store.GetOrderResponse(ctx, key); ok {
				if op != nil {
					m := opNewValueMap(op)
					m["idempotency"] = map[string]interface{}{"hit": true, "key_hash": key}
				}
				return cached, nil
			}
		}
	}
	if op != nil && key != "" {
		m := opNewValueMap(op)
		m["idempotency"] = map[string]interface{}{"hit": false, "key_hash": key}
	}

	if op != nil && g.logSvc != nil {
		_ = g.logSvc.LogOperation(ctx, op)
	}

	callCtx := ctx
	cancel := func() {}
	if g.sendTimeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, g.sendTimeout)
		defer cancel()
	}
	engStart := time.Now()
	resp, err := g.engine.OrderModify(callCtx, userID, req)
	engDur := time.Since(engStart)
	g.recordEngineResult(err, engDur.Milliseconds())
	if err == nil && resp != nil && key != "" && g.store != nil {
		_ = g.store.SetOrderResponse(ctx, key, resp, g.storeTTL)
	}
	if op != nil {
		setEngineLatency(op, engDur)
		if err != nil {
			op.Status = model.OperationStatusFailed
			op.ErrorMessage = err.Error()
			setErrorReason(op, classifyEngineError(err))
		} else {
			op.Status = model.OperationStatusCompleted
		}
		op.DurationMs = time.Since(op.CreatedAt).Milliseconds()
		if g.logSvc != nil {
			_ = g.logSvc.LogOperation(ctx, op)
		}
	}
	return resp, err
}

func (g *ExecutionGateway) OrderClose(ctx context.Context, userID uuid.UUID, req *OrderCloseRequest, op *model.SystemOperationLog) (*OrderResponse, error) {
	if g == nil || g.engine == nil {
		return nil, errors.New("engine not available")
	}
	if req == nil {
		return nil, errors.New("invalid request")
	}
	if err := g.checkCircuitBreaker(op); err != nil {
		return nil, err
	}

	rel, ok := g.tryAcquire(g.limiterKey(req.AccountID, ""))
	if !ok {
		if op != nil {
			op.Status = model.OperationStatusFailed
			op.ErrorMessage = ErrExecutionLimited.Error()
			m := opNewValueMap(op)
			m["limit"] = map[string]interface{}{"hit": true, "max_concurrent": g.maxConcurrent}
			setErrorReason(op, ErrorReasonLimitExceeded)
			if g.logSvc != nil {
				_ = g.logSvc.LogOperation(ctx, op)
			}
		}
		return nil, ErrExecutionLimited
	}
	defer rel()

	key := ""
	if g.store != nil {
		payload := fmt.Sprintf("%d:%f", req.Ticket, req.Volume)
		key = g.idempotencyKey(op, req.AccountID, "order_close", payload)
		if key != "" {
			if cached, ok, _ := g.store.GetOrderResponse(ctx, key); ok {
				if op != nil {
					m := opNewValueMap(op)
					m["idempotency"] = map[string]interface{}{"hit": true, "key_hash": key}
				}
				return cached, nil
			}
		}
	}
	if op != nil && key != "" {
		m := opNewValueMap(op)
		m["idempotency"] = map[string]interface{}{"hit": false, "key_hash": key}
	}

	if op != nil && g.logSvc != nil {
		_ = g.logSvc.LogOperation(ctx, op)
	}

	callCtx := ctx
	cancel := func() {}
	if g.closeTimeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, g.closeTimeout)
		defer cancel()
	}
	engStart := time.Now()
	resp, err := g.engine.OrderClose(callCtx, userID, req)
	engDur := time.Since(engStart)
	g.recordEngineResult(err, engDur.Milliseconds())
	if err == nil && resp != nil && key != "" && g.store != nil {
		_ = g.store.SetOrderResponse(ctx, key, resp, g.storeTTL)
	}
	if op != nil {
		setEngineLatency(op, engDur)
		if err != nil {
			op.Status = model.OperationStatusFailed
			op.ErrorMessage = err.Error()
			setErrorReason(op, classifyEngineError(err))
		} else {
			op.Status = model.OperationStatusCompleted
		}
		op.DurationMs = time.Since(op.CreatedAt).Milliseconds()
		if g.logSvc != nil {
			_ = g.logSvc.LogOperation(ctx, op)
		}
	}
	return resp, err
}
