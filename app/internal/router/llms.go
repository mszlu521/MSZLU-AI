package router

import (
	"app/internal/llms"

	"github.com/gin-gonic/gin"
)

type LLMRouter struct {
}

func (u *LLMRouter) Register(engine *gin.Engine) {
	llmGroup := engine.Group("/api/v1/provider-configs")
	{
		llmHandler := llms.NewHandler()
		llmGroup.POST("/", llmHandler.CreateProviderConfig)
		llmGroup.GET("/", llmHandler.ListProviderConfigs)
	}
	llmsGroup := engine.Group("/api/v1/llms")
	{
		llmsHandler := llms.NewHandler()
		llmsGroup.POST("/", llmsHandler.CreateLLM)
		llmsGroup.GET("/", llmsHandler.ListLLMs)
	}
}
