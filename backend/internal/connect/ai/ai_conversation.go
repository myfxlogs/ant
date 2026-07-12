package ai

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func (s *AIServer) ListConversations(ctx context.Context, req *connect.Request[antv1.ListConversationsRequest]) (*connect.Response[antv1.ListConversationsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	convs, err := s.conversations.ListByUser(ctx, uid)
	if err != nil {
		s.log.Error("ListConversations", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
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
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
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
	msgs, err := s.conversations.GetMessages(ctx, uid, cid)
	if err != nil {
		s.log.Error("GetConversation messages", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	var messages []*antv1.ConversationMessage
	for _, m := range msgs {
		messages = append(messages, &antv1.ConversationMessage{
			Id:        m.ID.String(),
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: timestamppb.New(m.CreatedAt),
				TurnData:  m.TurnData,
			})
		}
		return connect.NewResponse(&antv1.GetConversationResponse{
		Conversation: summary,
		Messages:     messages,
	}), nil
}

func (s *AIServer) CreateConversation(ctx context.Context, req *connect.Request[antv1.CreateConversationRequest]) (*connect.Response[antv1.CreateConversationResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	title := req.Msg.Title
	if title == "" {
		title = "新对话"
	}
	if len(title) > 200 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("title too long: %d characters", len(title)))
	}
	conv, err := s.conversations.Create(ctx, uid, title)
	if err != nil {
		s.log.Error("CreateConversation", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
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
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := s.conversations.GetByID(ctx, cid, uid); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation not found"))
	}
	if err := s.conversations.DeleteMessagesByConversation(ctx, uid, cid); err != nil {
		s.log.Error("DeleteConversation: failed to delete messages", zap.String("cid", cid.String()), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	if err := s.conversations.Delete(ctx, cid, uid); err != nil {
		s.log.Error("DeleteConversation", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.DeleteConversationResponse{}), nil
}

func (s *AIServer) UpdateConversationTitle(ctx context.Context, req *connect.Request[antv1.UpdateConversationTitleRequest]) (*connect.Response[antv1.UpdateConversationTitleResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.Title) > 200 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("title too long: %d characters", len(req.Msg.Title)))
	}
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.conversations.UpdateTitle(ctx, cid, uid, req.Msg.Title); err != nil {
		s.log.Error("UpdateConversationTitle", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.UpdateConversationTitleResponse{}), nil
}
