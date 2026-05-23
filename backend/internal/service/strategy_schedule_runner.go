package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/connection"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

type StrategyScheduleRunner struct {
	scheduleRepo *repository.StrategyScheduleRepository
	templateRepo *repository.StrategyTemplateRepository
	accountRepo  *repository.AccountRepository
	connMgr      *connection.ConnectionManager
	klineSvc     *KlineService
	pythonSvc    *PythonStrategyService
	gateway      *ExecutionGateway
	tradingSvc   *TradingService // 用于 risk gate 查询持仓 / 账户权益
	logSvc       *LogService
	stateStore   ScheduleRuntimeStateStore

	mu      sync.Mutex
	running map[uuid.UUID]context.CancelFunc
	stateMu sync.Mutex
	states  map[uuid.UUID]*ScheduleRuntimeState

	ctx    context.Context
	cancel context.CancelFunc
}

type ScheduleRuntimeState struct {
	ScheduleID uuid.UUID
	StartedAt  time.Time

	LastBarOpenTime string
	LastSignalKey   string

	LastEvalAt   time.Time
	LastOrderAt  time.Time
	LastSignal   string
	LastSignalAt time.Time

	MartingaleLevel int

	// PeakEquity 用于 __risk.max_drawdown_pct 检查；初次调用时会被初始化为当前权益。
	// 仅存在内存态；重启后从第一次观测到的权益重新计算，这是可以接受的近似。
	PeakEquity float64

	Data map[string]interface{}
}

var (
	scheduleRunnerLogThrottleMu   sync.Mutex
	scheduleRunnerLogThrottleLast = make(map[string]time.Time)
	scheduleRunnerLogSuppressed   = make(map[string]int)
)

type ScheduleRuntimeStateStore interface {
	Load(ctx context.Context, scheduleID uuid.UUID) (*ScheduleRuntimeState, bool, error)
	Save(ctx context.Context, state *ScheduleRuntimeState) error
	Delete(ctx context.Context, scheduleID uuid.UUID) error
}

func NewStrategyScheduleRunner(
	scheduleRepo *repository.StrategyScheduleRepository,
	templateRepo *repository.StrategyTemplateRepository,
	accountRepo *repository.AccountRepository,
	connMgr *connection.ConnectionManager,
	klineSvc *KlineService,
	pythonSvc *PythonStrategyService,
	gateway *ExecutionGateway,
	tradingSvc *TradingService,
	logSvc *LogService,
) *StrategyScheduleRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &StrategyScheduleRunner{
		scheduleRepo: scheduleRepo,
		templateRepo: templateRepo,
		accountRepo:  accountRepo,
		connMgr:      connMgr,
		klineSvc:     klineSvc,
		pythonSvc:    pythonSvc,
		gateway:      gateway,
		tradingSvc:   tradingSvc,
		logSvc:       logSvc,
		running:      map[uuid.UUID]context.CancelFunc{},
		states:       map[uuid.UUID]*ScheduleRuntimeState{},
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (r *StrategyScheduleRunner) SetStateStore(store ScheduleRuntimeStateStore) {
	if r == nil {
		return
	}
	r.stateMu.Lock()
	r.stateStore = store
	r.stateMu.Unlock()
}

func (r *StrategyScheduleRunner) Start() {
	if r == nil {
		return
	}
	go r.supervise()
}

func (r *StrategyScheduleRunner) Stop() {
	if r == nil {
		return
	}
	r.cancel()

	r.mu.Lock()
	for _, cancel := range r.running {
		cancel()
	}
	r.running = map[uuid.UUID]context.CancelFunc{}
	r.mu.Unlock()

	r.stateMu.Lock()
	r.states = map[uuid.UUID]*ScheduleRuntimeState{}
	r.stateMu.Unlock()
}

func (r *StrategyScheduleRunner) supervise() {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-tick.C:
			r.reconcile()
		}
	}
}

func (r *StrategyScheduleRunner) reconcile() {
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()

	active, err := r.scheduleRepo.GetActiveSchedules(ctx)
	if err != nil {
		logger.Warn("StrategyScheduleRunner: load active schedules failed", zap.Error(err))
		return
	}

	activeSet := make(map[uuid.UUID]*model.StrategySchedule, len(active))
	for _, s := range active {
		if s == nil {
			continue
		}
		activeSet[s.ID] = s
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// stop missing
	for id, cancelFn := range r.running {
		if _, ok := activeSet[id]; !ok {
			cancelFn()
			delete(r.running, id)
		}
	}

	// start new
	for id, sch := range activeSet {
		if _, ok := r.running[id]; ok {
			continue
		}
		r.ensureState(id)

		sctx, scancel := context.WithCancel(r.ctx)
		r.running[id] = scancel
		go r.runSchedule(sctx, sch)
	}
}
