package router

import (
	"app/internal/nodes"

	"github.com/gin-gonic/gin"
)

type NodeRouter struct {
}

func (n *NodeRouter) Register(engine *gin.Engine) {
	nodeGroup := engine.Group("/api/v1/nodes")
	{
		nodeHandler := nodes.NewHandler()
		nodeGroup.POST("/test", nodeHandler.TestNode)
	}
}
