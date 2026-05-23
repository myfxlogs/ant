package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

type ExecutionAdapter interface {
	OrderSend(ctx context.Context, userID uuid.UUID, req *OrderSendRequest, op *model.SystemOperationLog) (*ExecutionAdapterResult, error)
	OrderModify(ctx context.Context, userID uuid.UUID, req *OrderModifyRequest, op *model.SystemOperationLog) (*ExecutionAdapterResult, error)
	OrderClose(ctx context.Context, userID uuid.UUID, req *OrderCloseRequest, op *model.SystemOperationLog) (*ExecutionAdapterResult, error)
}

type ExecutionAdapterResult struct {
	Response  *OrderResponse
	Status    string
	Reason    string
	LatencyMs int64
	Engine    string
	RawError  string
	Metadata  map[string]string
}

type GatewayExecutionAdapter struct {
	gateway *ExecutionGateway
	engine  string
}

func NewGatewayExecutionAdapter(gateway *ExecutionGateway, engine string) *GatewayExecutionAdapter {
	if engine == "" {
		engine = "default"
	}
	return &GatewayExecutionAdapter{gateway: gateway, engine: engine}
}

func (a *GatewayExecutionAdapter) OrderSend(ctx context.Context, userID uuid.UUID, req *OrderSendRequest, op *model.SystemOperationLog) (*ExecutionAdapterResult, error) {
	if a == nil || a.gateway == nil {
		return nil, errors.New("execution gateway not available")
	}
	started := time.Now()
	resp, err := a.gateway.OrderSend(ctx, userID, req, op)
	return a.result(resp, err, time.Since(started)), err
}

func (a *GatewayExecutionAdapter) OrderModify(ctx context.Context, userID uuid.UUID, req *OrderModifyRequest, op *model.SystemOperationLog) (*ExecutionAdapterResult, error) {
	if a == nil || a.gateway == nil {
		return nil, errors.New("execution gateway not available")
	}
	started := time.Now()
	resp, err := a.gateway.OrderModify(ctx, userID, req, op)
	return a.result(resp, err, time.Since(started)), err
}

func (a *GatewayExecutionAdapter) OrderClose(ctx context.Context, userID uuid.UUID, req *OrderCloseRequest, op *model.SystemOperationLog) (*ExecutionAdapterResult, error) {
	if a == nil || a.gateway == nil {
		return nil, errors.New("execution gateway not available")
	}
	started := time.Now()
	resp, err := a.gateway.OrderClose(ctx, userID, req, op)
	return a.result(resp, err, time.Since(started)), err
}

func (a *GatewayExecutionAdapter) result(resp *OrderResponse, err error, latency time.Duration) *ExecutionAdapterResult {
	status := "ok"
	reason := ""
	raw := ""
	if err != nil {
		status = "error"
		reason = classifyEngineError(err)
		raw = err.Error()
	}
	return &ExecutionAdapterResult{Response: resp, Status: status, Reason: reason, LatencyMs: latency.Milliseconds(), Engine: a.engine, RawError: raw, Metadata: map[string]string{"adapter": "gateway"}}
}

func NewExecutionAdapterFactory(gateway *ExecutionGateway) ExecutionAdapter {
	return NewGatewayExecutionAdapter(gateway, "matching_engine")
}
