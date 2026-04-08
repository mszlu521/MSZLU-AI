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
		agentsGroup.DELETE("/:id", agentsHandler.DeleteAgent)
		agentsGroup.POST("/chat", agentsHandler.AgentMessage)
		agentsGroup.POST("/:id/tools/batch", agentsHandler.UpdateAgentTool)
		agentsGroup.DELETE("/:id/tools/:toolId", agentsHandler.DeleteAgentTool)
		agentsGroup.POST("/:id/knowledge-bases", agentsHandler.AddAgentKnowledgeBase)
		agentsGroup.DELETE("/:id/knowledge-bases/:kbId", agentsHandler.DeleteAgentKnowledgeBase)
		agentsGroup.POST("/market/add", agentsHandler.AddAgentAgent)
		agentsGroup.POST("/market/delete", agentsHandler.DeleteAgentAgent)
		agentsGroup.POST("/:id/workflows", agentsHandler.AddWorkflowToAgent)
		agentsGroup.DELETE("/:id/workflows/:workflowId", agentsHandler.DeleteWorkflowFromAgent)
		//会话相关
		agentsGroup.POST("/sessions", agentsHandler.CreateSession)
		agentsGroup.GET("/sessions", agentsHandler.ListSessions)
		agentsGroup.GET("/sessions/:sessionId/messages", agentsHandler.GetSessionMessages)
		agentsGroup.DELETE("/sessions/:sessionId", agentsHandler.DeleteSession)
		//skill 关联
		agentsGroup.POST("/:id/skills", agentsHandler.AddSkillToAgent)
		agentsGroup.POST("/:id/skills/:skillId", agentsHandler.DeleteSkillFromAgent)
	}
}
