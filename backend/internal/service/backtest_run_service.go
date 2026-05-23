package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/lib/pq"

	"anttrader/internal/repository"
)

type BacktestRunRepo interface {
	Create(ctx context.Context, run *repository.BacktestRun) (uuid.UUID, error)
	GetByID(ctx context.Context, userID, runID uuid.UUID) (*repository.BacktestRun, error)
	ListByUser(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, limit, offset int) ([]*repository.BacktestRun, error)
	UpdateAsyncFields(ctx context.Context, userID, runID uuid.UUID, status string, errMsg string, startedAt, finishedAt *time.Time, metrics, equityCurve []byte) error
	ClaimNextForWork(ctx context.Context, leaseUntil time.Time) (*repository.BacktestRun, error)
	ExtendLease(ctx context.Context, userID, runID uuid.UUID, leaseUntil time.Time) error
	RequestCancel(ctx context.Context, userID, runID uuid.UUID) error
	Delete(ctx context.Context, userID, runID uuid.UUID) (bool, error)
	CountActiveByUser(ctx context.Context, userID uuid.UUID) (int, error)
	CountPendingByUser(ctx context.Context, userID uuid.UUID) (int, error)
	CountRecentStartsByUser(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	CountActiveByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error)
	CountPendingByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error)
	GetStatusAndCancelRequestedAt(ctx context.Context, userID, runID uuid.UUID) (string, *time.Time, error)
}

type BacktestRunService struct {
	repo BacktestRunRepo
}

func NewBacktestRunService(repo BacktestRunRepo) *BacktestRunService {
	return &BacktestRunService{repo: repo}
}

type BacktestRunRecordRequest struct {
	UserID            uuid.UUID
	AccountID         uuid.UUID
	Symbol            string
	Timeframe         string
	DatasetID         *uuid.UUID
	StrategyCode      string
	PythonSvcVersion  string
	CostModel         *BacktestCostModel
	Metrics           *BacktestMetricsPython
	EquityCurve       []float64
}

func (s *BacktestRunService) Record(ctx context.Context, req *BacktestRunRecordRequest) (uuid.UUID, error) {
	if req == nil {
		return uuid.Nil, nil
	}
	if s == nil || s.repo == nil {
		return uuid.Nil, nil
	}

	ver := req.PythonSvcVersion
	if ver == "" {
		ver = os.Getenv("ANTRADER_PYTHON_SERVICE_VERSION")
	}
	var verPtr *string
	if ver != "" {
		verPtr = &ver
	}

	costJSON, _ := json.Marshal(req.CostModel)
	metricsJSON, _ := json.Marshal(req.Metrics)
	equityJSON, _ := json.Marshal(req.EquityCurve)
	now := time.Now()

	run := &repository.BacktestRun{
		ID:                  uuid.New(),
		UserID:              req.UserID,
		AccountID:           req.AccountID,
		Symbol:              req.Symbol,
		Timeframe:           req.Timeframe,
		DatasetID:           req.DatasetID,
		StrategyCodeHash:    hashStrategyCode(req.StrategyCode),
		PythonServiceVersion: verPtr,
		CostModelSnapshot:   costJSON,
		Metrics:             metricsJSON,
		EquityCurve:         equityJSON,
		Status:              "SUCCEEDED",
		Error:               "",
		StartedAt:           &now,
		FinishedAt:          &now,
	}
	return s.repo.Create(ctx, run)
}

type CreateBacktestRunRequest struct {
	UserID         uuid.UUID
	AccountID      uuid.UUID
	Symbol         string
	Timeframe      string
	DatasetID      *uuid.UUID
	Mode           string
	FromTs         *time.Time
	ToTs           *time.Time
	StrategyCode   string
	InitialCapital float64
	TemplateID      *uuid.UUID
	TemplateDraftID *uuid.UUID
	ExtraSymbols    []string
}

