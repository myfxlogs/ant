package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
)

func (s *AIService) ListConversations(ctx context.Context, req *connect.Request[v1.ListConversationsRequest]) (*connect.Response[v1.ListConversationsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.convRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("conversation repository not initialized"))
	}

	convs, err := s.convRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var out []*v1.ConversationSummary
	for _, c := range convs {
		out = append(out, &v1.ConversationSummary{
			Id:           c.ID.String(),
			Title:        c.Title,
			MessageCount: int32(c.MessageCount),
			CreatedAt:    timestamppb.New(c.CreatedAt),
			UpdatedAt:    timestamppb.New(c.UpdatedAt),
		})
	}
	return connect.NewResponse(&v1.ListConversationsResponse{Conversations: out}), nil
}

func (s *AIService) GetConversation(ctx context.Context, req *connect.Request[v1.GetConversationRequest]) (*connect.Response[v1.GetConversationResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.convRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("conversation repository not initialized"))
	}

	convID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	conv, err := s.convRepo.GetByID(ctx, convID, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	msgs, err := s.convRepo.GetMessages(ctx, convID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbMsgs []*v1.ConversationMessage
	for _, m := range msgs {
		pbMsgs = append(pbMsgs, &v1.ConversationMessage{
			Id:        m.ID.String(),
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: timestamppb.New(m.CreatedAt),
		})
	}

	return connect.NewResponse(&v1.GetConversationResponse{
		Conversation: &v1.ConversationSummary{
			Id:           conv.ID.String(),
			Title:        conv.Title,
			MessageCount: int32(len(msgs)),
			CreatedAt:    timestamppb.New(conv.CreatedAt),
			UpdatedAt:    timestamppb.New(conv.UpdatedAt),
		},
		Messages: pbMsgs,
	}), nil
}

func (s *AIService) CreateConversation(ctx context.Context, req *connect.Request[v1.CreateConversationRequest]) (*connect.Response[v1.CreateConversationResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.convRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("conversation repository not initialized"))
	}

	title := req.Msg.Title
	if title == "" {
		title = "ai.conversation.defaultTitle"
	}

	conv, err := s.convRepo.Create(ctx, userID, title)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.CreateConversationResponse{
		Conversation: &v1.ConversationSummary{
			Id:        conv.ID.String(),
			Title:     conv.Title,
			CreatedAt: timestamppb.New(conv.CreatedAt),
			UpdatedAt: timestamppb.New(conv.UpdatedAt),
		},
	}), nil
}

func (s *AIService) DeleteConversation(ctx context.Context, req *connect.Request[v1.DeleteConversationRequest]) (*connect.Response[v1.DeleteConversationResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.convRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("conversation repository not initialized"))
	}

	convID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.convRepo.Delete(ctx, convID, userID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.DeleteConversationResponse{Success: true}), nil
}

func (s *AIService) UpdateConversationTitle(ctx context.Context, req *connect.Request[v1.UpdateConversationTitleRequest]) (*connect.Response[v1.UpdateConversationTitleResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.convRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("conversation repository not initialized"))
	}

	convID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.convRepo.UpdateTitle(ctx, convID, userID, req.Msg.Title); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.UpdateConversationTitleResponse{Success: true}), nil
}
