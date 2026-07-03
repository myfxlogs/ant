package agent

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
)

// authenticatedUser extracts and validates the user ID from the ConnectRPC context.
func (s *GatewayServer) authenticatedUser(ctx context.Context) (uuid.UUID, error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, errors.New("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid user ID"))
	}
	return uid, nil
}

// SearchExperience searches the knowledge base for similar experiences (ADR-0025 §4).
func (s *GatewayServer) SearchExperience(
	ctx context.Context,
	req *connect.Request[antv1.SearchExperienceRequest],
) (*connect.Response[antv1.SearchExperienceResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.SearchExperienceResponse{}), nil
	}

	entries, err := s.memory.SearchExperiences(ctx, uid, req.Msg.Query, req.Msg.Category, int(req.Msg.Limit))
	if err != nil {
		s.log.Warn("SearchExperience failed", zap.Error(err))
		return connect.NewResponse(&antv1.SearchExperienceResponse{}), nil
	}
	return connect.NewResponse(&antv1.SearchExperienceResponse{Entries: entries}), nil
}

// StoreExperience stores an experience to the knowledge base (ADR-0025 §6.2).
func (s *GatewayServer) StoreExperience(
	ctx context.Context,
	req *connect.Request[antv1.StoreExperienceRequest],
) (*connect.Response[antv1.StoreExperienceResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.StoreExperienceResponse{
			Id:      uuid.New().String(),
			Success: false,
		}), nil
	}

	id, err := s.memory.StoreExperience(ctx, uid, req.Msg.Category, req.Msg.Content, req.Msg.Fingerprint, nil, "")
	if err != nil {
		s.log.Warn("StoreExperience failed", zap.Error(err))
		return connect.NewResponse(&antv1.StoreExperienceResponse{
			Success: false,
		}), nil
	}
	return connect.NewResponse(&antv1.StoreExperienceResponse{
		Id:      id,
		Success: true,
	}), nil
}

// ListMemory returns user templates and agent experiences (ADR-0025 §4.4).
func (s *GatewayServer) ListMemory(
	ctx context.Context,
	req *connect.Request[antv1.ListMemoryRequest],
) (*connect.Response[antv1.ListMemoryResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.ListMemoryResponse{}), nil
	}

	templates, err := s.memory.ListUserTemplates(ctx, uid)
	if err != nil {
		s.log.Warn("ListMemory: templates failed", zap.Error(err))
	}
	experiences, err := s.memory.ListAgentExperiences(ctx, uid)
	if err != nil {
		s.log.Warn("ListMemory: experiences failed", zap.Error(err))
	}
	return connect.NewResponse(&antv1.ListMemoryResponse{
		Templates:   templates,
		Experiences: experiences,
	}), nil
}

// SaveUserTemplate saves a user strategy template (ADR-0025 §4.4).
func (s *GatewayServer) SaveUserTemplate(
	ctx context.Context,
	req *connect.Request[antv1.SaveUserTemplateRequest],
) (*connect.Response[antv1.SaveUserTemplateResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.SaveUserTemplateResponse{Success: false}), nil
	}

	err = s.memory.SaveUserTemplate(ctx, uid, req.Msg.Name, req.Msg.Content, req.Msg.ScopeJson)
	if err != nil {
		s.log.Warn("SaveUserTemplate failed", zap.Error(err))
		return connect.NewResponse(&antv1.SaveUserTemplateResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.SaveUserTemplateResponse{Success: true}), nil
}

// DeleteUserTemplate deletes a user strategy template (ADR-0025 §4.4).
func (s *GatewayServer) DeleteUserTemplate(
	ctx context.Context,
	req *connect.Request[antv1.DeleteUserTemplateRequest],
) (*connect.Response[antv1.DeleteUserTemplateResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.DeleteUserTemplateResponse{Success: false}), nil
	}

	err = s.memory.DeleteUserTemplate(ctx, uid, req.Msg.Id)
	if err != nil {
		s.log.Warn("DeleteUserTemplate failed", zap.Error(err))
		return connect.NewResponse(&antv1.DeleteUserTemplateResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.DeleteUserTemplateResponse{Success: true}), nil
}

