package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/ai"
	"anttrader/internal/model"
	apperrors "anttrader/internal/pkg/errors"
	"anttrader/internal/pkg/hash"
	"anttrader/internal/repository"
)

type AdminService struct {
	adminRepo        *repository.AdminRepository
	userRepo         *repository.UserRepository
	dynamicConfigSvc *DynamicConfigService
}

const (
	allowedSystemConfigKeyMaxAccountsPerUser                  = "max_accounts_per_user"
	allowedSystemConfigKeyAIProviderCatalog                   = "ai.provider_catalog"
	allowedSystemConfigKeyStrategyScheduleHealthGradingConfig = "strategy.schedule.health_grading_config"
	// Translation-related configs for economic calendar localization.
	allowedSystemConfigKeyEconZhipuAPIKey = "econ.translation.zhipu_api_key"
	allowedSystemConfigKeyEconZhipuModel  = "econ.translation.zhipu_model"
	// Unified translation AI config (JSON: provider/api_key/model/base_url/enabled)
	allowedSystemConfigKeyEconAIConfig = "econ.translation.ai_config"
)

func isAllowedSystemConfigKey(key string) bool {
	switch key {
	case allowedSystemConfigKeyMaxAccountsPerUser,
		allowedSystemConfigKeyAIProviderCatalog,
		allowedSystemConfigKeyStrategyScheduleHealthGradingConfig,
		allowedSystemConfigKeyEconZhipuAPIKey,
		allowedSystemConfigKeyEconZhipuModel,
		allowedSystemConfigKeyEconAIConfig:
		return true
	default:
		return false
	}
}

func allowedSystemConfigKeys() []string {
	return []string{
		allowedSystemConfigKeyMaxAccountsPerUser,
		allowedSystemConfigKeyAIProviderCatalog,
		allowedSystemConfigKeyStrategyScheduleHealthGradingConfig,
		allowedSystemConfigKeyEconZhipuAPIKey,
		allowedSystemConfigKeyEconZhipuModel,
		allowedSystemConfigKeyEconAIConfig,
	}
}

func NewAdminService(adminRepo *repository.AdminRepository, userRepo *repository.UserRepository, dynamicConfigSvc *DynamicConfigService) *AdminService {
	return &AdminService{
		adminRepo:        adminRepo,
		userRepo:         userRepo,
		dynamicConfigSvc: dynamicConfigSvc,
	}
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

type UpdateUserRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

func (s *AdminService) GetDashboardStats(ctx context.Context) (*model.DashboardStats, error) {
	return s.adminRepo.GetDashboardStats(ctx)
}

func (s *AdminService) ListUsers(ctx context.Context, params *model.UserListParams) (*PaginatedResponse, error) {
	users, total, err := s.adminRepo.ListUsers(ctx, params)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Data:       users,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *AdminService) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.adminRepo.GetUserByID(ctx, id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.New(apperrors.UserNotFound, nil)
		}
		return nil, apperrors.New(apperrors.InternalError, err)
	}
	return user, nil
}

func (s *AdminService) CreateUser(ctx context.Context, req *CreateUserRequest, operatorID uuid.UUID) (*model.User, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}
	if exists {
		return nil, apperrors.New(apperrors.UserAlreadyExists, nil)
	}

	passwordHash, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	role := req.Role
	if role == "" {
		role = "user"
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Email
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		Nickname:     &nickname,
		Role:         role,
		Status:       "active",
	}

	if err := s.adminRepo.CreateUser(ctx, user); err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "user_management", "create", "user", user.ID.String(), map[string]interface{}{
		"email": user.Email,
		"role":  user.Role,
	}, true, "")

	return user, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateUserRequest, operatorID uuid.UUID) (*model.User, error) {
	user, err := s.adminRepo.GetUserByID(ctx, id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.New(apperrors.UserNotFound, nil)
		}
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	if req.Nickname != "" {
		user.Nickname = &req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = &req.Avatar
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := s.adminRepo.UpdateUser(ctx, user); err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "user_management", "update", "user", id.String(), map[string]interface{}{
		"nickname": user.Nickname,
		"role":     user.Role,
		"status":   user.Status,
	}, true, "")

	return user, nil
}

func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID, operatorID uuid.UUID) error {
	err := s.adminRepo.DeleteUser(ctx, id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.New(apperrors.UserNotFound, nil)
		}
		return apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "user_management", "delete", "user", id.String(), nil, true, "")
	return nil
}

