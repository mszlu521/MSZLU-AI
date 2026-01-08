package router

import (
	"app/internal/knowledges"

	"github.com/gin-gonic/gin"
)

type KnowledgeBaseRouter struct {
}

func (u *KnowledgeBaseRouter) Register(engine *gin.Engine) {
	knowledgesGroup := engine.Group("/api/v1/knowledge")
	{
		knowledgesHandler := knowledges.NewHandler()
		knowledgesGroup.POST("/", knowledgesHandler.CreateKnowledgeBase)
		knowledgesGroup.POST("/list", knowledgesHandler.ListKnowledgeBases)
		knowledgesGroup.GET("/:id", knowledgesHandler.GetKnowledgeBase)
		knowledgesGroup.PUT("/:id", knowledgesHandler.UpdateKnowledgeBase)
		knowledgesGroup.POST("/:id/search", knowledgesHandler.SearchKnowledgeBase)
		knowledgesGroup.DELETE("/:id", knowledgesHandler.DeleteKnowledgeBase)
		knowledgesGroup.GET("/:id/documents", knowledgesHandler.ListDocuments)
		knowledgesGroup.POST("/:id/documents", knowledgesHandler.UploadDocuments)
		knowledgesGroup.DELETE("/:id/documents/:documentId", knowledgesHandler.DeleteDocuments)
	}
}
