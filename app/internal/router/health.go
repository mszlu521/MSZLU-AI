package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mszlu521/thunder/res"
)

type HealthRouter struct {
}

// HealthCheck 健康检查响应
// @Summary 健康检查
// @Description 用于K8s健康检查探针
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "{\"status\":\"ok\"}"
// @Router /health [get]
func (r *HealthRouter) Register(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) {
		res.Success(c, gin.H{
			"status": "ok",
		})
	})
}
