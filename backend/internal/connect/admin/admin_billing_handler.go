package admin

import (
	"context"
	"database/sql"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/repository"
)

type AdminBillingServer struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

var _ antv1c.AdminBillingServiceHandler = (*AdminBillingServer)(nil)

func NewAdminBillingServer(repo *repository.AdminRepository, log *zap.Logger) *AdminBillingServer {
	return &AdminBillingServer{repo: repo, log: log}
}

func (s *AdminBillingServer) ListSubscriptions(ctx context.Context, req *connect.Request[antv1.ListAdminSubscriptionsRequest]) (*connect.Response[antv1.ListAdminSubscriptionsResponse], error) {
	page := int(req.Msg.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	query := `SELECT ups.id, ups.user_id, u.email, sp.name, sp.display_name,
	                 ups.status, ups.billing_cycle, ups.current_period_start,
	                 ups.current_period_end, ups.auto_renew, sp.price_monthly::text, ups.created_at
	          FROM user_platform_subscriptions ups
		  JOIN users u ON u.id = ups.user_id
		  JOIN subscription_plans sp ON sp.id = ups.plan_id`
	args := []interface{}{}
	argIdx := 1

	where := ""
	if req.Msg.Plan != "" {
		where = fmt.Sprintf(" WHERE sp.name = $%d", argIdx)
		args = append(args, req.Msg.Plan)
		argIdx++
	}
	if req.Msg.Status != "" {
		if where == "" {
			where = fmt.Sprintf(" WHERE ups.status = $%d", argIdx)
		} else {
			where += fmt.Sprintf(" AND ups.status = $%d", argIdx)
		}
		args = append(args, req.Msg.Status)
		argIdx++
	}
	query += where + fmt.Sprintf(" ORDER BY ups.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.repo.DB().Query(ctx, query, args...)
	if err != nil {
		s.log.Error("ListSubscriptions: query failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query failed"))
	}
	defer rows.Close()

	var subs []*antv1.AdminSubscriptionDetail
	for rows.Next() {
		var sub antv1.AdminSubscriptionDetail
		var periodStart, periodEnd, createdAt sql.NullTime
		var autoRenew bool
		err := rows.Scan(
			&sub.Id, &sub.UserId, &sub.UserEmail, &sub.PlanName, &sub.PlanDisplayName,
			&sub.Status, &sub.BillingCycle, &periodStart, &periodEnd, &autoRenew,
			&sub.Price, &createdAt,
		)
		if err != nil {
			s.log.Error("ListSubscriptions: scan failed", zap.Error(err))
			continue
		}
		sub.AutoRenew = autoRenew
		if periodStart.Valid {
			sub.CurrentPeriodStart = timestamppb.New(periodStart.Time)
		}
		if periodEnd.Valid {
			sub.CurrentPeriodEnd = timestamppb.New(periodEnd.Time)
		}
		if createdAt.Valid {
			sub.CreatedAt = timestamppb.New(createdAt.Time)
		}
		subs = append(subs, &sub)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM user_platform_subscriptions ups JOIN subscription_plans sp ON sp.id = ups.plan_id" + where
	countArgs := args[:argIdx-1]
	var total int64
	_ = s.repo.DB().QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	return connect.NewResponse(&antv1.ListAdminSubscriptionsResponse{
		Subscriptions: subs,
		Total:         total,
	}), nil
}

func (s *AdminBillingServer) GetRevenueSummary(ctx context.Context, req *connect.Request[antv1.GetRevenueSummaryRequest]) (*connect.Response[antv1.GetRevenueSummaryResponse], error) {
	rows, err := s.repo.DB().Query(ctx,
		`SELECT sp.name, sp.display_name,
		        (SELECT COUNT(*) FROM user_platform_subscriptions ups
		         WHERE ups.plan_id = sp.id AND ups.status = 'active'),
		        (SELECT COALESCE(SUM(sp2.price_monthly), 0)::text
		         FROM user_platform_subscriptions ups2
		         JOIN subscription_plans sp2 ON sp2.id = ups2.plan_id
		         WHERE ups2.plan_id = sp.id AND ups2.status = 'active'),
		        (SELECT COALESCE(SUM(-wt.amount::numeric), 0)::text
		         FROM wallet_transactions wt
		         WHERE wt.tx_type = 'purchase'
		           AND wt.description LIKE 'Platform subscription: ' || sp.display_name || ' (%')
		 FROM subscription_plans sp
		 WHERE sp.is_active = true
		 ORDER BY sp.sort_order`)
	if err != nil {
		s.log.Error("GetRevenueSummary: query failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query failed"))
	}
	defer rows.Close()

	var plans []*antv1.PlanRevenue
	var totalMonthly, totalRev string
	for rows.Next() {
		var p antv1.PlanRevenue
		if err := rows.Scan(&p.PlanName, &p.DisplayName, &p.ActiveCount, &p.MonthlyRevenue, &p.TotalRevenue); err != nil {
			continue
		}
		plans = append(plans, &p)
	}

	// Totals
	_ = s.repo.DB().QueryRow(ctx,
		`SELECT COALESCE(SUM(sp.price_monthly), 0)::text
		 FROM user_platform_subscriptions ups
		 JOIN subscription_plans sp ON sp.id = ups.plan_id
		 WHERE ups.status = 'active'`,
	).Scan(&totalMonthly)
	_ = s.repo.DB().QueryRow(ctx,
		`SELECT COALESCE(SUM(-amount::numeric), 0)::text FROM wallet_transactions
		 WHERE tx_type = 'purchase' AND description LIKE 'Platform subscription:%'`,
	).Scan(&totalRev)

	return connect.NewResponse(&antv1.GetRevenueSummaryResponse{
		Plans:               plans,
		TotalMonthlyRevenue: totalMonthly,
		TotalRevenue:        totalRev,
	}), nil
}

func (s *AdminBillingServer) ListAdminWalletTransactions(ctx context.Context, req *connect.Request[antv1.ListAdminWalletTransactionsRequest]) (*connect.Response[antv1.ListAdminWalletTransactionsResponse], error) {
	page := int(req.Msg.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	query := `SELECT wt.id, wt.user_id, u.email, wt.tx_type, wt.amount::text,
	                 wt.balance_before::text, wt.balance_after::text, wt.description, wt.created_at
	          FROM wallet_transactions wt
		  JOIN users u ON u.id = wt.user_id`
	args := []interface{}{}
	argIdx := 1

	where := ""
	if req.Msg.TxType != "" {
		where = fmt.Sprintf(" WHERE wt.tx_type = $%d", argIdx)
		args = append(args, req.Msg.TxType)
		argIdx++
	}
	if req.Msg.UserId != "" {
		if _, err := uuid.Parse(req.Msg.UserId); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user_id"))
		}
		if where == "" {
			where = fmt.Sprintf(" WHERE wt.user_id = $%d", argIdx)
		} else {
			where += fmt.Sprintf(" AND wt.user_id = $%d", argIdx)
		}
		args = append(args, req.Msg.UserId)
		argIdx++
	}
	query += where + fmt.Sprintf(" ORDER BY wt.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.repo.DB().Query(ctx, query, args...)
	if err != nil {
		s.log.Error("ListAdminWalletTransactions: query failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query failed"))
	}
	defer rows.Close()

	var txs []*antv1.AdminWalletTransactionDetail
	for rows.Next() {
		var tx antv1.AdminWalletTransactionDetail
		var createdAt sql.NullTime
		if err := rows.Scan(
			&tx.Id, &tx.UserId, &tx.UserEmail, &tx.TxType, &tx.Amount,
			&tx.BalanceBefore, &tx.BalanceAfter, &tx.Description, &createdAt,
		); err != nil {
			continue
		}
		if createdAt.Valid {
			tx.CreatedAt = timestamppb.New(createdAt.Time)
		}
		txs = append(txs, &tx)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM wallet_transactions wt" + where
	countArgs := args[:argIdx-1]
	var total int64
	_ = s.repo.DB().QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	return connect.NewResponse(&antv1.ListAdminWalletTransactionsResponse{
		Transactions: txs,
		Total:        total,
	}), nil
}
