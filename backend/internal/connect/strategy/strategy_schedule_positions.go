// strategy_schedule_positions.go — GetSchedulePositions RPC: per-strategy open
// positions (live = position snapshot filtered by schedule magic; paper = open
// paper orders). Extracted from strategy_schedules.go per file-size limits.

package strategy

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/service"
)

func (s *StrategyServer) GetSchedulePositions(ctx context.Context, req *connect.Request[antv1.GetSchedulePositionsRequest]) (*connect.Response[antv1.GetSchedulePositionsResponse], error) {
	id, err := uuid.Parse(req.Msg.ScheduleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	uid := s.userID(ctx)
	row, err := s.svc.GetSchedule(ctx, id, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	expectedMagic := model.StrategyMagic(id)
	if row.MagicNumber != nil {
		expectedMagic = *row.MagicNumber
	}

	// Live path: last persisted position snapshot filtered by schedule magic.
	var raw []byte
	err = s.svc.DB().QueryRow(ctx,
		`SELECT payload_proto FROM mt_position_snapshots WHERE account_id = $1`,
		row.AccountID.String(),
	).Scan(&raw)

	if err == nil && len(raw) > 0 {
		record := &antv1.MtPositionSnapshotRecord{}
		if err := proto.Unmarshal(raw, record); err == nil {
			positions := make([]*antv1.MtPositionSnapshotItem, 0, len(record.GetPositions()))
			for _, pos := range record.GetPositions() {
				if pos.GetMagicNumber() == int64(expectedMagic) {
					positions = append(positions, pos)
				}
			}
			return connect.NewResponse(&antv1.GetSchedulePositionsResponse{Positions: positions}), nil
		}
	}

	// Paper path: open paper orders for the schedule's account + symbol.
	return s.paperSchedulePositions(ctx, row)
}

func (s *StrategyServer) paperSchedulePositions(ctx context.Context, row *service.ScheduleRow) (*connect.Response[antv1.GetSchedulePositionsResponse], error) {
	rows, err := s.svc.DB().Query(ctx,
		`SELECT id, symbol, side, volume, fill_price, pnl, created_at
		 FROM paper_orders
		 WHERE paper_account_id = $1 AND symbol = $2 AND state = 'open'
		 ORDER BY created_at DESC`,
		row.AccountID.String(), row.Symbol,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var positions []*antv1.MtPositionSnapshotItem
	for rows.Next() {
		var oid string
		var symbol, side string
		var volume, fillPrice, pnl decimal.Decimal
		var createdAt time.Time
		if err := rows.Scan(&oid, &symbol, &side, &volume, &fillPrice, &pnl, &createdAt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		_ = oid
		positions = append(positions, &antv1.MtPositionSnapshotItem{
			Ticket:       0,
			Symbol:       symbol,
			Type:         side,
			MagicNumber:  0,
			Volume:       volume.String(),
			OpenPrice:    fillPrice.String(),
			CurrentPrice: fillPrice.String(),
			Profit:       pnl.String(),
			OpenTime:     createdAt.Unix(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetSchedulePositionsResponse{Positions: positions}), nil
}

// scheduleRowToProto converts a schedule row to proto, enriching with live
// session state from the registry (is_running/active_run_id/signal_count).
func (s *StrategyServer) scheduleRowToProto(r *service.ScheduleRow) *antv1.StrategySchedule {
	return buildScheduleProto(r, s.sessionRegistry)
}

func buildScheduleProto(r *service.ScheduleRow, reg *SessionRegistry) *antv1.StrategySchedule {
	if r == nil {
		return nil
	}
	s := &antv1.StrategySchedule{
		Id:           r.ID.String(),
		UserId:       r.UserID.String(),
		AccountId:    r.AccountID.String(),
		Name:         r.Name,
		Symbol:       r.Symbol,
		Timeframe:    r.Timeframe,
		ScheduleType: r.ScheduleType,
		IsActive:     r.IsActive,
		RunCount:     r.RunCount,
		LastError:    r.LastError,
		EnableCount:  r.EnableCount,
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
	}
	if r.TemplateID != uuid.Nil {
		s.TemplateId = r.TemplateID.String()
	}
	if r.RiskScore != nil {
		s.RiskScore = *r.RiskScore
	}
	if r.RiskLevel != "" {
		s.RiskLevel = r.RiskLevel
	}
	if len(r.Parameters) > 0 {
		var params antv1.StrategyParams
		if err := proto.Unmarshal(r.Parameters, &params); err == nil {
			s.Parameters = params.Values
		}
	}
	if len(r.ScheduleConfig) > 0 {
		var cfg antv1.ScheduleConfig
		if err := proto.Unmarshal(r.ScheduleConfig, &cfg); err == nil {
			s.ScheduleConfig = &cfg
		}
	}
	if len(r.BacktestMetrics) > 0 {
		var metrics antv1.BacktestMetrics
		if err := proto.Unmarshal(r.BacktestMetrics, &metrics); err == nil {
			s.BacktestMetrics = &metrics
		}
	}
	if len(r.RiskReasons) > 0 {
		var risk antv1.BacktestRisk
		if err := proto.Unmarshal(r.RiskReasons, &risk); err == nil {
			s.RiskReasons = risk.Reasons
		}
	}
	if len(r.RiskWarnings) > 0 {
		var risk antv1.BacktestRisk
		if err := proto.Unmarshal(r.RiskWarnings, &risk); err == nil {
			s.RiskWarnings = risk.Warnings
		}
	}
	if r.LastRunAt != nil {
		s.LastRunAt = timestamppb.New(*r.LastRunAt)
	}
	if r.NextRunAt != nil {
		s.NextRunAt = timestamppb.New(*r.NextRunAt)
	}
	if reg != nil {
		if sess, ok := reg.GetByScheduleID(r.ID); ok {
			s.IsRunning = true
			s.ActiveRunId = sess.RunID.String()
			s.SignalCount = int32(sess.SignalCount)
		}
	}
	return s
}

func scheduleParamsToProto(params map[string]string) []byte {
	data, _ := proto.Marshal(&antv1.StrategyParams{Values: params})
	return data
}

func stringListToProto(list []string) []byte {
	data, _ := proto.Marshal(&antv1.BacktestRisk{Reasons: list})
	return data
}
