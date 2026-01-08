package llms

import (
	"context"
	"model"
	"time"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/errs"
	"github.com/mszlu521/thunder/logs"
)

type service struct {
	repo repository
}

func (s *service) createProviderConfig(ctx context.Context, userID uuid.UUID, req CreateProviderConfigReq) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	config := model.ProviderConfig{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Provider:    req.Provider,
		Status:      req.Status,
		APIKey:      req.APIKey,
		APIBase:     req.APIBase,
	}
	err := s.repo.createProviderConfig(ctx, &config)
	if err != nil {
		logs.Errorf("create provider config error: %v", err)
		return nil, errs.DBError
	}
	return config, nil
}

func (s *service) listProviderConfigs(ctx context.Context, userId uuid.UUID) (*ListProviderConfigsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	list, total, err := s.repo.listProviderConfigs(ctx, userId)
	if err != nil {
		logs.Errorf("list provider configs error: %v", err)
		return nil, errs.DBError
	}
	return &ListProviderConfigsResponse{
		ProviderConfigs: list,
		Total:           total,
	}, nil
}

func (s *service) createLLM(ctx context.Context, userID uuid.UUID, req CreateLLMReq) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	llm := model.LLM{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		UserID:           userID,
		Name:             req.Name,
		Description:      req.Description,
		ProviderConfigID: req.ProviderConfigID,
		ModelName:        req.ModelName,
		ModelType:        req.ModelType,
		Config:           req.Config,
		Status:           req.Status,
	}
	err := s.repo.createLLM(ctx, &llm)
	if err != nil {
		logs.Errorf("create llm error: %v", err)
		return nil, errs.DBError
	}
	return llm, nil
}

func (s *service) listLLMs(ctx context.Context, userID uuid.UUID, req ListLLMsReq) (*ListLLMsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	filter := LLMFilter{
		ModelType: req.ModelType,
	}
	list, total, err := s.repo.listLLMs(ctx, userID, filter)
	if err != nil {
		logs.Errorf("list llms error: %v", err)
		return nil, errs.DBError
	}
	return &ListLLMsResponse{
		LLMs:  list,
		Total: total,
	}, nil
}

func (s *service) listLLMAll(ctx context.Context, userID uuid.UUID, req ListLLMsReq) ([]*model.LLM, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	filter := LLMFilter{
		ModelType: req.ModelType,
	}
	list, err := s.repo.listLLMAll(ctx, userID, filter)
	if err != nil {
		logs.Errorf("list llms error: %v", err)
		return nil, errs.DBError
	}
	return list, nil
}

func newService() *service {
	return &service{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
