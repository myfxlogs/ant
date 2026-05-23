//go:build nautilus_training

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"anttrader/pkg/logger"
)

// TrainingEngine AI 训练引擎，整合 nautilus_trader 高性能回测引擎
// 支持 RL/ES 等强化学习算法训练
type TrainingEngine struct {
	nautilusPath string
	cmd          *exec.Cmd
	proc         *os.Process
	mu           sync.RWMutex

	// 训练任务管理
	trainers map[string]*TrainingSession
	ctx      context.Context
	cancel   context.CancelFunc

	// 回调
	OnTrainStart  func(session *TrainingSession)
	OnTrainEnd    func(session *TrainingSession)
	OnTrainError  func(session *TrainingSession, err error)
	OnTrainMetric func(session *TrainingSession, metric string, value float64)
}

// TrainingSession 训练会话
type TrainingSession struct {
	ID          string
	StrategyID  string
	Environment string
	Algorithm   string
	Params      json.RawMessage
	Status      TrainingStatus
	StartTime   time.Time
	EndTime     time.Time
	Metrics     map[string]MetricPoint
	Logs        []string
	Errors      []string
}

type MetricPoint struct {
	Time   time.Time
	Value  float64
	Metric string
}

type TrainingStatus string

const (
	StatusPending   TrainingStatus = "pending"
	StatusRunning   TrainingStatus = "running"
	StatusCompleted TrainingStatus = "completed"
	StatusFailed    TrainingStatus = "failed"
	StatusCancelled TrainingStatus = "cancelled"
)

const (
	defaultNautilusPath = "/home/mluser/code/anttrader-grpc/nautilus_trader/nautilus_trader"
)

