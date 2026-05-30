package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
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

func (h *DebateV2Server) userID(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	return id, nil
}

func (h *DebateV2Server) StartDebateV2(ctx context.Context, req *connect.Request[antv1.StartDebateV2Request]) (*connect.Response[antv1.DebateV2Session], error) {
	h.log.Info("StartDebateV2: entering handler")
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
		h.log.Error("StartDebateV2AdvanceJob: RunChatJob failed", zap.String("job_id", jobID.String()), zap.Error(err))
	}
	return connect.NewResponse(&antv1.StartDebateV2AdvanceJobResponse{
		JobId: jobID.String(), SessionId: sid,
	}), nil
}

func (h *DebateV2Server) GetDebateV2AdvanceJob(ctx context.Context, req *connect.Request[antv1.GetDebateV2AdvanceJobRequest]) (*connect.Response[antv1.GetDebateV2AdvanceJobResponse], error) {
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
		h.log.Error("StartDebateV2RejectCodeJob: RunChatJob failed", zap.String("job_id", jobID.String()), zap.Error(err))
	}
	return connect.NewResponse(&antv1.StartDebateV2AdvanceJobResponse{
		JobId: jobID.String(), SessionId: sid,
	}), nil
}

func (h *DebateV2Server) ListDebateV2Sessions(ctx context.Context, req *connect.Request[antv1.ListDebateV2SessionsRequest]) (*connect.Response[antv1.ListDebateV2SessionsResponse], error) {
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
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
	uid, err := h.userID(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, req.Msg.SessionId, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// --- SSE HTTP handlers ---

func (h *DebateV2Server) HandleDebateV2AdvanceJobSSE(w http.ResponseWriter, r *http.Request, authInterceptor *interceptor.AuthInterceptor) {
	h.handleJobSSE(w, r, authInterceptor)
}

func (h *DebateV2Server) HandleDebateV2ChatJobSSE(w http.ResponseWriter, r *http.Request, authInterceptor *interceptor.AuthInterceptor) {
	h.handleJobSSE(w, r, authInterceptor)
}

func (h *DebateV2Server) handleJobSSE(w http.ResponseWriter, r *http.Request, authInterceptor *interceptor.AuthInterceptor) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sse/debate-v2/"), "/")
	if len(parts) < 3 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	jobID, err := uuid.Parse(parts[1])
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	userID, err := authInterceptor.UserIDFromHTTP(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ch, err := h.svc.JobChannel(jobID, userID)
	if err != nil {
		http.Error(w, `{"error":"job not running"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				h.log.Error("handleJobSSE: json marshal failed", zap.Error(err))
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
			if ev.Phase == "completed" || ev.Phase == "failed" {
				return
			}
		default:
			// Don't block if neither channel nor context is ready.
		}
	}
}

// --- proto conversion ---

func toProtoSession(sess *debateV2.V2Session) *antv1.DebateV2Session {
	if sess == nil {
		return &antv1.DebateV2Session{}
	}
	pb := &antv1.DebateV2Session{
		Id:          sess.ID,
		Title:       sess.Title,
		Status:      sess.Status,
		CurrentStep: sess.CurrentStep,
		Agents:      sess.Agents,
		Provider:    sess.Provider,
		Model:       sess.Model,
		CreatedAt:   sess.CreatedAt,
		UpdatedAt:   sess.UpdatedAt,
	}
	for _, step := range sess.Steps {
		s := &antv1.DebateV2Step{
			StepKey:   step.StepKey,
			AgentKey:  step.AgentKey,
			AgentName: step.AgentName,
		}
		for _, msg := range step.Messages {
			s.Messages = append(s.Messages, &antv1.DebateV2Message{
				Id:      msg.ID,
				Role:    msg.Role,
				Content: msg.Content,
				Kind:    msg.Kind,
			})
		}
		pb.Steps = append(pb.Steps, s)
	}
	if sess.Code != nil {
		pb.Code = &antv1.DebateV2Code{
			Text:   sess.Code.Text,
			Python: sess.Code.Python,
		}
	}
	if sess.Usage != nil {
		pb.Usage = &antv1.DebateV2Usage{
			PromptTokens:     sess.Usage.PromptTokens,
			CompletionTokens: sess.Usage.CompletionTokens,
			TotalTokens:      sess.Usage.TotalTokens,
		}
	}
	return pb
}