// DeleteAgentExperience deletes an agent experience (ADR-0025 §4.4).
func (s *GatewayServer) DeleteAgentExperience(
	ctx context.Context,
	req *connect.Request[antv1.DeleteAgentExperienceRequest],
) (*connect.Response[antv1.DeleteAgentExperienceResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.DeleteAgentExperienceResponse{Success: false}), nil
	}

	err = s.memory.DeleteAgentExperience(ctx, uid, req.Msg.Id)
	if err != nil {
		s.log.Warn("DeleteAgentExperience failed", zap.Error(err))
		return connect.NewResponse(&antv1.DeleteAgentExperienceResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.DeleteAgentExperienceResponse{Success: true}), nil
}

// GetAgentSettings returns resolved settings for the authenticated user (ADR-0025 §5).
func (s *GatewayServer) GetAgentSettings(
	ctx context.Context,
	req *connect.Request[antv1.GetAgentSettingsRequest],
) (*connect.Response[antv1.GetAgentSettingsResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.settings == nil {
		return connect.NewResponse(&antv1.GetAgentSettingsResponse{}), nil
	}

	rs, err := s.settings.ResolveSettings(ctx, uid)
	if err != nil {
		s.log.Warn("GetAgentSettings failed", zap.Error(err))
		return connect.NewResponse(&antv1.GetAgentSettingsResponse{}), nil
	}

	var entries []*antv1.AgentSettingEntry
	for k, v := range rs.Flat {
		tier := rs.Tiers[k]
		if tier == "" {
			tier = "default"
		}
		entries = append(entries, &antv1.AgentSettingEntry{Key: k, Value: v, Tier: tier})
	}
	return connect.NewResponse(&antv1.GetAgentSettingsResponse{Settings: entries}), nil
}

// SetUserSetting sets a user-level setting override (ADR-0025 §5).
func (s *GatewayServer) SetUserSetting(
	ctx context.Context,
	req *connect.Request[antv1.SetUserSettingRequest],
) (*connect.Response[antv1.SetUserSettingResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.settings == nil {
		return connect.NewResponse(&antv1.SetUserSettingResponse{Success: false}), nil
	}

	err = s.settings.SetUserSetting(ctx, uid, req.Msg.Key, req.Msg.Value)
	if err != nil {
		s.log.Warn("SetUserSetting failed", zap.Error(err))
		return connect.NewResponse(&antv1.SetUserSettingResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.SetUserSettingResponse{Success: true}), nil
}

// DeleteUserSetting removes a user-level setting override (ADR-0025 §5).
func (s *GatewayServer) DeleteUserSetting(
	ctx context.Context,
	req *connect.Request[antv1.DeleteUserSettingRequest],
) (*connect.Response[antv1.DeleteUserSettingResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.settings == nil {
		return connect.NewResponse(&antv1.DeleteUserSettingResponse{Success: false}), nil
	}

	err = s.settings.DeleteUserSetting(ctx, uid, req.Msg.Key)
	if err != nil {
		s.log.Warn("DeleteUserSetting failed", zap.Error(err))
		return connect.NewResponse(&antv1.DeleteUserSettingResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.DeleteUserSettingResponse{Success: true}), nil
}

// GetCapabilities returns the capability permissions for the authenticated user (ADR-0025 §9).
func (s *GatewayServer) GetCapabilities(
	ctx context.Context,
	req *connect.Request[antv1.GetCapabilitiesRequest],
) (*connect.Response[antv1.GetCapabilitiesResponse], error) {
	uid, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	if s.permissions == nil {
		return connect.NewResponse(&antv1.GetCapabilitiesResponse{}), nil
	}

	caps := s.permissions.CapabilitiesForUser(ctx, uid)
	var entries []*antv1.CapabilityEntry
	for cap, allowed := range caps {
		entries = append(entries, &antv1.CapabilityEntry{
			Capability: string(cap),
			Allowed:    allowed,
		})
	}
	return connect.NewResponse(&antv1.GetCapabilitiesResponse{Capabilities: entries}), nil
}
