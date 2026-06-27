// Package algo implements the ExecutionAlgoService ConnectRPC handler (M12-A2).
// It wraps the execalgo.Executor runtime, managing execution lifecycles
// (start, status query, cancel) with a thread-safe registry.
package algo

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/execalgo"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
)

// ExecutionAlgoServer implements ant.v1.ExecutionAlgoServiceHandler.
type ExecutionAlgoServer struct {
	mu       sync.RWMutex
	active   map[string]*execution // executionID → running execution
	broker   mthub.BrokerRegistry  // M12-C2: multi-broker registry
	log      *zap.Logger
}

// execution tracks a running algo execution.
type execution struct {
	exec     *execalgo.Executor
	algoName string
	parent   execalgo.ParentOrder
	started  time.Time
}

var _ antv1c.ExecutionAlgoServiceHandler = (*ExecutionAlgoServer)(nil)

// NewExecutionAlgoServer creates the handler. broker may be nil if no
// multi-broker registry is configured.
func NewExecutionAlgoServer(broker mthub.BrokerRegistry, log *zap.Logger) *ExecutionAlgoServer {
	return &ExecutionAlgoServer{
		active: make(map[string]*execution),
		broker: broker,
		log:    log,
	}
}

func (s *ExecutionAlgoServer) StartAlgo(ctx context.Context, req *connect.Request[antv1.StartAlgoRequest]) (*connect.Response[antv1.StartAlgoResponse], error) {
	m := req.Msg
	if err := validateStartAlgoRequest(m); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	algo, err := selectAlgo(m)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	startTime, endTime := m.StartTime.AsTime(), m.EndTime.AsTime()
	totalVol, _ := strconv.ParseFloat(m.TotalVolume, 64)
	limitPx, _ := strconv.ParseFloat(m.LimitPrice, 64)
	arrivalPx, _ := strconv.ParseFloat(m.ArrivalPrice, 64)
	parent := execalgo.ParentOrder{
		Symbol: m.Symbol, Side: m.Side, TotalVolume: totalVol,
		StartTime: startTime, EndTime: endTime,
		LimitPrice: limitPx, ArrivalPrice: arrivalPx,
	}

	schedule, err := algo.Schedule(parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("schedule generation: %w", err))
	}

	brokerExec, err := s.resolveBrokerExecutor()
	if err != nil {
		return nil, err
	}

	execID := s.createAndStartExecutor(ctx, schedule, algo.Name(), parent, m.AccountId, brokerExec)

	s.log.Info("algo execution started",
		zap.String("execution_id", execID), zap.String("algo", algo.Name()),
		zap.String("user_id", userID), zap.Int("total_slices", len(schedule.Slices)))

	return connect.NewResponse(&antv1.StartAlgoResponse{
		ExecutionId: execID, Algo: algo.Name(), TotalSlices: int32(len(schedule.Slices)),
	}), nil
}

func validateStartAlgoRequest(m *antv1.StartAlgoRequest) error {
	if m.AccountId == "" { return fmt.Errorf("account_id is required") }
	if m.Symbol == "" { return fmt.Errorf("symbol is required") }
	if m.Side != "buy" && m.Side != "sell" { return fmt.Errorf("side must be 'buy' or 'sell'") }
	v, _ := strconv.ParseFloat(m.TotalVolume, 64)
	if m.TotalVolume == "" || v <= 0 { return fmt.Errorf("total_volume must be positive") }
	if m.StartTime == nil || m.EndTime == nil { return fmt.Errorf("start_time and end_time are required") }
	startTime, endTime := m.StartTime.AsTime(), m.EndTime.AsTime()
	if !endTime.After(startTime) { return fmt.Errorf("end_time must be after start_time") }
	return nil
}

func (s *ExecutionAlgoServer) resolveBrokerExecutor() (mthub.BrokerExecutor, error) {
	if s.broker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("no broker configured for algo execution"))
	}
	names := s.broker.List()
	if len(names) == 0 {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("no broker configured for algo execution"))
	}
	be, err := s.broker.Resolve(names[0])
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("broker resolution: %w", err))
	}
	return be, nil
}

func (s *ExecutionAlgoServer) createAndStartExecutor(
	ctx context.Context,
	schedule *execalgo.Schedule,
	algoName string,
	parent execalgo.ParentOrder,
	accountID string,
	brokerExec mthub.BrokerExecutor,
) string {
	execID := uuid.New().String()
	executor := execalgo.NewExecutor(execalgo.ExecutorConfig{
		Schedule: schedule, Broker: brokerExec, AccountID: accountID,
	})

	s.mu.Lock()
	s.active[execID] = &execution{exec: executor, algoName: algoName, parent: parent, started: time.Now()}
	s.mu.Unlock()

	executor.Start(ctx)

	go func() {
		for range executor.Events() {}
		s.mu.Lock()
		delete(s.active, execID)
		s.mu.Unlock()
		s.log.Info("algo execution completed", zap.String("execution_id", execID), zap.String("algo", algoName))
	}()

	return execID
}