func (s *AdminService) DisableUser(ctx context.Context, id uuid.UUID, operatorID uuid.UUID) error {
	err := s.adminRepo.SetUserStatus(ctx, id, "suspended")
	if err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.New(apperrors.UserNotFound, nil)
		}
		return apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "user_management", "disable", "user", id.String(), nil, true, "")
	return nil
}

func (s *AdminService) EnableUser(ctx context.Context, id uuid.UUID, operatorID uuid.UUID) error {
	err := s.adminRepo.SetUserStatus(ctx, id, "active")
	if err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.New(apperrors.UserNotFound, nil)
		}
		return apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "user_management", "enable", "user", id.String(), nil, true, "")
	return nil
}

func (s *AdminService) ResetUserPassword(ctx context.Context, id uuid.UUID, req *ResetPasswordRequest, operatorID uuid.UUID) error {
	passwordHash, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return apperrors.New(apperrors.InternalError, err)
	}

	err = s.adminRepo.ResetUserPassword(ctx, id, passwordHash)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.New(apperrors.UserNotFound, nil)
		}
		return apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "user_management", "reset_password", "user", id.String(), nil, true, "")
	return nil
}

func (s *AdminService) ListAccounts(ctx context.Context, params *model.AccountListParams) (*PaginatedResponse, error) {
	accounts, total, err := s.adminRepo.ListAccounts(ctx, params)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Data:       accounts,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *AdminService) GetAccount(ctx context.Context, id uuid.UUID) (*repository.AccountWithUser, error) {
	account, err := s.adminRepo.GetAccountByID(ctx, id)
	if err != nil {
		if err == repository.ErrAccountNotFound {
			return nil, apperrors.New(apperrors.AccountNotFound, nil)
		}
		return nil, apperrors.New(apperrors.InternalError, err)
	}
	return account, nil
}

func (s *AdminService) FreezeAccount(ctx context.Context, id uuid.UUID, operatorID uuid.UUID) error {
	err := s.adminRepo.SetAccountStatus(ctx, id, "frozen")
	if err != nil {
		if err == repository.ErrAccountNotFound {
			return apperrors.New(apperrors.AccountNotFound, nil)
		}
		return apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "account_management", "freeze", "account", id.String(), nil, true, "")
	return nil
}

func (s *AdminService) UnfreezeAccount(ctx context.Context, id uuid.UUID, operatorID uuid.UUID) error {
	err := s.adminRepo.SetAccountStatus(ctx, id, "disconnected")
	if err != nil {
		if err == repository.ErrAccountNotFound {
			return apperrors.New(apperrors.AccountNotFound, nil)
		}
		return apperrors.New(apperrors.InternalError, err)
	}

	s.logOperation(ctx, operatorID, "account_management", "unfreeze", "account", id.String(), nil, true, "")
	return nil
}

