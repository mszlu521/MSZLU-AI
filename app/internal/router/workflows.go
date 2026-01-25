package router

import (
	"app/internal/workflows"

	"github.com/gin-gonic/gin"
)

type WorkflowRouter struct {
}

func (r *WorkflowRouter) Register(e *gin.Engine) {
	group := e.Group("/api/v1/workflows")
	{
		handler := workflows.NewHandler()
		group.POST("/list", handler.ListWorkflows)
		group.POST("/create", handler.CreateWorkflow)
		group.PUT("/update", handler.UpdateWorkflow)
		group.GET("/:id", handler.GetWorkflow)
		group.DELETE("/:id", handler.DeleteWorkflow)
		group.POST("/save", handler.SaveWorkflow)
		group.POST("/execute", handler.Execute)
	}
}