func (s *ExecutionAlgoServer) GetAlgoStatus(ctx context.Context, req *connect.Request[antv1.GetAlgoStatusRequest]) (*connect.Response[antv1.GetAlgoStatusResponse], error) {
	execID := req.Msg.ExecutionId
	if execID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("execution_id is required"))
	}

	s.mu.RLock()
	ex, ok := s.active[execID]
	s.mu.RUnlock()

	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("execution %q not found", execID))
	}

	state := ex.exec.State()
	submitted, total := ex.exec.Progress()

	resp := &antv1.GetAlgoStatusResponse{
		ExecutionId:     execID,
		Algo:            ex.algoName,
		State:           state.String(),
		SubmittedSlices: int32(submitted),
		TotalSlices:     int32(total),
		ParentSymbol:    ex.parent.Symbol,
		ParentSide:      ex.parent.Side,
		ParentVolume:    strconv.FormatFloat(ex.parent.TotalVolume, 'f', -1, 64),
	}

	return connect.NewResponse(resp), nil
}

func (s *ExecutionAlgoServer) CancelAlgo(ctx context.Context, req *connect.Request[antv1.CancelAlgoRequest]) (*connect.Response[antv1.CancelAlgoResponse], error) {
	execID := req.Msg.ExecutionId
	if execID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("execution_id is required"))
	}

	s.mu.RLock()
	ex, ok := s.active[execID]
	s.mu.RUnlock()

	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("execution %q not found", execID))
	}

	ex.exec.Cancel()
	return connect.NewResponse(&antv1.CancelAlgoResponse{
		ExecutionId: execID,
		State:       ex.exec.State().String(),
	}), nil
}

func (s *ExecutionAlgoServer) ListAlgos(ctx context.Context, req *connect.Request[antv1.ListAlgosRequest]) (*connect.Response[antv1.ListAlgosResponse], error) {
	algos := []*antv1.AlgoInfo{
		{
			Name:        "twap",
			Description: "Time-Weighted Average Price — equal slices at regular intervals. Best for low-urgency orders in liquid markets.",
			Parameters:  []string{"slice_interval"},
		},
		{
			Name:        "vwap",
			Description: "Volume-Weighted Average Price — slices proportional to historical volume profile. Minimizes market impact by trading when volume is highest.",
			Parameters:  []string{"slice_interval"},
		},
		{
			Name:        "pov",
			Description: "Percentage of Volume — targets a fixed participation rate of market volume. Adapts to changing liquidity conditions.",
			Parameters:  []string{"slice_interval", "participation_rate"},
		},
		{
			Name:        "shortfall",
			Description: "Implementation Shortfall — front-loads execution to minimize drift from arrival price. Best for urgent orders where price risk dominates impact cost.",
			Parameters:  []string{"slice_interval", "urgency"},
		},
	}
	return connect.NewResponse(&antv1.ListAlgosResponse{Algos: algos}), nil
}

// selectAlgo creates the appropriate Algo from the request parameters.
func selectAlgo(m *antv1.StartAlgoRequest) (execalgo.Algo, error) {
	interval := 1 * time.Minute
	if m.SliceInterval != nil {
		interval = m.SliceInterval.AsDuration()
		if interval <= 0 {
			interval = 1 * time.Minute
		}
	}

	switch m.Algo {
	case "twap":
		return execalgo.NewTwap(interval), nil
	case "vwap":
		return execalgo.NewVwap(execalgo.FlatVolumeProfile{}, 12), nil
	case "pov":
		rate := m.ParticipationRate
		if rate <= 0 || rate > 1 {
			rate = 0.1
		}
		return execalgo.NewPov(rate, interval, 0), nil
	case "shortfall":
		urgency := m.Urgency
		if urgency < 0 {
			urgency = 0
		}
		if urgency > 1 {
			urgency = 1
		}
		// numSlices = duration / interval
		if m.StartTime == nil || m.EndTime == nil {
			return nil, fmt.Errorf("start_time and end_time are required")
		}
		startTime := m.StartTime.AsTime()
		endTime := m.EndTime.AsTime()
		dur := endTime.Sub(startTime)
		numSlices := int(dur / interval)
		if numSlices < 1 {
			numSlices = 1
		}
		return execalgo.NewShortfall(urgency, numSlices), nil
	default:
		return nil, fmt.Errorf("unknown algo %q (supported: twap, vwap, pov, shortfall)", m.Algo)
	}
}