func (s *AdminService) ListPositions(ctx context.Context, userID, accountID, symbol string, page, pageSize int) (*PaginatedResponse, error) {
	positions, total, err := s.adminRepo.ListPositions(ctx, userID, accountID, symbol, page, pageSize)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Data:       positions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *AdminService) ListOrders(ctx context.Context, userID, accountID, symbol, orderType, status string, page, pageSize int) (*PaginatedResponse, error) {
	orders, total, err := s.adminRepo.ListOrders(ctx, userID, accountID, symbol, orderType, status, page, pageSize)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Data:       orders,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *AdminService) GetTradingSummary(ctx context.Context, startDate, endDate string) (*model.TradingSummary, error) {
	if startDate == "" {
		startDate = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	return s.adminRepo.GetTradingSummary(ctx, startDate, endDate)
}

func (s *AdminService) ListLogs(ctx context.Context, params *model.LogListParams) (*PaginatedResponse, error) {
	logs, total, err := s.adminRepo.ListLogs(ctx, params)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Data:       logs,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *AdminService) GetConfig(ctx context.Context, key string) (*model.SystemConfig, error) {
	if !isAllowedSystemConfigKey(key) {
		return nil, apperrors.New(apperrors.NotFound, nil)
	}
	config, err := s.adminRepo.GetConfig(ctx, key)
	if err != nil {
		if err == repository.ErrConfigNotFound {
			return nil, apperrors.New(apperrors.NotFound, nil)
		}
		return nil, apperrors.New(apperrors.InternalError, err)
	}
	return config, nil
}

func (s *AdminService) ListConfigs(ctx context.Context) ([]*model.SystemConfig, error) {
	configs, err := s.adminRepo.ListConfigs(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.InternalError, err)
	}
	out := make([]*model.SystemConfig, 0, 2)
	seen := map[string]bool{}
	for _, c := range configs {
		if c != nil && isAllowedSystemConfigKey(c.Key) {
			out = append(out, c)
			seen[c.Key] = true
		}
	}

	// Ensure allowed keys are visible in admin UI even if not created yet.
	now := time.Now()
	enabled := true
	if !seen[allowedSystemConfigKeyMaxAccountsPerUser] {
		out = append(out, &model.SystemConfig{
			Key:         allowedSystemConfigKeyMaxAccountsPerUser,
			Value:       "",
			Description: "每用户最大账户数",
			Enabled:     &enabled,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if !seen[allowedSystemConfigKeyAIProviderCatalog] {
		out = append(out, &model.SystemConfig{
			Key:         allowedSystemConfigKeyAIProviderCatalog,
			Value:       "",
			Description: "AI provider catalog (JSON)",
			Enabled:     &enabled,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if !seen[allowedSystemConfigKeyStrategyScheduleHealthGradingConfig] {
		out = append(out, &model.SystemConfig{
			Key:         allowedSystemConfigKeyStrategyScheduleHealthGradingConfig,
			Value:       "",
			Description: "策略健康分级阈值（JSON）",
			Enabled:     &enabled,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if !seen[allowedSystemConfigKeyEconAIConfig] {
		out = append(out, &model.SystemConfig{
			Key:         allowedSystemConfigKeyEconAIConfig,
			Value:       "",
			Description: "经济日历翻译模型配置（JSON）",
			Enabled:     &enabled,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return out, nil
}

func (s *AdminService) SetConfig(ctx context.Context, key string, value, description string, operatorID uuid.UUID) error {
	if !isAllowedSystemConfigKey(key) {
		return apperrors.New(apperrors.NotFound, nil)
	}

	if key == allowedSystemConfigKeyAIProviderCatalog {
		raw := strings.TrimSpace(value)
		if raw == "" {
			return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("ai.provider_catalog cannot be empty"))
		}
		// Support both formats:
		// 1) {"providers": [{"type": "zhipu"}, ...]}
		// 2) [{"type": "zhipu"}, ...]
		var providers []struct {
			Type string `json:"type"`
		}
		var wrapped struct {
			Providers []struct {
				Type string `json:"type"`
			} `json:"providers"`
		}

		if err := json.Unmarshal([]byte(raw), &wrapped); err == nil && len(wrapped.Providers) > 0 {
			providers = wrapped.Providers
		} else {
			if err := json.Unmarshal([]byte(raw), &providers); err != nil {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("invalid ai.provider_catalog json"))
			}
		}
		if len(providers) == 0 {
			return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("ai.provider_catalog must contain at least one provider"))
		}

		for _, p := range providers {
			pt := ai.ProviderType(strings.TrimSpace(p.Type))
			if !pt.IsValid() {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("invalid provider type: %s", p.Type))
			}
		}
	}

	if key == allowedSystemConfigKeyEconAIConfig {
		raw := strings.TrimSpace(value)
		if raw != "" {
			var cfg struct {
				Provider string `json:"provider"`
				APIKey   string `json:"api_key"`
				Model    string `json:"model"`
				BaseURL  string `json:"base_url"`
				Enabled  bool   `json:"enabled"`
			}
			if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("invalid econ.translation.ai_config json"))
			}
			p := strings.ToLower(strings.TrimSpace(cfg.Provider))
			switch p {
			case "", "zhipu", "glm", "glm-4", "glm-4-flash", "deepseek", "custom", "openai":
				// allowed
			default:
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("invalid provider in econ.translation.ai_config: %s", cfg.Provider))
			}
		}
	}

	if key == allowedSystemConfigKeyStrategyScheduleHealthGradingConfig {
		raw := strings.TrimSpace(value)
		if raw != "" {
			var cfg struct {
				GreenSuccessRate  float64 `json:"green_success_rate"`
				GreenMaxFailed    int     `json:"green_max_failed_runs"`
				YellowSuccessRate float64 `json:"yellow_success_rate"`
				MinSampleSize     int     `json:"min_sample_size"`
			}
			if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("invalid strategy.schedule.health_grading_config json"))
			}
			if cfg.GreenSuccessRate < 0 || cfg.GreenSuccessRate > 100 {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("green_success_rate must be between 0 and 100"))
			}
			if cfg.YellowSuccessRate < 0 || cfg.YellowSuccessRate > 100 {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("yellow_success_rate must be between 0 and 100"))
			}
			if cfg.YellowSuccessRate > cfg.GreenSuccessRate {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("yellow_success_rate cannot be greater than green_success_rate"))
			}
			if cfg.GreenMaxFailed < 0 {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("green_max_failed_runs must be >= 0"))
			}
			if cfg.MinSampleSize < 0 {
				return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("min_sample_size must be >= 0"))
			}
		}
	}

	err := s.adminRepo.SetConfig(ctx, key, value, description)
	if err != nil {
		return apperrors.New(apperrors.InternalError, err)
	}
	if s.dynamicConfigSvc != nil {
		s.dynamicConfigSvc.InvalidateCache(key)
	}

	s.logOperation(ctx, operatorID, "system_config", "update", "config", key, map[string]interface{}{
		"value":       value,
		"description": description,
	}, true, "")
	return nil
}

