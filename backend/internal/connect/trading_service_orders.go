package connect

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/service"
)

func (s *TradingService) OrderSend(ctx context.Context, req *connect.Request[v1.OrderSendRequest]) (*connect.Response[v1.OrderSendResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	requestID := uuid.New().String()
	op := "order_send"
	idk := req.Msg.GetIdempotencyKey()
	if idk != "" {
		cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
		cached := &v1.OrderSendResponse{}
		if ok, err := s.getCachedProto(ctx, cacheKey, cached); err == nil && ok {
			return connect.NewResponse(cached), nil
		}
	}

	if b, err := protojson.Marshal(req.Msg); err == nil {
		s.publishTradeCommand(req.Msg.AccountId, &v1.TradeCommandEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Symbol:         req.Msg.Symbol,
			PayloadJson:    string(b),
		})
	}

	callCtx := service.WithTradeTriggerSource(ctx, service.TriggerSourceManual)
	result, err := s.tradingSvc.OrderSend(callCtx, uid, &service.OrderSendRequest{
		AccountID:  req.Msg.AccountId,
		Symbol:     req.Msg.Symbol,
		Type:       req.Msg.Type,
		Volume:     req.Msg.Volume,
		Price:      req.Msg.Price,
		StopLoss:   req.Msg.StopLoss,
		TakeProfit: req.Msg.TakeProfit,
		Comment:    req.Msg.Comment,
		Magic:      req.Msg.MagicNumber,
	})

	resp := &v1.OrderSendResponse{RequestId: requestID}
	if err != nil {
		errCode, errMessage, riskErr, isRisk := s.tradeErrorPayload(err)
		resp.Error = errCode
		resp.Retcode = 1
		resp.Message = errMessage
		resp.RiskError = toProtoRiskError(riskErr, map[string]interface{}{
			"request_id": requestID,
			"account_id": req.Msg.AccountId,
			"symbol":     req.Msg.Symbol,
			"order_type": req.Msg.Type,
			"volume":     req.Msg.Volume,
			"trigger_source": service.TradeTriggerSourceFromContext(ctx),
		})
		if isRisk {
			s.logTradeRiskValidation(ctx, uid, req.Msg.AccountId, op, requestID, "reject", errCode, errMessage, req.Msg.Symbol, req.Msg.Type, req.Msg.Volume)
		}
		s.publishTradeReceipt(req.Msg.AccountId, &v1.TradeReceiptEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Retcode:        resp.Retcode,
			Message:        resp.Message,
			Error:          resp.Error,
		})
		if idk != "" {
			cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
			s.setCachedProto(ctx, cacheKey, resp, 24*time.Hour)
		}
		return connect.NewResponse(resp), nil
	}

	resp.Order = convertOrderResponse(result, req.Msg.AccountId)
	resp.Retcode = 0
	resp.Message = "ok"
	s.logTradeRiskValidation(ctx, uid, req.Msg.AccountId, op, requestID, "pass", "OK", "", req.Msg.Symbol, req.Msg.Type, req.Msg.Volume)
	if resp.Order != nil {
		s.publishTradeReceipt(req.Msg.AccountId, &v1.TradeReceiptEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Retcode:        resp.Retcode,
			Message:        resp.Message,
			Ticket:         resp.Order.Ticket,
		})
	} else {
		s.publishTradeReceipt(req.Msg.AccountId, &v1.TradeReceiptEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Retcode:        resp.Retcode,
			Message:        resp.Message,
		})
	}
	if idk != "" {
		cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
		s.setCachedProto(ctx, cacheKey, resp, 24*time.Hour)
	}
	return connect.NewResponse(resp), nil
}

