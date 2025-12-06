package shared

import "model"

type GetProviderConfigsRequest struct {
	LLMType   model.LLMType
	Provider  string
	ModelName string
}
