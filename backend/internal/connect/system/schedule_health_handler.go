package system

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
)

// ScheduleHealthServer implements ant.v1.ScheduleHealthServiceHandler.
type ScheduleHealthServer struct {
	repo *repository.ScheduleHealthRepository
	log  *zap.Logger
}

var _ antv1c.ScheduleHealthServiceHandler = (*ScheduleHealthServer)(nil)

func NewScheduleHealthServer(repo *repository.ScheduleHealthRepository, log *zap.Logger) *ScheduleHealthServer {
	return &ScheduleHealthServer{repo: repo, log: log}
}

func (s *ScheduleHealthServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func computeGrade(successRate float64, failedRuns int, cfg repository.HealthGradingConfig) (level, color, noteCode string) {
	if failedRuns <= cfg.GreenMaxFailedRuns && successRate >= cfg.GreenSuccessRate {
		return "green", "#52c41a", "all_clear"
	}
	if successRate >= cfg.YellowSuccessRate {
		return "yellow", "#faad14", "degraded"
	}
	return "red", "#ff4d4f", "critical"
}

func ts(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func (s *ScheduleHealthServer) GetScheduleHealth(ctx context.Context, req *connect.Request[antv1.GetScheduleHealthRequest]) (*connect.Response[antv1.GetScheduleHealthResponse], error) {
	uid := s.userID(ctx)
	scheduleID, err := uuid.Parse(req.Msg.ScheduleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	runLimit := int(req.Msg.RunLimit)
	if runLimit <= 0 || runLimit > 100 {
		runLimit = 20
	}
	orderLimit := int(req.Msg.OrderLimit)
	if orderLimit <= 0 || orderLimit > 100 {
		orderLimit = 20
	}

	totalRuns, successRuns, failedRuns, successRate, lastRunAt, latestError, err :=
		s.repo.GetScheduleStats(ctx, uid, scheduleID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	latestOrderTicket, latestOrderProfit, hasLatestOrderProfit :=
		s.repo.GetLatestOrderProfit(ctx, uid, scheduleID)

	cfg := s.repo.GetGradingConfig(ctx)
	gradeLevel, gradeColor, gradeNoteCode := computeGrade(successRate, int(failedRuns), cfg)

	summary := &antv1.ScheduleHealthSummary{
		TotalRuns:            totalRuns,
		SuccessRuns:          successRuns,
		FailedRuns:           failedRuns,
		SuccessRate:          successRate,
		LastRunAt:            ts(lastRunAt),
		LatestError:          latestError,
		LatestOrderTicket:    strconv.FormatInt(latestOrderTicket, 10),
		LatestOrderProfit:    latestOrderProfit,
		HasLatestOrderProfit: hasLatestOrderProfit,
		GradeLevel:           gradeLevel,
		GradeColor:           gradeColor,
		GradeNoteCode:        gradeNoteCode,
		GreenSuccessRate:     cfg.GreenSuccessRate,
		GreenMaxFailedRuns:   int32(cfg.GreenMaxFailedRuns),
		YellowSuccessRate:    cfg.YellowSuccessRate,
		MinSampleSize:        int32(cfg.MinSampleSize),
	}

	runLogs, err := s.repo.ListRunLogs(ctx, uid, scheduleID, runLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	protoRunLogs := make([]*antv1.ScheduleHealthRunLog, 0, len(runLogs))
	for _, l := range runLogs {
		protoRunLogs = append(protoRunLogs, &antv1.ScheduleHealthRunLog{
			Id: l.ID, Status: l.Status, SignalType: l.SignalType,
			DurationMs: l.DurationMs, ErrorMessage: l.ErrorMessage,
			CreatedAt: timestamppb.New(l.CreatedAt),
		})
	}

	orders, err := s.repo.ListOrders(ctx, uid, scheduleID, orderLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	protoOrders := make([]*antv1.ScheduleHealthOrder, 0, len(orders))
	for _, o := range orders {
		protoOrders = append(protoOrders, &antv1.ScheduleHealthOrder{
			Id: o.ID, Ticket: o.Ticket, OrderType: o.OrderType,
			Symbol: o.Symbol, Profit: o.Profit,
			OpenTime: ts(o.OpenTime), CloseTime: ts(o.CloseTime),
		})
	}

	return connect.NewResponse(&antv1.GetScheduleHealthResponse{
		Summary: summary,
		RunLogs: protoRunLogs,
		Orders:  protoOrders,
	}), nil
}