func (s *TradingService) OrderModify(ctx context.Context, req *connect.Request[v1.OrderModifyRequest]) (*connect.Response[v1.OrderModifyResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	requestID := uuid.New().String()
	op := "order_modify"
	idk := req.Msg.GetIdempotencyKey()
	if idk != "" {
		cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
		cached := &v1.OrderModifyResponse{}
		if ok, err := s.getCachedProto(ctx, cacheKey, cached); err == nil && ok {
			return connect.NewResponse(cached), nil
		}
	}

	if b, err := protojson.Marshal(req.Msg); err == nil {
		s.publishTradeCommand(req.Msg.AccountId, &v1.TradeCommandEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Symbol:         "",
			PayloadJson:    string(b),
		})
	}

	callCtx := service.WithTradeTriggerSource(ctx, service.TriggerSourceManual)
	result, err := s.tradingSvc.OrderModify(callCtx, uid, &service.OrderModifyRequest{
		AccountID:  req.Msg.AccountId,
		Ticket:     req.Msg.Ticket,
		StopLoss:   req.Msg.StopLoss,
		TakeProfit: req.Msg.TakeProfit,
		Price:      req.Msg.Price,
	})

	resp := &v1.OrderModifyResponse{RequestId: requestID}
	if err != nil {
		errCode, errMessage, riskErr, isRisk := s.tradeErrorPayload(err)
		resp.Error = errCode
		resp.Retcode = 1
		resp.Message = errMessage
		resp.RiskError = toProtoRiskError(riskErr, map[string]interface{}{
			"request_id": requestID,
			"account_id": req.Msg.AccountId,
			"ticket":     req.Msg.Ticket,
			"trigger_source": service.TradeTriggerSourceFromContext(ctx),
		})
		if isRisk {
			s.logTradeRiskValidation(ctx, uid, req.Msg.AccountId, op, requestID, "reject", errCode, errMessage, "", "", 0)
		}
		s.publishTradeReceipt(req.Msg.AccountId, &v1.TradeReceiptEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Retcode:        resp.Retcode,
			Message:        resp.Message,
			Error:          resp.Error,
			Ticket:         req.Msg.Ticket,
		})
		if idk != "" {
			cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
			s.setCachedProto(ctx, cacheKey, resp, 24*time.Hour)
		}
		return connect.NewResponse(resp), nil
	}

	resp.Order = convertOrderResponse(result, req.Msg.AccountId)
	resp.Retcode = 0
	resp.Message = "ok"
	s.logTradeRiskValidation(ctx, uid, req.Msg.AccountId, op, requestID, "pass", "OK", "", "", "", 0)
	s.publishTradeReceipt(req.Msg.AccountId, &v1.TradeReceiptEvent{
		RequestId:      requestID,
		IdempotencyKey: idk,
		Operation:      op,
		AccountId:      req.Msg.AccountId,
		Retcode:        resp.Retcode,
		Message:        resp.Message,
		Ticket:         req.Msg.Ticket,
	})
	if idk != "" {
		cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
		s.setCachedProto(ctx, cacheKey, resp, 24*time.Hour)
	}
	return connect.NewResponse(resp), nil
}

func (s *TradingService) OrderClose(ctx context.Context, req *connect.Request[v1.OrderCloseRequest]) (*connect.Response[v1.OrderCloseResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	requestID := uuid.New().String()
	op := "order_close"
	idk := req.Msg.GetIdempotencyKey()
	if idk != "" {
		cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
		cached := &v1.OrderCloseResponse{}
		if ok, err := s.getCachedProto(ctx, cacheKey, cached); err == nil && ok {
			return connect.NewResponse(cached), nil
		}
	}

	if b, err := protojson.Marshal(req.Msg); err == nil {
		s.publishTradeCommand(req.Msg.AccountId, &v1.TradeCommandEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Symbol:         "",
			PayloadJson:    string(b),
		})
	}

	callCtx := service.WithTradeTriggerSource(ctx, service.TriggerSourceManual)
	result, err := s.tradingSvc.OrderClose(callCtx, uid, &service.OrderCloseRequest{
		AccountID:   req.Msg.AccountId,
		Ticket:      req.Msg.Ticket,
		Volume:      req.Msg.Volume,
		CloseReason: req.Msg.CloseReason,
	})

	resp := &v1.OrderCloseResponse{RequestId: requestID}
	if err != nil {
		errCode, errMessage, riskErr, isRisk := s.tradeErrorPayload(err)
		resp.Error = errCode
		resp.Retcode = 1
		resp.Message = errMessage
		resp.RiskError = toProtoRiskError(riskErr, map[string]interface{}{
			"request_id": requestID,
			"account_id": req.Msg.AccountId,
			"ticket":     req.Msg.Ticket,
			"volume":     req.Msg.Volume,
			"close_reason": req.Msg.CloseReason,
			"trigger_source": service.TradeTriggerSourceFromContext(ctx),
		})
		if isRisk {
			s.logTradeRiskValidation(ctx, uid, req.Msg.AccountId, op, requestID, "reject", errCode, errMessage, "", "", req.Msg.Volume)
		}
		s.publishTradeReceipt(req.Msg.AccountId, &v1.TradeReceiptEvent{
			RequestId:      requestID,
			IdempotencyKey: idk,
			Operation:      op,
			AccountId:      req.Msg.AccountId,
			Retcode:        resp.Retcode,
			Message:        resp.Message,
			Error:          resp.Error,
			Ticket:         req.Msg.Ticket,
		})
		if idk != "" {
			cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
			s.setCachedProto(ctx, cacheKey, resp, 24*time.Hour)
		}
		return connect.NewResponse(resp), nil
	}

	resp.Order = convertOrderResponse(result, req.Msg.AccountId)
	resp.Retcode = 0
	resp.Message = "ok"
	s.logTradeRiskValidation(ctx, uid, req.Msg.AccountId, op, requestID, "pass", "OK", "", "", "", req.Msg.Volume)
	s.publishTradeReceipt(req.Msg.AccountId, &v1.TradeReceiptEvent{
		RequestId:      requestID,
		IdempotencyKey: idk,
		Operation:      op,
		AccountId:      req.Msg.AccountId,
		Retcode:        resp.Retcode,
		Message:        resp.Message,
		Ticket:         req.Msg.Ticket,
	})
	if idk != "" {
		cacheKey := tradeIdempotencyKey(req.Msg.AccountId, op, idk)
		s.setCachedProto(ctx, cacheKey, resp, 24*time.Hour)
	}
	return connect.NewResponse(resp), nil
}

