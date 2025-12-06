package llms

import (
	"context"
	"model"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/gorms"
	"gorm.io/gorm"
)

type models struct {
	db *gorm.DB
}

func (m *models) getProviderConfig(ctx context.Context, provider string) (*model.ProviderConfig, error) {
	var pc model.ProviderConfig
	err := m.db.WithContext(ctx).Where("provider = ? ", provider).First(&pc).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &pc, err
}

func (m *models) listLLMs(ctx context.Context, userID uuid.UUID, filter LLMFilter) ([]*model.LLM, int64, error) {
	var llms []*model.LLM
	var count int64
	query := m.db.WithContext(ctx).Model(&model.LLM{})
	query = query.Where("user_id = ?", userID)
	if filter.ModelType != "" {
		query = query.Where("model_type = ?", filter.ModelType)
	}
	if filter.Limit > 0 && filter.Offset >= 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}
	return llms, count, query.Preload("ProviderConfig").Find(&llms).Count(&count).Error
}

type LLMFilter struct {
	ModelType model.LLMType
	Limit     int
	Offset    int
}

func (m *models) createLLM(ctx context.Context, llm *model.LLM) error {
	return m.db.WithContext(ctx).Create(llm).Error
}

func (m *models) listProviderConfigs(ctx context.Context, userId uuid.UUID) ([]*model.ProviderConfig, int64, error) {
	var providerConfigs []*model.ProviderConfig
	var count int64
	query := m.db.WithContext(ctx).Model(&model.ProviderConfig{})
	return providerConfigs, count, query.Where("user_id = ?", userId).Find(&providerConfigs).Count(&count).Error
}

func (m *models) createProviderConfig(ctx context.Context, pc *model.ProviderConfig) error {
	return m.db.WithContext(ctx).Create(pc).Error
}

func newModels(db *gorm.DB) *models {
	return &models{
		db: db,
	}
}
