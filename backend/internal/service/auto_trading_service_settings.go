package service

import (
	"context"
	"errors"

	"anttrader/internal/model"
	"anttrader/internal/repository"

	"github.com/google/uuid"
)

func (s *AutoTradingService) GetGlobalSettings(ctx context.Context, userID uuid.UUID) (*model.GlobalSettings, error) {
	settings, err := s.autoTradingRepo.GetGlobalSettingsByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGlobalSettingsNotFound) {
			settings = model.NewGlobalSettings(userID)
			if err := s.autoTradingRepo.CreateGlobalSettings(ctx, settings); err != nil {
				return nil, err
			}
			return settings, nil
		}
		return nil, err
	}
	return settings, nil
}

func (s *AutoTradingService) UpdateGlobalSettings(ctx context.Context, userID uuid.UUID, settings *model.GlobalSettings) error {
	existing, err := s.autoTradingRepo.GetGlobalSettingsByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGlobalSettingsNotFound) {
			settings.UserID = userID
			return s.autoTradingRepo.CreateGlobalSettings(ctx, settings)
		}
		return err
	}
	settings.ID = existing.ID
	settings.UserID = userID
	return s.autoTradingRepo.UpdateGlobalSettings(ctx, settings)
}

func (s *AutoTradingService) ToggleAutoTrade(ctx context.Context, userID uuid.UUID, enabled bool) error {
	_, err := s.autoTradingRepo.GetGlobalSettingsByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGlobalSettingsNotFound) {
			settings := model.NewGlobalSettings(userID)
			settings.AutoTradeEnabled = enabled
			return s.autoTradingRepo.CreateGlobalSettings(ctx, settings)
		}
		return err
	}
	return s.autoTradingRepo.UpdateAutoTradeEnabled(ctx, userID, enabled)
}
