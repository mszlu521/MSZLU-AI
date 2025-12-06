package llms

import (
	"app/shared"
	"context"
	"time"

	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/event"
	"github.com/mszlu521/thunder/logs"
)

type PublicService struct {
	repo repository
}

func (s PublicService) GetProviderConfig(e event.Event) (any, error) {
	request := e.Data.(*shared.GetProviderConfigsRequest)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	providerConfig, err := s.repo.getProviderConfig(ctx, request.Provider)
	if err != nil {
		logs.Errorf("get provider config error: %v", err)
		return nil, err
	}
	return providerConfig, nil
}

func NewPublicService() *PublicService {
	return &PublicService{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
