package connect

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/service"
)

type DebateV2Service struct {
	svc *service.DebateV2Service
}

func NewDebateV2Service(svc *service.DebateV2Service) *DebateV2Service {
	return &DebateV2Service{svc: svc}
}

func (s *DebateV2Service) StartDebateV2(ctx context.Context, req *connect.Request[v1.StartDebateV2Request]) (*connect.Response[v1.DebateV2Session], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	dto, err := s.svc.Start(ctx, userID, req.Msg.Agents, req.Msg.Title, req.Msg.Locale)
	return debateV2SessionResponse(dto, err)
}

func (s *DebateV2Service) ChatDebateV2(ctx context.Context, req *connect.Request[v1.ChatDebateV2Request]) (*connect.Response[v1.DebateV2Session], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	dto, err := s.svc.Chat(ctx, userID, sid, req.Msg.Message, req.Msg.Locale)
	return debateV2SessionResponse(dto, err)
}

func (s *DebateV2Service) AdvanceDebateV2(ctx context.Context, req *connect.Request[v1.DebateV2SessionRequest]) (*connect.Response[v1.DebateV2Session], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	dto, err := s.svc.Advance(ctx, userID, sid, req.Msg.Locale)
	return debateV2SessionResponse(dto, err)
}

func (s *DebateV2Service) StartDebateV2AdvanceJob(ctx context.Context, req *connect.Request[v1.DebateV2SessionRequest]) (*connect.Response[v1.StartDebateV2AdvanceJobResponse], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	jobID, err := s.svc.StartAdvanceJob(ctx, userID, sid, req.Msg.Locale)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.StartDebateV2AdvanceJobResponse{JobId: jobID, SessionId: sid.String()}), nil
}