func (s *TradingService) tradeErrorPayload(err error) (code string, message string, riskErr *service.RiskError, isRisk bool) {
	if riskErr, ok := service.AsRiskError(err); ok && riskErr != nil {
		msg := riskErr.Reason
		if msg == "" {
			msg = string(riskErr.Code)
		}
		return string(riskErr.Code), msg, riskErr, true
	}
	if errors.Is(err, service.ErrNoTradePermission) {
		re := &service.RiskError{
			Code:      service.RiskAccountTradeDisabled,
			Reason:    "account has no trade permission",
			Retryable: false,
		}
		return string(re.Code), re.Reason, re, true
	}
	return err.Error(), err.Error(), nil, false
}

func (s *TradingService) logTradeRiskValidation(
	ctx context.Context,
	userID uuid.UUID,
	accountID string,
	action string,
	requestID string,
	result string,
	riskCode string,
	riskMessage string,
	symbol string,
	orderType string,
	volume float64,
) {
	if s.logSvc == nil {
		return
	}
	accountUUID, err := uuid.Parse(accountID)
	if err != nil {
		return
	}
	log := model.NewSystemOperationLog(userID, model.OperationTypeUpdate, "trading_risk", "pre_trade_validate")
	log.ResourceType = "mt_account"
	log.ResourceID = accountUUID
	if result == "pass" {
		log.Status = model.OperationStatusSuccess
	} else {
		log.Status = model.OperationStatusFailed
		log.ErrorMessage = riskMessage
	}
	log.NewValue = map[string]interface{}{
		"request_id": requestID,
		"account_id": accountID,
		"action":     action,
		"result":     result,
		"risk_code":  riskCode,
		"message":    riskMessage,
		"symbol":     symbol,
		"order_type": orderType,
		"volume":     volume,
		"trigger_source": service.TradeTriggerSourceFromContext(ctx),
	}
	logCtx := ctx
	if logCtx == nil || logCtx.Err() != nil {
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		logCtx = bg
	}
	_ = s.logSvc.LogOperation(logCtx, log)
}

func toProtoRiskError(riskErr *service.RiskError, contextMap map[string]interface{}) *v1.RiskError {
	if riskErr == nil {
		return nil
	}
	ctxJSON := ""
	if len(contextMap) > 0 {
		if b, err := json.Marshal(contextMap); err == nil {
			ctxJSON = string(b)
		}
	}
	return &v1.RiskError{
		Code:       string(riskErr.Code),
		Reason:     riskErr.Reason,
		UserMessage: riskErr.UserMessage,
		Retryable:  riskErr.Retryable,
		ContextJson: ctxJSON,
	}
}
