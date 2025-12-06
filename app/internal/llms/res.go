package llms

import "model"

type ListProviderConfigsResponse struct {
	Total           int64                   `json:"total"`
	ProviderConfigs []*model.ProviderConfig `json:"providerConfigs"`
}

type ListLLMsResponse struct {
	Total int64        `json:"total"`
	LLMs  []*model.LLM `json:"llms"`
}