func (s *DebateV2Service) GetDebateV2AdvanceJob(ctx context.Context, req *connect.Request[v1.GetDebateV2AdvanceJobRequest]) (*connect.Response[v1.GetDebateV2AdvanceJobResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	jid := strings.TrimSpace(req.Msg.JobId)
	if jid == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_id is required"))
	}
	phase, msg, sessionID, err := s.svc.GetAdvanceJobStatus(jid, userID)
	if err != nil {
		if err.Error() == "job not found" {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if err.Error() == "forbidden" {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetDebateV2AdvanceJobResponse{Phase: phase, Message: msg, SessionId: sessionID}), nil
}

func (s *DebateV2Service) PrepareDebateV2ChatJob(ctx context.Context, req *connect.Request[v1.ChatDebateV2Request]) (*connect.Response[v1.StartDebateV2ChatJobResponse], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	jobID, err := s.svc.PrepareChatJob(ctx, userID, sid, req.Msg.Message, req.Msg.Locale)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.StartDebateV2ChatJobResponse{JobId: jobID, SessionId: sid.String()}), nil
}

func (s *DebateV2Service) RunDebateV2ChatJob(ctx context.Context, req *connect.Request[v1.RunDebateV2ChatJobRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	jid := strings.TrimSpace(req.Msg.JobId)
	if jid == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_id is required"))
	}
	err = s.svc.RunChatJob(ctx, userID, jid)
	if err != nil {
		if errors.Is(err, service.ErrChatJobNotRunnable) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *DebateV2Service) GetDebateV2ChatJob(ctx context.Context, req *connect.Request[v1.GetDebateV2ChatJobRequest]) (*connect.Response[v1.GetDebateV2ChatJobResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	jid := strings.TrimSpace(req.Msg.JobId)
	if jid == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_id is required"))
	}
	phase, msg, sessionID, err := s.svc.GetChatJobStatus(jid, userID)
	if err != nil {
		if err.Error() == "job not found" {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if err.Error() == "forbidden" {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetDebateV2ChatJobResponse{Phase: phase, Message: msg, SessionId: sessionID}), nil
}

func (s *DebateV2Service) BackDebateV2(ctx context.Context, req *connect.Request[v1.DebateV2SessionRequest]) (*connect.Response[v1.DebateV2Session], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	dto, err := s.svc.Back(ctx, userID, sid, req.Msg.Locale)
	return debateV2SessionResponse(dto, err)
}

func (s *DebateV2Service) SetDebateV2Params(ctx context.Context, req *connect.Request[v1.SetDebateV2ParamsRequest]) (*connect.Response[v1.DebateV2Session], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	dto, err := s.svc.SetParamSchema(ctx, userID, sid, debateV2ModelParams(req.Msg.Params), req.Msg.Locale)
	return debateV2SessionResponse(dto, err)
}

func (s *DebateV2Service) RejectDebateV2Code(ctx context.Context, req *connect.Request[v1.RejectDebateV2CodeRequest]) (*connect.Response[v1.DebateV2Session], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	dto, err := s.svc.RejectCode(ctx, userID, sid, req.Msg.Feedback, req.Msg.Locale)
	return debateV2SessionResponse(dto, err)
}

func (s *DebateV2Service) StartDebateV2RejectCodeJob(ctx context.Context, req *connect.Request[v1.RejectDebateV2CodeRequest]) (*connect.Response[v1.StartDebateV2AdvanceJobResponse], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	jobID, err := s.svc.StartRejectCodeJob(ctx, userID, sid, req.Msg.Feedback, req.Msg.Locale)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.StartDebateV2AdvanceJobResponse{JobId: jobID, SessionId: sid.String()}), nil
}

func (s *DebateV2Service) ListDebateV2Sessions(ctx context.Context, req *connect.Request[v1.ListDebateV2SessionsRequest]) (*connect.Response[v1.ListDebateV2SessionsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 50
	}
	dtos, err := s.svc.List(ctx, userID, limit, req.Msg.Locale)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.DebateV2Session, 0, len(dtos))
	for i := range dtos {
		out = append(out, debateV2Session(&dtos[i]))
	}
	return connect.NewResponse(&v1.ListDebateV2SessionsResponse{Sessions: out}), nil
}

func (s *DebateV2Service) GetDebateV2Session(ctx context.Context, req *connect.Request[v1.GetDebateV2SessionRequest]) (*connect.Response[v1.DebateV2Session], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	dto, err := s.svc.Get(ctx, userID, sid, req.Msg.Locale)
	return debateV2SessionResponse(dto, err)
}

func (s *DebateV2Service) DeleteDebateV2Session(ctx context.Context, req *connect.Request[v1.DeleteDebateV2SessionRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Delete(ctx, userID, sid); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func debateV2UserAndSession(ctx context.Context, raw string) (uuid.UUID, uuid.UUID, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	sid, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, uuid.Nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return userID, sid, nil
}

func debateV2SessionResponse(dto *service.V2SessionDTO, err error) (*connect.Response[v1.DebateV2Session], error) {
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(debateV2Session(dto)), nil
}

func debateV2Session(dto *service.V2SessionDTO) *v1.DebateV2Session {
	out := &v1.DebateV2Session{Id: dto.ID.String(), Title: dto.Title, Status: dto.Status, CurrentStep: dto.CurrentStep, Agents: dto.Agents, Provider: dto.Provider, Model: dto.Model, CreatedAt: dto.CreatedAt, UpdatedAt: dto.UpdatedAt}
	out.Usage = &v1.DebateV2Usage{PromptTokens: int32(dto.Usage.PromptTokens), CompletionTokens: int32(dto.Usage.CompletionTokens), TotalTokens: int32(dto.Usage.TotalTokens)}
	if dto.Code != nil {
		out.Code = &v1.DebateV2Code{Text: dto.Code.Text, Python: dto.Code.Python}
	}
	out.ParamSchema = debateV2ProtoParams(dto.ParamSchema)
	for _, st := range dto.Steps {
		step := &v1.DebateV2Step{StepKey: st.StepKey, AgentKey: st.AgentKey, AgentName: st.AgentName}
		for _, m := range st.Messages {
			step.Messages = append(step.Messages, &v1.DebateV2Message{Id: m.ID.String(), Role: m.Role, Content: m.Content, Kind: m.Kind})
		}
		out.Steps = append(out.Steps, step)
	}
	return out
}

func debateV2ProtoParams(params []model.TemplateParameter) []*v1.TemplateParameter {
	out := make([]*v1.TemplateParameter, 0, len(params))
	for _, p := range params {
		out = append(out, &v1.TemplateParameter{Name: p.Name, Type: p.Type, Default: p.Default, Min: p.Min, Max: p.Max, Step: p.Step, Label: p.Label, Description: p.Description, Options: p.Options})
	}
	return out
}

func debateV2ModelParams(params []*v1.TemplateParameter) []model.TemplateParameter {
	out := make([]model.TemplateParameter, 0, len(params))
	for _, p := range params {
		if p != nil {
			out = append(out, model.TemplateParameter{Name: p.Name, Type: p.Type, Default: p.Default, Min: p.Min, Max: p.Max, Step: p.Step, Label: p.Label, Description: p.Description, Options: p.Options})
		}
	}
	return out
}
