package shared

import "github.com/google/uuid"

type GetKnowledgeBaseRequest struct {
	UserId          uuid.UUID `json:"userId"`
	KnowledgeBaseId uuid.UUID `json:"knowledgeBaseId"`
}

type SearchKnowledgeBaseRequest struct {
	UserId          uuid.UUID `json:"userId"`
	KnowledgeBaseId uuid.UUID `json:"knowledgeBaseId"`
	Query           string    `json:"query"`
}

type SearchKnowledgeBaseResponse struct {
	Results []*SearchKnowledgeBaseResult `json:"results"`
}

type SearchKnowledgeBaseResult struct {
	Content string `json:"content"`
}
