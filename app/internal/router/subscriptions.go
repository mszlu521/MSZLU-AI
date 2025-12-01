package router

import (
	"app/internal/subscriptions"

	"github.com/gin-gonic/gin"
)

type SubscriptionRouter struct {
}

func (u *SubscriptionRouter) Register(engine *gin.Engine) {
	handler := subscriptions.NewHandler()
	group := engine.Group("/api/v1/subscription")
	{
		group.GET("/current", handler.GetUserSubscription)
	}
}
