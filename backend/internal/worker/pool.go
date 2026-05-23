package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type Task func(ctx context.Context) error

type WorkerPool struct {
	poolSize    int
	taskQueue   chan Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	running     bool
	stats       *PoolStats
}

type PoolStats struct {
	mu          sync.RWMutex
	TotalTasks  int64
	Completed   int64
	Failed      int64
	QueueLength int64
	ActiveWorkers int64
}

func NewWorkerPool(poolSize int, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		poolSize:  poolSize,
		taskQueue: make(chan Task, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		stats: &PoolStats{},
	}
}

func (p *WorkerPool) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	for i := 0; i < p.poolSize; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}

			p.mu.RLock()
			running := p.running
			p.mu.RUnlock()

			if !running {
				return
			}

			p.stats.mu.Lock()
			p.stats.TotalTasks++
			p.stats.ActiveWorkers++
			p.stats.mu.Unlock()

			err := task(p.ctx)

			p.stats.mu.Lock()
			p.stats.ActiveWorkers--
			if err != nil {
				p.stats.Failed++
			} else {
				p.stats.Completed++
			}
			p.stats.mu.Unlock()

			if err != nil {
				logger.Warn("Worker task failed",
					zap.Int("worker_id", id),
					zap.Error(err))
			}
		}
	}
}

func (p *WorkerPool) Submit(task Task) error {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	if !running {
		return context.Canceled
	}

	select {
	case <-p.ctx.Done():
		return context.Canceled
	case p.taskQueue <- task:
		p.stats.mu.Lock()
		p.stats.QueueLength = int64(len(p.taskQueue))
		p.stats.mu.Unlock()
		return nil
	}
}

func (p *WorkerPool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		errChan <- p.Submit(task)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *WorkerPool) Shutdown() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	close(p.taskQueue)
	p.cancel()
	p.wg.Wait()

}

func (p *WorkerPool) Stats() *PoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	return &PoolStats{
		TotalTasks:    p.stats.TotalTasks,
		Completed:     p.stats.Completed,
		Failed:        p.stats.Failed,
		QueueLength:   p.stats.QueueLength,
		ActiveWorkers: p.stats.ActiveWorkers,
	}
}

func (s *PoolStats) GetTotalTasks() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalTasks
}

func (s *PoolStats) GetCompleted() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Completed
}

func (s *PoolStats) GetFailed() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Failed
}

func (s *PoolStats) GetQueueLength() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.QueueLength
}

func (s *PoolStats) GetActiveWorkers() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ActiveWorkers
}

func (s *PoolStats) GetSuccessRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := s.Completed + s.Failed
	if total == 0 {
		return 100.0
	}

	return float64(s.Completed) / float64(total) * 100
}

type StrategyTask struct {
	UserID      string
	AccountID   string
	StrategyID  string
	Prompt      string
	ResultChan  chan<- string
	ErrorChan   chan<- error
	CreatedAt   time.Time
	Timeout     time.Duration
}

type StrategyWorkerPool struct {
	pool      *WorkerPool
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewStrategyWorkerPool(poolSize int, queueSize int) *StrategyWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &StrategyWorkerPool{
		pool:      NewWorkerPool(poolSize, queueSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (p *StrategyWorkerPool) Start() {
	p.pool.Start()
}

func (p *StrategyWorkerPool) SubmitStrategyTask(task *StrategyTask) error {
	return p.pool.Submit(func(ctx context.Context) error {
		return p.executeStrategyTask(ctx, task)
	})
}

func (p *StrategyWorkerPool) executeStrategyTask(ctx context.Context, task *StrategyTask) error {
	defer func() {
		if r := recover(); r != nil {
			task.ErrorChan <- fmt.Errorf("strategy execution panic: %v", r)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(task.Timeout):
		return context.DeadlineExceeded
	}
}

func (p *StrategyWorkerPool) Shutdown() {
	p.pool.Shutdown()
	p.cancel()
}

func (p *StrategyWorkerPool) Stats() *PoolStats {
	return p.pool.Stats()
}
