package connect

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	aiSvc "anttrader/internal/service/systemai"
)

// AIServer implements the ant.v1.AIServiceHandler interface.
type AIServer struct {
	systemSvc *aiSvc.Service
	pg        *pgxpool.Pool
	log       *zap.Logger
}

// NewAIServer creates an AI service ConnectRPC handler.
func NewAIServer(systemSvc *aiSvc.Service, pg *pgxpool.Pool, log *zap.Logger) *AIServer {
	return &AIServer{systemSvc: systemSvc, pg: pg, log: log}
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
			now := time.Now()
			_, _ = s.pg.Exec(ctx,
				`INSERT INTO ai_messages (id, conversation_id, role, content, created_at) VALUES (gen_random_uuid(), $1, 'user', $2, $3)`,
				cid, m.Message, now,
			)
			_, _ = s.pg.Exec(ctx,
				`INSERT INTO ai_messages (id, conversation_id, role, content, created_at) VALUES (gen_random_uuid(), $1, 'assistant', $2, $3)`,
				cid, reply, now,
			)
			_, _ = s.pg.Exec(ctx, `UPDATE ai_conversations SET updated_at=$1 WHERE id=$2 AND user_id=$3`, now, cid, uid)
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
	rows, err := s.pg.Query(ctx, `
		SELECT c.id, c.title, c.created_at, c.updated_at, COALESCE(m.cnt, 0) AS message_count
		FROM ai_conversations c
		LEFT JOIN (SELECT conversation_id, COUNT(*) AS cnt FROM ai_messages GROUP BY conversation_id) m
		  ON m.conversation_id = c.id
		WHERE c.user_id = $1
		ORDER BY c.updated_at DESC
	`, uid)
	if err != nil {
		s.log.Error("ListConversations", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var convs []*antv1.ConversationSummary
	for rows.Next() {
		var id, title string
		var msgCount int32
		var cat, uat time.Time
		if err := rows.Scan(&id, &title, &cat, &uat, &msgCount); err != nil {
			continue
		}
		convs = append(convs, &antv1.ConversationSummary{
			Id:           id,
			Title:        title,
			MessageCount: msgCount,
			CreatedAt:    timestamppb.New(cat),
			UpdatedAt:    timestamppb.New(uat),
		})
	}
	if err := rows.Err(); err != nil {
		s.log.Warn("ListConversations rows iteration error", zap.Error(err))
	}
	return connect.NewResponse(&antv1.ListConversationsResponse{Conversations: convs}), nil
}

func (s *AIServer) GetConversation(ctx context.Context, req *connect.Request[antv1.GetConversationRequest]) (*connect.Response[antv1.GetConversationResponse], error) {
	uid := s.userID(ctx)
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var id, title string
	var cat, uat time.Time
	err = s.pg.QueryRow(ctx,
		`SELECT id, title, created_at, updated_at FROM ai_conversations WHERE id=$1 AND user_id=$2`, cid, uid,
	).Scan(&id, &title, &cat, &uat)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation not found"))
	}

	conv := &antv1.ConversationSummary{
		Id:        id,
		Title:     title,
		CreatedAt: timestamppb.New(cat),
		UpdatedAt: timestamppb.New(uat),
	}

	msgRows, err := s.pg.Query(ctx,
		`SELECT id, role, content, created_at FROM ai_messages WHERE conversation_id=$1 ORDER BY created_at ASC`, cid,
	)
	if err != nil {
		s.log.Error("GetConversation messages", zap.Error(err))
		return connect.NewResponse(&antv1.GetConversationResponse{Conversation: conv}), nil
	}
	defer msgRows.Close()

	var messages []*antv1.ConversationMessage
	for msgRows.Next() {
		var mid, role, content string
		var mcat time.Time
		if err := msgRows.Scan(&mid, &role, &content, &mcat); err != nil {
			continue
		}
		messages = append(messages, &antv1.ConversationMessage{
			Id:        mid,
			Role:      role,
			Content:   content,
			CreatedAt: timestamppb.New(mcat),
		})
	}
	return connect.NewResponse(&antv1.GetConversationResponse{
		Conversation: conv,
		Messages:     messages,
	}), nil
}

func (s *AIServer) CreateConversation(ctx context.Context, req *connect.Request[antv1.CreateConversationRequest]) (*connect.Response[antv1.CreateConversationResponse], error) {
	uid := s.userID(ctx)
	title := req.Msg.Title
	if title == "" {
		title = "新对话"
	}
	id := uuid.New()
	now := time.Now()
	_, err := s.pg.Exec(ctx,
		`INSERT INTO ai_conversations (id, user_id, title, created_at, updated_at) VALUES ($1, $2, $3, $4, $4)`,
		id, uid, title, now,
	)
	if err != nil {
		s.log.Error("CreateConversation", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CreateConversationResponse{
		Conversation: &antv1.ConversationSummary{
			Id:        id.String(),
			Title:     title,
			CreatedAt: timestamppb.New(now),
			UpdatedAt: timestamppb.New(now),
		},
	}), nil
}

func (s *AIServer) DeleteConversation(ctx context.Context, req *connect.Request[antv1.DeleteConversationRequest]) (*connect.Response[antv1.DeleteConversationResponse], error) {
	uid := s.userID(ctx)
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	_, _ = s.pg.Exec(ctx, `DELETE FROM ai_messages WHERE conversation_id=$1`, cid)
	_, err = s.pg.Exec(ctx, `DELETE FROM ai_conversations WHERE id=$1 AND user_id=$2`, cid, uid)
	if err != nil {
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
	_, err = s.pg.Exec(ctx,
		`UPDATE ai_conversations SET title=$3, updated_at=NOW() WHERE id=$1 AND user_id=$2`,
		cid, uid, req.Msg.Title,
	)
	if err != nil {
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