func (s *BacktestRunService) CreatePending(ctx context.Context, req *CreateBacktestRunRequest) (uuid.UUID, error) {
	if req == nil {
		return uuid.Nil, errors.New("nil request")
	}
	if s == nil || s.repo == nil {
		return uuid.Nil, errors.New("service not initialized")
	}
	mode := req.Mode
	if mode == "" {
		mode = "KLINE_RANGE"
	}
	code := req.StrategyCode
	cap0 := req.InitialCapital
	now := time.Now()

	// Ensure ExtraSymbols is never nil to satisfy NOT NULL constraint on column
	extraSyms := req.ExtraSymbols
	if extraSyms == nil {
		extraSyms = []string{}
	}

	run := &repository.BacktestRun{
		ID:               uuid.New(),
		UserID:           req.UserID,
		AccountID:        req.AccountID,
		Symbol:           req.Symbol,
		Timeframe:        req.Timeframe,
		DatasetID:        req.DatasetID,
		TemplateID:       req.TemplateID,
		TemplateDraftID:  req.TemplateDraftID,
		Mode:             mode,
		FromTs:           req.FromTs,
		ToTs:             req.ToTs,
		StrategyCodeHash: hashStrategyCode(req.StrategyCode),
		Status:           "PENDING",
		Error:            "",
		StartedAt:        nil,
		FinishedAt:       nil,
		StrategyCode:     &code,
		InitialCapital:   &cap0,
		ExtraSymbols:     pq.StringArray(extraSyms),
		Metrics:          []byte("null"),
		EquityCurve:      []byte("null"),
		CreatedAt:        now,
	}
	return s.repo.Create(ctx, run)
}

func (s *BacktestRunService) MarkRunning(ctx context.Context, userID, runID uuid.UUID) error {
	now := time.Now()
	return s.repo.UpdateAsyncFields(ctx, userID, runID, "RUNNING", "", &now, nil, nil, nil)
}

func (s *BacktestRunService) MarkSucceeded(ctx context.Context, userID, runID uuid.UUID, metrics, equityCurve []byte) error {
	now := time.Now()
	return s.repo.UpdateAsyncFields(ctx, userID, runID, "SUCCEEDED", "", nil, &now, metrics, equityCurve)
}

func (s *BacktestRunService) MarkFailed(ctx context.Context, userID, runID uuid.UUID, errMsg string) error {
	now := time.Now()
	if errMsg == "" {
		errMsg = "unknown error"
	}
	return s.repo.UpdateAsyncFields(ctx, userID, runID, "FAILED", errMsg, nil, &now, nil, nil)
}

func (s *BacktestRunService) MarkCanceled(ctx context.Context, userID, runID uuid.UUID, errMsg string) error {
	now := time.Now()
	return s.repo.UpdateAsyncFields(ctx, userID, runID, "CANCELED", errMsg, nil, &now, nil, nil)
}

func (s *BacktestRunService) Get(ctx context.Context, userID, runID uuid.UUID) (*repository.BacktestRun, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("service not initialized")
	}
	return s.repo.GetByID(ctx, userID, runID)
}

func (s *BacktestRunService) List(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, limit, offset int) ([]*repository.BacktestRun, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("service not initialized")
	}
	return s.repo.ListByUser(ctx, userID, accountID, limit, offset)
}

func (s *BacktestRunService) RequestCancel(ctx context.Context, userID, runID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return errors.New("service not initialized")
	}
	return s.repo.RequestCancel(ctx, userID, runID)
}

func (s *BacktestRunService) Delete(ctx context.Context, userID, runID uuid.UUID) (bool, error) {
	if s == nil || s.repo == nil {
		return false, errors.New("service not initialized")
	}
	return s.repo.Delete(ctx, userID, runID)
}

func (s *BacktestRunService) ClaimNextForWork(ctx context.Context, leaseUntil time.Time) (*repository.BacktestRun, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("service not initialized")
	}
	run, err := s.repo.ClaimNextForWork(ctx, leaseUntil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return run, nil
}

func (s *BacktestRunService) ExtendLease(ctx context.Context, userID, runID uuid.UUID, leaseUntil time.Time) error {
	if s == nil || s.repo == nil {
		return errors.New("service not initialized")
	}
	return s.repo.ExtendLease(ctx, userID, runID, leaseUntil)
}

func (s *BacktestRunService) CountActiveByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("service not initialized")
	}
	return s.repo.CountActiveByUser(ctx, userID)
}

func (s *BacktestRunService) CountPendingByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("service not initialized")
	}
	return s.repo.CountPendingByUser(ctx, userID)
}

func (s *BacktestRunService) CountRecentStartsByUser(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("service not initialized")
	}
	return s.repo.CountRecentStartsByUser(ctx, userID, since)
}

func (s *BacktestRunService) CountActiveByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("service not initialized")
	}
	return s.repo.CountActiveByAccount(ctx, userID, accountID)
}

func (s *BacktestRunService) CountPendingByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("service not initialized")
	}
	return s.repo.CountPendingByAccount(ctx, userID, accountID)
}

func (s *BacktestRunService) GetStatusAndCancelRequestedAt(ctx context.Context, userID, runID uuid.UUID) (string, *time.Time, error) {
	if s == nil || s.repo == nil {
		return "", nil, errors.New("service not initialized")
	}
	return s.repo.GetStatusAndCancelRequestedAt(ctx, userID, runID)
}

func hashStrategyCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
