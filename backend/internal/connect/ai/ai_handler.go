package ai

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	aiSvc "anttrader/internal/service/systemai"
)

// AIServer implements the ant.v1.AIServiceHandler interface.
type AIServer struct {
	systemSvc    *aiSvc.Service
	conversations *repository.AIConversationRepository
	log          *zap.Logger
}

// NewAIServer creates an AI service ConnectRPC handler.
func NewAIServer(systemSvc *aiSvc.Service, conversations *repository.AIConversationRepository, log *zap.Logger) *AIServer {
	return &AIServer{systemSvc: systemSvc, conversations: conversations, log: log}
}

var _ antv1c.AIServiceHandler = (*AIServer)(nil)

func (s *AIServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

// ── Reports ──

func (s *AIServer) GetAIReports(ctx context.Context, req *connect.Request[antv1.GetAIReportsRequest]) (*connect.Response[antv1.GetAIReportsResponse], error) {
	return connect.NewResponse(&antv1.GetAIReportsResponse{
		Reports: []*antv1.AIReport{},
	}), nil
}

func (s *AIServer) GenerateReport(ctx context.Context, req *connect.Request[antv1.GenerateReportRequest]) (*connect.Response[antv1.GenerateReportResponse], error) {
	return connect.NewResponse(&antv1.GenerateReportResponse{
		Report: &antv1.AIReport{
			Id:         uuid.New().String(),
			Title:      "Mock Report",
			ReportType: "market_analysis",
		},
	}), nil
}

// ── Chat ──

func (s *AIServer) Chat(ctx context.Context, req *connect.Request[antv1.ChatRequest]) (*connect.Response[antv1.ChatResponse], error) {
	m := req.Msg
	reply := fmt.Sprintf("收到消息：「%s」。AI 助手正在开发中，这是模拟回复。", m.Message)
	if m.ConversationId != "" {
		uid := s.userID(ctx)
		cid, err := uuid.Parse(m.ConversationId)
		if err == nil {
			// Verify conversation belongs to the requesting user before adding messages.
			if _, err := s.conversations.GetByID(ctx, cid, uid); err == nil {
				now := time.Now()
				_, _ = s.conversations.AddMessage(ctx, cid, "user", m.Message)
				_, _ = s.conversations.AddMessage(ctx, cid, "assistant", reply)
				_ = s.conversations.Touch(ctx, cid)
				_ = now
			}
		}
	}
	return connect.NewResponse(&antv1.ChatResponse{
		Message:     reply,
		Suggestions: []string{"分析当前持仓风险", "推荐交易策略", "解释技术指标"},
	}), nil
}

func (s *AIServer) ChatStream(ctx context.Context, req *connect.Request[antv1.ChatRequest], stream *connect.ServerStream[antv1.ChatStreamChunk]) error {
	text := fmt.Sprintf("收到消息：「%s」。AI 助手正在开发中。", req.Msg.Message)
	runes := []rune(text)
	for i, c := range runes {
		chunk := &antv1.ChatStreamChunk{Delta: string(c), Done: false}
		if i == len(runes)-1 {
			chunk.Done = true
			chunk.PromptTokens = 10
			chunk.CompletionTokens = int32(len(runes) / 4)
		}
		if err := stream.Send(chunk); err != nil {
			return fmt.Errorf("send chat stream chunk: %w", err)
		}
	}
	return nil
}

// ── Conversations ──

func (s *AIServer) ListConversations(ctx context.Context, req *connect.Request[antv1.ListConversationsRequest]) (*connect.Response[antv1.ListConversationsResponse], error) {
	uid := s.userID(ctx)
	convs, err := s.conversations.ListByUser(ctx, uid)
	if err != nil {
		s.log.Error("ListConversations", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var out []*antv1.ConversationSummary
	for _, c := range convs {
		out = append(out, &antv1.ConversationSummary{
			Id:           c.ID.String(),
			Title:        c.Title,
			MessageCount: int32(c.MessageCount),
			CreatedAt:    timestamppb.New(c.CreatedAt),
			UpdatedAt:    timestamppb.New(c.UpdatedAt),
		})
	}
	return connect.NewResponse(&antv1.ListConversationsResponse{Conversations: out}), nil
}

func (s *AIServer) GetConversation(ctx context.Context, req *connect.Request[antv1.GetConversationRequest]) (*connect.Response[antv1.GetConversationResponse], error) {
	uid := s.userID(ctx)
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	conv, err := s.conversations.GetByID(ctx, cid, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation not found"))
	}
	summary := &antv1.ConversationSummary{
		Id:        conv.ID.String(),
		Title:     conv.Title,
		CreatedAt: timestamppb.New(conv.CreatedAt),
		UpdatedAt: timestamppb.New(conv.UpdatedAt),
	}
	msgs, err := s.conversations.GetMessages(ctx, cid)
	if err != nil {
		s.log.Error("GetConversation messages", zap.Error(err))
		return connect.NewResponse(&antv1.GetConversationResponse{Conversation: summary}), nil
	}
	var messages []*antv1.ConversationMessage
	for _, m := range msgs {
		messages = append(messages, &antv1.ConversationMessage{
			Id:        m.ID.String(),
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: timestamppb.New(m.CreatedAt),
		})
	}
	return connect.NewResponse(&antv1.GetConversationResponse{
		Conversation: summary,
		Messages:     messages,
	}), nil
}

func (s *AIServer) CreateConversation(ctx context.Context, req *connect.Request[antv1.CreateConversationRequest]) (*connect.Response[antv1.CreateConversationResponse], error) {
	uid := s.userID(ctx)
	title := req.Msg.Title
	if title == "" {
		title = "新对话"
	}
	conv, err := s.conversations.Create(ctx, uid, title)
	if err != nil {
		s.log.Error("CreateConversation", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CreateConversationResponse{
		Conversation: &antv1.ConversationSummary{
			Id:        conv.ID.String(),
			Title:     conv.Title,
			CreatedAt: timestamppb.New(conv.CreatedAt),
			UpdatedAt: timestamppb.New(conv.UpdatedAt),
		},
	}), nil
}

func (s *AIServer) DeleteConversation(ctx context.Context, req *connect.Request[antv1.DeleteConversationRequest]) (*connect.Response[antv1.DeleteConversationResponse], error) {
	uid := s.userID(ctx)
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Verify ownership before deleting messages or the conversation itself.
	if _, err := s.conversations.GetByID(ctx, cid, uid); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation not found"))
	}
	if err := s.conversations.DeleteMessagesByConversation(ctx, cid); err != nil {
		s.log.Warn("DeleteConversation: failed to delete messages", zap.String("cid", cid.String()), zap.Error(err))
	}
	if err := s.conversations.Delete(ctx, cid, uid); err != nil {
		s.log.Error("DeleteConversation", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteConversationResponse{}), nil
}

func (s *AIServer) UpdateConversationTitle(ctx context.Context, req *connect.Request[antv1.UpdateConversationTitleRequest]) (*connect.Response[antv1.UpdateConversationTitleResponse], error) {
	uid := s.userID(ctx)
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.conversations.UpdateTitle(ctx, cid, uid, req.Msg.Title); err != nil {
		s.log.Error("UpdateConversationTitle", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UpdateConversationTitleResponse{}), nil
}

// ── Workflow ──

func (s *AIServer) CreateWorkflowRun(ctx context.Context, req *connect.Request[antv1.CreateWorkflowRunRequest]) (*connect.Response[antv1.CreateWorkflowRunResponse], error) {
	return connect.NewResponse(&antv1.CreateWorkflowRunResponse{
		Run: &antv1.WorkflowRunSummary{
			Id:     uuid.New().String(),
			Status: "running",
		},
	}), nil
}

func (s *AIServer) AppendWorkflowStep(ctx context.Context, req *connect.Request[antv1.AppendWorkflowStepRequest]) (*connect.Response[antv1.AppendWorkflowStepResponse], error) {
	return connect.NewResponse(&antv1.AppendWorkflowStepResponse{
		Step: &antv1.WorkflowStep{
			Id:     uuid.New().String(),
			Status: "completed",
		},
	}), nil
}

func (s *AIServer) ListWorkflowRuns(ctx context.Context, req *connect.Request[antv1.ListWorkflowRunsRequest]) (*connect.Response[antv1.ListWorkflowRunsResponse], error) {
	return connect.NewResponse(&antv1.ListWorkflowRunsResponse{
		Runs: []*antv1.WorkflowRunSummary{},
	}), nil
}

func (s *AIServer) GetWorkflowRun(ctx context.Context, req *connect.Request[antv1.GetWorkflowRunRequest]) (*connect.Response[antv1.GetWorkflowRunResponse], error) {
	return connect.NewResponse(&antv1.GetWorkflowRunResponse{
		Run: &antv1.WorkflowRunSummary{
			Id:     req.Msg.Id,
			Title:  "Mock Workflow",
			Status: "completed",
		},
	}), nil
}

// ── Agents ──

var defaultAgents = []*antv1.AIAgentDefinition{
	{AgentKey: "strategist", Type: "primary", Name: "策略分析师", Identity: "资深量化交易策略师", Enabled: true, Position: 1},
	{AgentKey: "risk_manager", Type: "secondary", Name: "风控经理", Identity: "严格的风险控制专家", Enabled: true, Position: 2},
	{AgentKey: "executor", Type: "secondary", Name: "执行顾问", Identity: "交易执行优化专家", Enabled: false, Position: 3},
	{AgentKey: "researcher", Type: "secondary", Name: "市场研究员", Identity: "宏观经济和行业研究员", Enabled: false, Position: 4},
}

func (s *AIServer) ListAgents(ctx context.Context, req *connect.Request[antv1.ListAgentsRequest]) (*connect.Response[antv1.ListAgentsResponse], error) {
	return connect.NewResponse(&antv1.ListAgentsResponse{Agents: defaultAgents}), nil
}

func (s *AIServer) SetAgents(ctx context.Context, req *connect.Request[antv1.SetAgentsRequest]) (*connect.Response[antv1.SetAgentsResponse], error) {
	return connect.NewResponse(&antv1.SetAgentsResponse{}), nil
}
