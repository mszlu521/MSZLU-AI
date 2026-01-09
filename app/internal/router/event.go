package router

import (
	"app/internal/knowledges"
	"app/internal/llms"
	"app/internal/tools"

	"github.com/mszlu521/thunder/event"
)

type Event struct {
}

func (u *Event) Register() {
	//TODO 注册事件相关的路由
	llmService := llms.NewPublicService()
	event.Register("getProviderConfig", llmService.GetProviderConfig)
	event.Register("getEmbeddingConfig", llmService.GetEmbeddingConfig)
	toolService := tools.NewPublicService()
	event.Register("getToolsByIds", toolService.GetToolsByIds)
	knowledgeService := knowledges.NewPublicService()
	event.Register("getKnowledgeBase", knowledgeService.GetKnowledgeBase)
	event.Register("searchKnowledgeBase", knowledgeService.SearchKnowledgeBase)
}