func (s *AdminService) SetConfigEnabled(ctx context.Context, key string, enabled bool, operatorID uuid.UUID) error {
	if !isAllowedSystemConfigKey(key) {
		return apperrors.New(apperrors.NotFound, nil)
	}
	err := s.adminRepo.SetConfigEnabled(ctx, key, enabled)
	if err != nil {
		if err == repository.ErrConfigNotFound {
			return apperrors.New(apperrors.NotFound, nil)
		}
		return apperrors.New(apperrors.InternalError, err)
	}
	if s.dynamicConfigSvc != nil {
		s.dynamicConfigSvc.InvalidateCache(key)
	}

	s.logOperation(ctx, operatorID, "system_config", "toggle", "config", key, map[string]interface{}{
		"enabled": enabled,
	}, true, "")
	return nil
}

func (s *AdminService) HasPermission(ctx context.Context, role, permissionCode string) (bool, error) {
	return s.adminRepo.HasPermission(ctx, role, permissionCode)
}

func (s *AdminService) LogOperation(ctx context.Context, adminID *uuid.UUID, module, actionType, targetType, targetID string, details map[string]interface{}, success bool, errorMessage string) error {
	log := &model.AdminLog{
		AdminID:      adminID,
		Module:       module,
		ActionType:   actionType,
		TargetType:   targetType,
		TargetID:     targetID,
		Details:      details,
		Success:      success,
		ErrorMessage: errorMessage,
	}
	return s.adminRepo.CreateLog(ctx, log)
}

func (s *AdminService) logOperation(ctx context.Context, operatorID uuid.UUID, module, actionType, targetType, targetID string, details map[string]interface{}, success bool, errorMessage string) {
	log := &model.AdminLog{
		AdminID:      &operatorID,
		Module:       module,
		ActionType:   actionType,
		TargetType:   targetType,
		TargetID:     targetID,
		Details:      details,
		Success:      success,
		ErrorMessage: errorMessage,
	}
	s.adminRepo.CreateLog(ctx, log)
}

func (s *AdminService) ResolveAlert(ctx context.Context, alertID string, operatorID uuid.UUID) error {
	if strings.TrimSpace(alertID) == "" {
		return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("alert_id is required"))
	}
	return apperrors.New(apperrors.ServiceUnavailable, fmt.Errorf("alert resolution is not connected to the monitoring store"))
}

func (s *AdminService) ClearCache(ctx context.Context) error {
	if s.dynamicConfigSvc == nil {
		return apperrors.New(apperrors.ServiceUnavailable, fmt.Errorf("dynamic config cache is unavailable"))
	}
	for _, key := range allowedSystemConfigKeys() {
		s.dynamicConfigSvc.InvalidateCache(key)
	}
	return nil
}

func (s *AdminService) InvalidateCacheByTag(ctx context.Context, tag string) error {
	key := strings.TrimSpace(tag)
	if key == "" {
		return apperrors.New(apperrors.InvalidParameter, fmt.Errorf("cache tag is required"))
	}
	if !isAllowedSystemConfigKey(key) {
		return apperrors.New(apperrors.NotFound, nil)
	}
	if s.dynamicConfigSvc == nil {
		return apperrors.New(apperrors.ServiceUnavailable, fmt.Errorf("dynamic config cache is unavailable"))
	}
	s.dynamicConfigSvc.InvalidateCache(key)
	return nil
}
