package a2a

import (
	"model"

	"github.com/google/uuid"
)

type repository interface {
	getAgentCardByUrl(url string) (*model.AgentMarket, error)
	save(market *model.AgentMarket) error
	list() ([]*model.AgentMarket, error)
	delete(id uuid.UUID) error
}
