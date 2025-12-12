package router

import (
	"app/internal/tools"

	"github.com/gin-gonic/gin"
)

type ToolRouter struct {
}

func (t *ToolRouter) Register(r *gin.Engine) {
	toolsGroup := r.Group("/api/v1/tools")
	{
		toolsHandler := tools.NewHandler()
		toolsGroup.POST("/", toolsHandler.CreateTool)
		toolsGroup.GET("/", toolsHandler.ListTools)
		toolsGroup.PUT("/:id", toolsHandler.UpdateTool)
		toolsGroup.DELETE("/:id", toolsHandler.DeleteTool)
		toolsGroup.POST("/:id/test", toolsHandler.TestTool)
		toolsGroup.GET("/mcp/:mcpId/tools", toolsHandler.GetMcpTools)
	}
}
