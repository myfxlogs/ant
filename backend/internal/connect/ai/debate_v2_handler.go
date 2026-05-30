package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	debateV2 "anttrader/internal/service"
)

// DebateV2Server implements ant.v1.DebateV2ServiceHandler.
type DebateV2Server struct {
	svc *debateV2.DebateV2Service
	log *zap.Logger
}

var _ antv1c.DebateV2ServiceHandler = (*DebateV2Server)(nil)

func NewDebateV2Server(svc *debateV2.DebateV2Service, log *zap.Logger) *DebateV2Server {
	return &DebateV2Server{svc: svc, log: log}
}

func (h *DebateV2Server) StartDebateV2(ctx context.Context, req *connect.Request[antv1.StartDebateV2Request]) (*connect.Response[antv1.DebateV2Session], error) {
	h.log.Info("StartDebateV2: entering handler")
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.Agents) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("agents must not be empty"))
	}
	h.log.Info("StartDebateV2: calling Start", zap.String("user_id", uid.String()))
	sess, err := h.svc.Start(ctx, uid, req.Msg.Agents, req.Msg.Title)
	if err != nil {
		h.log.Error("StartDebateV2 failed", zap.Strings("agents", req.Msg.Agents), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.log.Info("StartDebateV2 ok", zap.String("session_id", fmt.Sprintf("%v", sess)))
	rsp := toProtoSession(sess)
	h.log.Info("StartDebateV2 proto ok")
	return connect.NewResponse(rsp), nil
}

func (h *DebateV2Server) ChatDebateV2(ctx context.Context, req *connect.Request[antv1.ChatDebateV2Request]) (*connect.Response[antv1.DebateV2Session], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := h.svc.Chat(ctx, req.Msg.SessionId, uid, req.Msg.Message)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProtoSession(sess)), nil
}

func (h *DebateV2Server) AdvanceDebateV2(ctx context.Context, req *connect.Request[antv1.DebateV2SessionRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := h.svc.Advance(ctx, req.Msg.SessionId, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProtoSession(sess)), nil
}

func (h *DebateV2Server) StartDebateV2AdvanceJob(ctx context.Context, req *connect.Request[antv1.DebateV2SessionRequest]) (*connect.Response[antv1.StartDebateV2AdvanceJobResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sid := req.Msg.SessionId
	if _, err := h.svc.Advance(ctx, sid, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	jobID, err := h.svc.PrepareChatJob(ctx, sid, uid, "advance")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.svc.RunChatJob(jobID, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.StartDebateV2AdvanceJobResponse{
		JobId: jobID.String(), SessionId: sid,
	}), nil
}

func (h *DebateV2Server) GetDebateV2AdvanceJob(ctx context.Context, req *connect.Request[antv1.GetDebateV2AdvanceJobRequest]) (*connect.Response[antv1.GetDebateV2AdvanceJobResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	jid, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid job id"))
	}
	phase, msg, err := h.svc.GetJob(ctx, jid, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetDebateV2AdvanceJobResponse{
		Phase: phase, Message: msg,
	}), nil
}

func (h *DebateV2Server) PrepareDebateV2ChatJob(ctx context.Context, req *connect.Request[antv1.ChatDebateV2Request]) (*connect.Response[antv1.StartDebateV2ChatJobResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := h.svc.PrepareChatJob(ctx, req.Msg.SessionId, uid, req.Msg.Message)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.StartDebateV2ChatJobResponse{
		JobId: jobID.String(), SessionId: req.Msg.SessionId,
	}), nil
}

func (h *DebateV2Server) RunDebateV2ChatJob(ctx context.Context, req *connect.Request[antv1.RunDebateV2ChatJobRequest]) (*connect.Response[emptypb.Empty], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	jid, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid job id"))
	}
	if err := h.svc.RunChatJob(jid, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (h *DebateV2Server) GetDebateV2ChatJob(ctx context.Context, req *connect.Request[antv1.GetDebateV2ChatJobRequest]) (*connect.Response[antv1.GetDebateV2ChatJobResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	jid, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid job id"))
	}
	phase, msg, err := h.svc.GetJob(ctx, jid, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetDebateV2ChatJobResponse{
		Phase: phase, Message: msg,
	}), nil
}

func (h *DebateV2Server) BackDebateV2(ctx context.Context, req *connect.Request[antv1.DebateV2SessionRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := h.svc.Back(ctx, req.Msg.SessionId, uid)
	if err != nil {
		h.log.Error("BackDebateV2 failed", zap.String("session", req.Msg.SessionId), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProtoSession(sess)), nil
}

func (h *DebateV2Server) SetDebateV2Params(ctx context.Context, req *connect.Request[antv1.SetDebateV2ParamsRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	paramsJSON, err := json.Marshal(req.Msg.Params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal params: %w", err))
	}
	sess, err := h.svc.SetParams(ctx, req.Msg.SessionId, uid, string(paramsJSON))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProtoSession(sess)), nil
}

func (h *DebateV2Server) RejectDebateV2Code(ctx context.Context, req *connect.Request[antv1.RejectDebateV2CodeRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := h.svc.Chat(ctx, req.Msg.SessionId, uid, req.Msg.Feedback)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProtoSession(sess)), nil
}

func (h *DebateV2Server) StartDebateV2RejectCodeJob(ctx context.Context, req *connect.Request[antv1.RejectDebateV2CodeRequest]) (*connect.Response[antv1.StartDebateV2AdvanceJobResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sid := req.Msg.SessionId
	if _, err := h.svc.Back(ctx, sid, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	jobID, err := h.svc.PrepareChatJob(ctx, sid, uid, req.Msg.Feedback)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.svc.RunChatJob(jobID, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.StartDebateV2AdvanceJobResponse{
		JobId: jobID.String(), SessionId: sid,
	}), nil
}

func (h *DebateV2Server) ListDebateV2Sessions(ctx context.Context, req *connect.Request[antv1.ListDebateV2SessionsRequest]) (*connect.Response[antv1.ListDebateV2SessionsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	sessions, err := h.svc.List(ctx, uid, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbSessions := make([]*antv1.DebateV2Session, 0, len(sessions))
	for _, s := range sessions {
		pbSessions = append(pbSessions, toProtoSession(s))
	}
	return connect.NewResponse(&antv1.ListDebateV2SessionsResponse{Sessions: pbSessions}), nil
}

func (h *DebateV2Server) GetDebateV2Session(ctx context.Context, req *connect.Request[antv1.GetDebateV2SessionRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := h.svc.Get(ctx, req.Msg.SessionId, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProtoSession(sess)), nil
}

func (h *DebateV2Server) DeleteDebateV2Session(ctx context.Context, req *connect.Request[antv1.DeleteDebateV2SessionRequest]) (*connect.Response[emptypb.Empty], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, req.Msg.SessionId, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}
