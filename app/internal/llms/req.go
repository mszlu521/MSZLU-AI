package llms

import (
	"model"

	"github.com/google/uuid"
)

type CreateProviderConfigReq struct {
	Name        string          `json:"name"`
	Provider    string          `json:"provider"`
	Description string          `json:"description"`
	APIKey      string          `json:"apiKey"`
	APIBase     string          `json:"apiBase"`
	Status      model.LLMStatus `json:"status"`
}

type CreateLLMReq struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	ProviderConfigID uuid.UUID       `json:"providerConfigId"`
	ModelName        string          `json:"modelName"`
	ModelType        model.LLMType   `json:"modelType"`
	Config           model.LLMConfig `json:"config"`
	Status           model.LLMStatus `json:"status"`
}

type ListLLMsReq struct {
	ModelType model.LLMType `json:"modelType"`
}
