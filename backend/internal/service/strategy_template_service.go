package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

var (
	ErrTemplateNameEmpty    = errors.New("template name cannot be empty")
	ErrTemplateCodeEmpty    = errors.New("template code cannot be empty")
	ErrTemplateNotDraft     = errors.New("template is not a draft")
	ErrTemplateUnauthorized = errors.New("unauthorized access to template")
)

type StrategyTemplateService struct {
	templateRepo *repository.StrategyTemplateRepository
}

func NewStrategyTemplateService(templateRepo *repository.StrategyTemplateRepository) *StrategyTemplateService {
	return &StrategyTemplateService{
		templateRepo: templateRepo,
	}
}

func (s *StrategyTemplateService) CreateTemplate(ctx context.Context, userID uuid.UUID, name, description, code string, parameters []model.TemplateParameter, isPublic bool, tags []string) (*model.StrategyTemplate, error) {
	if name == "" {
		return nil, ErrTemplateNameEmpty
	}
	if code == "" {
		return nil, ErrTemplateCodeEmpty
	}

	template := model.NewStrategyTemplate(userID, name, code)
	template.Status = model.StrategyTemplateStatusPublished
	template.Description = description
	template.IsPublic = isPublic
	template.Tags = tags

	if len(parameters) > 0 {
		if err := template.SetParameters(parameters); err != nil {
			return nil, err
		}
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		logger.Error("Failed to create template", zap.Error(err))
		return nil, err
	}

	return template, nil
}

func (s *StrategyTemplateService) CreateDraft(ctx context.Context, userID uuid.UUID, name string) (*model.StrategyTemplate, error) {
	if name == "" {
		return nil, ErrTemplateNameEmpty
	}
	tpl := model.NewStrategyTemplate(userID, name, "")
	tpl.Status = model.StrategyTemplateStatusDraft
	tpl.Description = ""
	tpl.IsPublic = false
	tpl.Tags = []string{}
	if err := s.templateRepo.Create(ctx, tpl); err != nil {
		logger.Error("Failed to create draft template", zap.Error(err))
		return nil, err
	}
	return tpl, nil
}

func (s *StrategyTemplateService) UpdateDraft(ctx context.Context, userID uuid.UUID, tpl *model.StrategyTemplate) error {
	if tpl == nil {
		return repository.ErrTemplateNotFound
	}
	if tpl.UserID != userID {
		return ErrTemplateUnauthorized
	}
	if tpl.Status != model.StrategyTemplateStatusDraft {
		return ErrTemplateNotDraft
	}
	return s.templateRepo.Update(ctx, tpl)
}

func (s *StrategyTemplateService) PublishDraft(ctx context.Context, userID, tplID uuid.UUID) (*model.StrategyTemplate, error) {
	tpl, err := s.templateRepo.GetByID(ctx, tplID)
	if err != nil {
		return nil, err
	}
	if tpl.UserID != userID {
		return nil, ErrTemplateUnauthorized
	}
	if tpl.Status != model.StrategyTemplateStatusDraft {
		return nil, ErrTemplateNotDraft
	}
	if tpl.Name == "" {
		return nil, ErrTemplateNameEmpty
	}
	if tpl.Code == "" {
		return nil, ErrTemplateCodeEmpty
	}
	tpl.Status = model.StrategyTemplateStatusPublished
	if err := s.templateRepo.Update(ctx, tpl); err != nil {
		return nil, err
	}
	return tpl, nil
}

func (s *StrategyTemplateService) CancelDraft(ctx context.Context, userID, tplID uuid.UUID) error {
	tpl, err := s.templateRepo.GetByID(ctx, tplID)
	if err != nil {
		return err
	}
	if tpl.UserID != userID {
		return ErrTemplateUnauthorized
	}
	if tpl.Status != model.StrategyTemplateStatusDraft {
		return ErrTemplateNotDraft
	}
	return s.templateRepo.SetStatus(ctx, tplID, model.StrategyTemplateStatusCanceled)
}

func (s *StrategyTemplateService) GetTemplate(ctx context.Context, templateID uuid.UUID) (*model.StrategyTemplate, error) {
	return s.templateRepo.GetByID(ctx, templateID)
}

func (s *StrategyTemplateService) GetTemplatesByUser(ctx context.Context, userID uuid.UUID) ([]*model.StrategyTemplate, error) {
	return s.templateRepo.GetByUserID(ctx, userID)
}

func (s *StrategyTemplateService) GetPublicTemplates(ctx context.Context, limit, offset int) ([]*model.StrategyTemplate, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.templateRepo.GetPublicTemplates(ctx, limit, offset)
}

func (s *StrategyTemplateService) SearchTemplates(ctx context.Context, userID uuid.UUID, keyword string) ([]*model.StrategyTemplate, error) {
	return s.templateRepo.Search(ctx, userID, keyword)
}

func (s *StrategyTemplateService) UpdateTemplate(ctx context.Context, template *model.StrategyTemplate) error {
	if template == nil {
		return repository.ErrTemplateNotFound
	}
	// Guard against in-place modification of system templates. The caller has
	// already loaded the template via GetTemplate() and mutated fields; if the
	// original row is a system template, refuse. This is a belt-and-braces
	// check on top of the repository-layer guard (repo Update() intentionally
	// never writes is_system back, but users could still try to edit
	// name/code/... on a preset).
	existing, err := s.templateRepo.GetByID(ctx, template.ID)
	if err != nil {
		return err
	}
	if existing != nil && existing.IsSystem {
		return repository.ErrTemplateIsSystem
	}
	return s.templateRepo.Update(ctx, template)
}

func (s *StrategyTemplateService) DeleteTemplate(ctx context.Context, templateID uuid.UUID) error {
	return s.templateRepo.Delete(ctx, templateID)
}

func (s *StrategyTemplateService) IncrementUseCount(ctx context.Context, templateID uuid.UUID) error {
	return s.templateRepo.IncrementUseCount(ctx, templateID)
}
