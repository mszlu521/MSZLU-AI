package router

import (
	"app/internal/llms"

	"github.com/mszlu521/thunder/event"
)

type Event struct {
}

func (u *Event) Register() {
	//TODO 注册事件相关的路由
	llmService := llms.NewPublicService()
	event.Register("getProviderConfig", llmService.GetProviderConfig)
}
