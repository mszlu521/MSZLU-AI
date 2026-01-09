package router

import (
	"app/internal/agents"

	"github.com/gin-gonic/gin"
)

type AgentRouter struct {
}

func (u *AgentRouter) Register(engine *gin.Engine) {
	agentsGroup := engine.Group("/api/v1/agents")
	{
		agentsHandler := agents.NewHandler()
		agentsGroup.POST("/create", agentsHandler.CreateAgent)
		agentsGroup.POST("/list", agentsHandler.ListAgents)
		agentsGroup.GET("/:id", agentsHandler.GetAgent)
		agentsGroup.PUT("/update", agentsHandler.UpdateAgent)
		agentsGroup.POST("/chat", agentsHandler.AgentMessage)
		agentsGroup.POST("/:id/tools/batch", agentsHandler.UpdateAgentTool)
		agentsGroup.POST("/:id/knowledge-bases", agentsHandler.AddAgentKnowledgeBase)
		agentsGroup.DELETE("/:id/knowledge-bases/:kbId", agentsHandler.DeleteAgentKnowledgeBase)
	}
}
