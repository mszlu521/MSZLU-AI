package tools

import (
	"app/shared"
	"context"
	"model"

	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/event"
)

type PublicService struct {
	repo repository
}

func (s *PublicService) GetToolsByIds(e event.Event) (any, error) {
	request := e.Data.(*shared.GetToolsByIdsRequest)
	if len(request.Ids) == 0 {
		return []*model.Tool{}, nil
	}
	toolsList, err := s.repo.getToolsByIds(context.Background(), request.Ids)
	return toolsList, err
}

func NewPublicService() *PublicService {
	return &PublicService{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
