package shared

import (
	"model"

	"github.com/google/uuid"
)

type GetProviderConfigsRequest struct {
	LLMType   model.LLMType
	Provider  string
	ModelName string
}

type LLMParams struct {
	Provider  string
	Model     string
	ModelType model.LLMType
	UserId    uuid.UUID
}

type EmbeddingConfigResponse struct {
	Model *model.LLM
}
