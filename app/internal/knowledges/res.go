package knowledges

import (
	"model"

	"github.com/google/uuid"
)

type ListResp struct {
	KnowledgeBases []*model.KnowledgeBase `json:"knowledgeBases"`
	Total          int64                  `json:"total"`
}

type KnowledgeBaseResponse struct {
	Id                     uuid.UUID         `json:"id"`
	Name                   string            `json:"name"`
	Tags                   []string          `json:"tags"`
	Description            string            `json:"description"`
	EmbeddingModelName     string            `json:"embeddingModelName"`
	EmbeddingModelProvider string            `json:"embeddingModelProvider"`
	ChatModelName          string            `json:"chatModelName"`
	ChatModelProvider      string            `json:"chatModelProvider"`
	StorageType            model.StorageType `json:"storageType"`
	StorageConfig          model.JSON        `json:"storageConfig"`
	DocumentCount          int               `json:"documentCount"`
	TotalSize              int64             `json:"totalSize"`
	CreatedAt              int64             `json:"createdAt"`
	UpdatedAt              int64             `json:"updatedAt"`
	CreatorId              uuid.UUID         `json:"creatorId"`
}

type ListDocumentsResp struct {
	Documents []*model.Document `json:"items"`
	Total     int64             `json:"total"`
}

type SearchResponse struct {
	Query   string          `json:"query"`
	Results []*SearchResult `json:"results"`
	Total   int64           `json:"total"`
	Took    int64           `json:"took"` //耗时
	KbId    uuid.UUID       `json:"kbId"`
}

type SearchResult struct {
	Id         uuid.UUID       `json:"id"`
	DocumentId uuid.UUID       `json:"documentId"`
	Content    string          `json:"content"`
	Score      float64         `json:"score"`
	Metadata   model.JSON      `json:"metadata"`
	Position   int             `json:"position"`
	Document   *model.Document `json:"document"`
}
