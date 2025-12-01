package router

import (
	"app/internal/auths"

	"github.com/gin-gonic/gin"
)

type AuthRouter struct {
}

func (u *AuthRouter) Register(engine *gin.Engine) {
	userGroup := engine.Group("/api/v1/auth")
	{
		userHandler := auths.NewHandler()
		userGroup.POST("/register", userHandler.Register)
		userGroup.GET("/verify-email", userHandler.VerifyEmail)
		userGroup.POST("/login", userHandler.Login)
		userGroup.POST("/refresh-token", userHandler.RefreshToken)
		userGroup.POST("/forgot-password", userHandler.ForgotPassword)
		userGroup.POST("/verify-code", userHandler.VerifyCode)
		userGroup.POST("/reset-password", userHandler.ResetPassword)
	}
}
