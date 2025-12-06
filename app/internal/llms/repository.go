package llms

import (
	"context"
	"model"

	"github.com/google/uuid"
)

type repository interface {
	createProviderConfig(ctx context.Context, m *model.ProviderConfig) error
	listProviderConfigs(ctx context.Context, userId uuid.UUID) ([]*model.ProviderConfig, int64, error)
	createLLM(ctx context.Context, llm *model.LLM) error
	listLLMs(ctx context.Context, userID uuid.UUID, filter LLMFilter) ([]*model.LLM, int64, error)
	getProviderConfig(ctx context.Context, provider string) (*model.ProviderConfig, error)
}
