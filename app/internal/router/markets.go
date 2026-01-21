package router

import (
	"app/internal/a2a"

	"github.com/gin-gonic/gin"
)

type A2ARouter struct {
}

func (u *A2ARouter) Register(engine *gin.Engine) {
	a2aGroup := engine.Group("/api/v1/a2a")
	{
		a2aHandler := a2a.NewHandler()
		a2aGroup.POST("/getAgentCard", a2aHandler.GetAgentCard)
		a2aGroup.POST("/saveAgentCard", a2aHandler.SaveAgentCard)
		a2aGroup.GET("/list", a2aHandler.ListAgentMarkets)
		a2aGroup.DELETE("/delete/:id", a2aHandler.Delete)
	}
}