// NewTrainingEngine 创建训练引擎
func NewTrainingEngine(nautilusPath string) *TrainingEngine {
	if nautilusPath == "" {
		nautilusPath = defaultNautilusPath
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &TrainingEngine{
		nautilusPath: nautilusPath,
		trainers:     make(map[string]*TrainingSession),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start 启动训练引擎（一次性启动 nautilus_trader 进程池）
func (e *TrainingEngine) Start() error {
	if e.cmd != nil && e.cmd.ProcessState == nil {
		return nil // 已启动
	}

	// nautilus_trader 通常作为库使用，训练通过 subprocess 调用
	// 这里预留进程池管理逻辑
	return nil
}

// Stop 停止训练引擎
func (e *TrainingEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cancel()

	e.mu.Lock()
	for _, session := range e.trainers {
		if session.Status == StatusRunning {
			session.Status = StatusCancelled
			session.EndTime = time.Now()
		}
	}
	e.mu.Unlock()

	if e.cmd != nil && e.cmd.ProcessState == nil {
		e.cmd.Process.Kill()
		e.cmd = nil
	}
}

// CreateSession 创建训练会话
func (e *TrainingEngine) CreateSession(ctx context.Context, req *CreateTrainingRequest) (*CreateTrainingResponse, error) {
	if req == nil || req.StrategyID == "" {
		return nil, errors.New("strategy ID is required")
	}

	session := &TrainingSession{
		ID:          fmt.Sprintf("train-%d", time.Now().UnixNano()),
		StrategyID:  req.StrategyID,
		Environment: req.Environment,
		Algorithm:   req.Algorithm,
		Status:      StatusPending,
		StartTime:   time.Now(),
		Metrics:     make(map[string]MetricPoint),
		Logs:        make([]string, 0),
		Errors:      make([]string, 0),
	}

	e.mu.Lock()
	e.trainers[session.ID] = session
	e.mu.Unlock()

	if e.OnTrainStart != nil {
		e.OnTrainStart(session)
	}

	logger.Info("Training session created", zap.String("session_id", session.ID))

	return &CreateTrainingResponse{
		SessionId: session.ID,
	}, nil
}

// Run 运行训练任务
func (e *TrainingEngine) Run(ctx context.Context, sessionID string, req *RunTrainingRequest) (*RunTrainingResponse, error) {
	e.mu.Lock()
	session, exists := e.trainers[sessionID]
	if !exists {
		e.mu.Unlock()
		return nil, errors.New("training session not found")
	}
	if session.Status != StatusPending {
		e.mu.Unlock()
		return nil, fmt.Errorf("session status is %s, cannot run", session.Status)
	}
	e.trainers[sessionID] = session
	e.mu.Unlock()

	session.Status = StatusRunning

	// 构建 nautilus_trader 训练命令
	// 根据策略类型选择不同的训练方式
	cmd, err := e.buildTrainingCommand(session, req)
	if err != nil {
		session.Status = StatusFailed
		session.EndTime = time.Now()
		session.Errors = append(session.Errors, err.Error())
		e.mu.Lock()
		e.trainers[sessionID] = session
		e.mu.Unlock()
		if e.OnTrainError != nil {
			e.OnTrainError(session, err)
		}
		return nil, err
	}

	logger.Info("Starting training command",
		zap.String("session_id", sessionID),
		zap.String("cmd", cmd))

	// 执行训练命令
	session.Cmd = cmd
	session.CmdProc = exec.CommandContext(ctx, cmd)

	output, err := session.CmdProc.CombinedOutput()
	if err != nil {
		session.Status = StatusFailed
		session.EndTime = time.Now()
		session.Errors = append(session.Errors, fmt.Sprintf("execution error: %v", err))
		session.Logs = append(session.Logs, string(output))
		e.mu.Lock()
		e.trainers[sessionID] = session
		e.mu.Unlock()
		if e.OnTrainError != nil {
			e.OnTrainError(session, err)
		}
		return nil, fmt.Errorf("training failed: %w", err)
	}

	// 解析训练结果
	result, err := e.parseTrainingOutput(session, output)
	if err != nil {
		session.Errors = append(session.Errors, err.Error())
		if e.OnTrainError != nil {
			e.OnTrainError(session, errors.New(err.Error()))
		}
	} else {
		session.Status = StatusCompleted
		session.EndTime = time.Now()
		if e.OnTrainEnd != nil {
			e.OnTrainEnd(session)
		}
	}

	e.mu.Lock()
	e.trainers[sessionID] = session
	e.mu.Unlock()

	return &RunTrainingResponse{
		Success:     session.Status == StatusCompleted,
		SessionId:   sessionID,
		Stats:       result,
		OutputLogs:  session.Logs,
	}, nil
}

// buildTrainingCommand 构建训练命令
func (e *TrainingEngine) buildTrainingCommand(session *TrainingSession, req *RunTrainingRequest) (string, error) {
	var args []string

	// 根据算法类型构建命令
	switch session.Algorithm {
	case "PPO", "SAC", "TD3", "DQN": // 强化学习算法
		args = append(args, "train_rl",
			"--strategy", session.StrategyID,
			"--env", session.Environment,
			"--algo", session.Algorithm,
		)
	case "CMA-ES", "CMA": // 进化策略
		args = append(args, "train_es",
			"--strategy", session.StrategyID,
			"--algo", session.Algorithm,
		)
	case "genetic", "ga": // 遗传算法
		args = append(args, "train_genetic",
			"--strategy", session.StrategyID,
		)
	case "bayesian", "bayes": // 贝叶斯优化
		args = append(args, "train_bayesian",
			"--strategy", session.StrategyID,
		)
	default:
		return "", fmt.Errorf("unknown algorithm: %s", session.Algorithm)
	}

	// 添加参数
	args = append(args, req.Params...)

	// 输出到文件
	outputDir := "/tmp/nautilus_training"
	if req.OutputDir != "" {
		outputDir = req.OutputDir
	}
	outputFile := filepath.Join(outputDir, session.ID+".json")
	args = append(args, "--output", outputFile)

	return "python", args
}

// parseTrainingOutput 解析训练输出
func (e *TrainingEngine) parseTrainingOutput(session *TrainingSession, output []byte) (*TrainingStats, error) {
	var stats TrainingStats
	var err error

	// 尝试解析 JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err == nil {
		stats = &TrainingStats{}
		if val, ok := result["stats"].(map[string]interface{}); ok {
			stats = e.fromMap(val)
		}
	}

	// 如果没有 JSON 解析，尝试从文本提取
	if stats == nil {
		stats = e.extractStatsFromText(string(output))
	}

	return stats, nil
}

// GetSession 获取训练会话详情
func (e *TrainingEngine) GetSession(ctx context.Context, sessionID string) (*GetSessionResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	session, exists := e.trainers[sessionID]
	if !exists {
		return nil, errors.New("training session not found")
	}

	return &GetSessionResponse{
		SessionId: session.ID,
		StrategyId: session.StrategyID,
		Environment: session.Environment,
		Algorithm: session.Algorithm,
		Params:     session.Params,
		Status:     string(session.Status),
		StartTime:  session.StartTime,
		EndTime:    session.EndTime,
		Metrics:    session.Metrics,
		Logs:       session.Logs,
		Errors:     session.Errors,
	}, nil
}

// ListSessions 列出所有训练会话
func (e *TrainingEngine) ListSessions(ctx context.Context, req *ListTrainingRequest) (*ListTrainingResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var sessions []*TrainingSession
	filter := req.Status

	for _, s := range e.trainers {
		if filter != "" && s.Status != TrainingStatus(filter) {
			continue
		}
		sessions = append(sessions, s)
	}

	return &ListTrainingResponse{
		Sessions: sessions,
	}, nil
}

// Cancel 取消训练任务
func (e *TrainingEngine) Cancel(ctx context.Context, sessionID string) (*CancelTrainingResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, exists := e.trainers[sessionID]
	if !exists {
		return nil, errors.New("training session not found")
	}

	if session.Status == StatusCompleted {
		return &CancelTrainingResponse{
			Success: false,
			Reason:  "session already completed",
		}, nil
	}

	if session.Status == StatusFailed {
		return &CancelTrainingResponse{
			Success: false,
			Reason:  "session already failed",
		}, nil
	}

	session.Status = StatusCancelled
	session.EndTime = time.Now()

	// 如果正在运行进程，终止它
	if session.CmdProc != nil {
		session.CmdProc.Kill()
		session.CmdProc.Wait()
	}

	return &CancelTrainingResponse{
		Success: true,
	}, nil
}

// fromMap 将 map 转换为 TrainingStats
func (e *TrainingEngine) fromMap(m map[string]interface{}) *TrainingStats {
	stats := &TrainingStats{}

	if val, ok := m["steps"].(float64); ok {
		stats.TrainingSteps = int64(val)
	}
	if val, ok := m["reward"].(float64); ok {
	 stats.TotalReward = val
	}
	if val, ok := m["episode_count"].(float64); ok {
		stats.EpisodeCount = int64(val)
	}
	if val, ok := m["best_reward"].(float64); ok {
		stats.BestReward = val
	}
	if val, ok := m["loss"].(float64); ok {
		stats.FinalLoss = val
	}
	if val, ok := m["validation_reward"].(float64); ok {
		stats.ValidationReward = val
	}
	if val, ok := m["model_path"].(string); ok {
		stats.ModelPath = val
	}

	return stats
}

// extractStatsFromText 从文本中提取训练统计
func (e *TrainingEngine) extractStatsFromText(text string) *TrainingStats {
	stats := &TrainingStats{}

	// 简单的文本提取（实际应该用正则）
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, "steps") {
			if parts := strings.Split(line, ":"); len(parts) >= 2 {
				if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					stats.TrainingSteps = int64(val)
				}
			}
		}
	}

	return stats
}

// GetTrainingHistory 获取训练历史
func (e *TrainingEngine) GetTrainingHistory(ctx context.Context, sessionID string) (*TrainingHistory, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	session, exists := e.trainers[sessionID]
	if !exists {
		return nil, errors.New("training session not found")
	}

	return &TrainingHistory{
		Sessions: []SessionSummary{
			{
				ID:        session.ID,
				Status:    string(session.Status),
				StartTime: session.StartTime,
				EndTime:   session.EndTime,
			},
		},
	}, nil
}

// DeleteSession 删除训练会话
func (e *TrainingEngine) DeleteSession(ctx context.Context, sessionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.trainers[sessionID]; exists {
		delete(e.trainers, sessionID)
	}

	// 清理输出文件
	outputFile := filepath.Join("/tmp/nautilus_training", sessionID+".json")
	if _, err := os.Stat(outputFile); err == nil {
		os.Remove(outputFile)
	}

	return nil
}
