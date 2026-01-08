package knowledges

import (
	"context"
	"model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository interface {
	createKnowledgeBase(ctx context.Context, m *model.KnowledgeBase) error
	listKnowledgeBases(ctx context.Context, userId uuid.UUID, filter KnowledgeBaseFilter) ([]*model.KnowledgeBase, int64, error)
	getKnowledgeBase(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*model.KnowledgeBase, error)
	countKnowledgeBaseDocuments(ctx context.Context, id uuid.UUID) (int64, int64, error)
	updateKnowledgeBase(ctx context.Context, kb *model.KnowledgeBase) error
	deleteKnowledgeBase(ctx context.Context, id uuid.UUID) error
	listDocuments(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, filter DocumentFilter) ([]*model.Document, int64, error)
	createDocument(ctx context.Context, doc *model.Document) error
	updateDocumentStatus(ctx context.Context, id uuid.UUID, status model.DocumentStatus) error
	createDocumentChunks(ctx context.Context, chunks []*model.DocumentChunk) error
	getDocument(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, documentId uuid.UUID) (*model.Document, error)
	transaction(ctx context.Context, f func(tx *gorm.DB) error) error
	deleteDocuments(ctx context.Context, tx *gorm.DB, userId uuid.UUID, kbId uuid.UUID, documentId uuid.UUID) error
	deleteDocumentChunks(ctx context.Context, tx *gorm.DB, kbId uuid.UUID, documentId uuid.UUID) error
}
