package router

import (
	"app/internal/skills"

	"github.com/gin-gonic/gin"
)

type SkillRouter struct {
}

func (r *SkillRouter) Register(engin *gin.Engine) {
	skillsGroup := engin.Group("/api/v1/skills")
	{
		handler := skills.NewHandler()
		skillsGroup.POST("", handler.CreateSkill)
		skillsGroup.POST("/list", handler.ListSkills)
		skillsGroup.GET("/all", handler.ListSkillsAll)
		skillsGroup.PUT("", handler.UpdateSkill)
		skillsGroup.DELETE("/:id", handler.DeleteSkill)
		skillsGroup.POST("/install", handler.InstallSkill)
		//github skill管理
		skillsGroup.POST("/sources", handler.CreateGithubSources)
		skillsGroup.PUT("/sources", handler.UpdateGithubSources)
		skillsGroup.DELETE("/sources/:id", handler.DeleteGithubSources)
		skillsGroup.POST("/sources/list", handler.ListGithubSources)
		skillsGroup.GET("/sources/:id", handler.GetGithubSource)
	}
}
