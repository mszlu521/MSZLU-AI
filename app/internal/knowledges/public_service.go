package knowledges

import (
	"app/shared"
	"context"

	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/event"
)

type PublicService struct {
	repo repository
}

func (s *PublicService) GetKnowledgeBase(e event.Event) (any, error) {
	request := e.Data.(*shared.GetKnowledgeBaseRequest)
	knowledgeBase, err := s.repo.getKnowledgeBase(context.Background(), request.UserId, request.KnowledgeBaseId)
	return knowledgeBase, err
}

func (s *PublicService) SearchKnowledgeBase(e event.Event) (any, error) {
	request := e.Data.(*shared.SearchKnowledgeBaseRequest)
	kbService := newService()
	response, err := kbService.searchKnowledgeBase(context.Background(), request.UserId, request.KnowledgeBaseId, searchParams{
		Query: request.Query,
	})
	if err != nil {
		return nil, err
	}
	var results []*shared.SearchKnowledgeBaseResult
	for _, v := range response.Results {
		results = append(results, &shared.SearchKnowledgeBaseResult{
			Content: v.Content,
		})
	}
	return &shared.SearchKnowledgeBaseResponse{
		Results: results,
	}, nil
}

func NewPublicService() *PublicService {
	return &PublicService{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
