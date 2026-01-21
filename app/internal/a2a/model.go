package a2a

import (
	"model"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/gorms"
	"gorm.io/gorm"
)

type models struct {
	db *gorm.DB
}

func (m *models) delete(id uuid.UUID) error {
	return m.db.Delete(&model.AgentMarket{}, id).Error
}

func (m *models) list() ([]*model.AgentMarket, error) {
	var agentMarkets []*model.AgentMarket
	return agentMarkets, m.db.Find(&agentMarkets).Error
}

func (m *models) getAgentCardByUrl(url string) (*model.AgentMarket, error) {
	var agentMarket model.AgentMarket
	err := m.db.Where("url = ?", url).First(&agentMarket).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &agentMarket, err
}

func (m *models) save(market *model.AgentMarket) error {
	return m.db.Create(market).Error
}

func newModels(db *gorm.DB) *models {
	return &models{
		db: db,
	}
}
