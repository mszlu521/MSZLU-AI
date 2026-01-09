package knowledges

import (
	"context"
	"model"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/gorms"
	"gorm.io/gorm"
)

type models struct {
	db *gorm.DB
}

func (m *models) getDocumentChunksByIds(ctx context.Context, ids []string) ([]*model.DocumentChunk, error) {
	var documentChunks []*model.DocumentChunk
	err := m.db.WithContext(ctx).Where("id in ?", ids).Find(&documentChunks).Error
	if err != nil {
		return nil, err
	}
	//这里我们需要手动排序 保证查询结果和ids的顺序一致
	chunkMap := make(map[string]*model.DocumentChunk)
	for _, chunk := range documentChunks {
		chunkMap[chunk.ID.String()] = chunk
	}
	orderChunks := make([]*model.DocumentChunk, 0, len(ids))
	for _, id := range ids {
		if chunk, ok := chunkMap[id]; ok {
			orderChunks = append(orderChunks, chunk)
		}
	}
	return orderChunks, nil
}

func (m *models) deleteDocuments(ctx context.Context, tx *gorm.DB, userId uuid.UUID, kbId uuid.UUID, documentId uuid.UUID) error {
	if tx == nil {
		tx = m.db
	}
	return tx.WithContext(ctx).Where("id = ? and creator_id=? and kb_id = ?", documentId, userId, kbId).Unscoped().Delete(&model.Document{}).Error
}

func (m *models) deleteDocumentChunks(ctx context.Context, tx *gorm.DB, kbId uuid.UUID, documentId uuid.UUID) error {
	if tx == nil {
		tx = m.db
	}
	//如果不想要通过deleted_at进行软删除，可以加上Unscoped
	return tx.WithContext(ctx).Where("document_id = ? and kb_id = ?", documentId, kbId).Unscoped().Delete(&model.DocumentChunk{}).Error
}

func (m *models) getDocument(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, documentId uuid.UUID) (*model.Document, error) {
	var doc model.Document
	err := m.db.WithContext(ctx).Where("id = ? and creator_id=? and kb_id = ?", documentId, userId, kbId).First(&doc).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &doc, err
}

func (m *models) transaction(ctx context.Context, f func(tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(f)
}

func (m *models) createDocumentChunks(ctx context.Context, chunks []*model.DocumentChunk) error {
	return m.db.WithContext(ctx).CreateInBatches(chunks, len(chunks)).Error
}

func (m *models) createDocument(ctx context.Context, doc *model.Document) error {
	return m.db.WithContext(ctx).Create(doc).Error
}

func (m *models) updateDocumentStatus(ctx context.Context, id uuid.UUID, status model.DocumentStatus) error {
	return m.db.WithContext(ctx).Model(&model.Document{}).Where("id = ?", id).Update("status", status).Error
}

func (m *models) listDocuments(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, filter DocumentFilter) ([]*model.Document, int64, error) {
	var documents []*model.Document
	var count int64
	query := m.db.WithContext(ctx).Model(&model.Document{})
	if filter.Search != "" {
		query = query.Where("name LIKE ?", "%"+filter.Search+"%")
	}
	query = query.Where("kb_id = ? and creator_id = ?", kbId, userId)
	query = query.Count(&count)
	query = query.Limit(filter.Limit).Offset(filter.Offset)
	return documents, count, query.Find(&documents).Error
}

type DocumentFilter struct {
	Limit  int
	Offset int
	Search string
	Status string
}

func (m *models) deleteKnowledgeBase(ctx context.Context, id uuid.UUID) error {
	return m.db.WithContext(ctx).Delete(&model.KnowledgeBase{}, id).Error
}

func (m *models) updateKnowledgeBase(ctx context.Context, kb *model.KnowledgeBase) error {
	return m.db.WithContext(ctx).Updates(kb).Error
}

func (m *models) getKnowledgeBase(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*model.KnowledgeBase, error) {
	var kb model.KnowledgeBase
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&kb).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &kb, err
}

func (m *models) countKnowledgeBaseDocuments(ctx context.Context, id uuid.UUID) (int64, int64, error) {
	var docCount int64
	var totalSize int64
	err := m.db.WithContext(ctx).Model(&model.Document{}).Where("kb_id = ?", id).Count(&docCount).Error
	if err != nil {
		return 0, 0, err
	}
	err = m.db.WithContext(ctx).Model(&model.Document{}).Where("kb_id = ?", id).Select("COALESCE(sum(size),0)").Scan(&totalSize).Error
	return docCount, totalSize, err
}

func (m *models) listKnowledgeBases(ctx context.Context, userId uuid.UUID, filter KnowledgeBaseFilter) ([]*model.KnowledgeBase, int64, error) {
	var kbs []*model.KnowledgeBase
	var count int64
	query := m.db.WithContext(ctx).Model(&model.KnowledgeBase{})
	if filter.Search != "" {
		query = query.Where("name LIKE ?", "%"+filter.Search+"%")
	}
	query = query.Where("creator_id = ?", userId)
	query = query.Count(&count)
	query = query.Limit(filter.Limit).Offset(filter.Offset)
	return kbs, count, query.Find(&kbs).Error
}

type KnowledgeBaseFilter struct {
	Limit  int
	Offset int
	Search string
}

func (m *models) createKnowledgeBase(ctx context.Context, kb *model.KnowledgeBase) error {
	return m.db.WithContext(ctx).Create(kb).Error
}

func newModels(db *gorm.DB) *models {
	return &models{
		db: db,
	}
}
