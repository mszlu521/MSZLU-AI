package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type StringArrayJSON []string

// Value  写入 PG 时调用
func (s StringArrayJSON) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan 从 PG 读取时调用
func (s *StringArrayJSON) Scan(value interface{}) error {
	if value == nil {
		*s = StringArrayJSON{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan StringArrayJSON")
	}

	return json.Unmarshal(bytes, s)
}

type StorageType string

const (
	StorageTypeElasticSearch StorageType = "es"
	StorageTypeMilvus        StorageType = "milvus"
)

type KnowledgeBase struct {
	BaseModel
	CreatorID              uuid.UUID           `json:"creatorId" gorm:"column:creator_id;type:uuid;not null;index"`
	Name                   string              `json:"name" gorm:"column:name;type:varchar(255);not null;index"`
	Description            string              `json:"description" gorm:"column:description;type:text"`
	ChatModelName          string              `json:"chatModelName" gorm:"column:chat_model_name;type:varchar(255)"`
	ChatModelProvider      string              `json:"chatModelProvider" gorm:"column:chat_model_provider;type:varchar(50)"`
	EmbeddingModelName     string              `json:"embeddingModelName" gorm:"column:embedding_model_name;type:varchar(255)"`
	EmbeddingModelProvider string              `json:"embeddingModelProvider" gorm:"column:embedding_model_provider;type:varchar(50)"`
	EmbeddingDimension     int                 `json:"embeddingDimension" gorm:"column:embedding_dimension;type:integer;not null"`
	StorageType            StorageType         `json:"storageType" gorm:"column:storage_type;type:varchar(50);not null;default:'es'"`
	StorageConfig          JSON                `json:"storageConfig" gorm:"column:storage_config;type:jsonb"`
	DocumentCount          uint                `json:"documentCount" gorm:"column:document_count;type:integer;not null;default:0"`
	Tags                   StringArrayJSON     `json:"tags" gorm:"column:tags;type:jsonb"`
	Status                 KnowledgeBaseStatus `json:"status" gorm:"column:status;type:varchar(20);not null;default:'active'"`

	// 关联关系
	Agents []Agent `json:"agents" gorm:"many2many:agent_knowledge_bases;"`
}

type KnowledgeBaseStatus string

const (
	KnowledgeBaseStatusActive   KnowledgeBaseStatus = "active"
	KnowledgeBaseStatusDisabled KnowledgeBaseStatus = "disabled"
)

// TableName 返回表名
func (*KnowledgeBase) TableName() string {
	return "knowledge_bases"
}

// Document 存储原始文档的元数据
type Document struct {
	BaseModel
	// 1. 归属信息
	KnowledgeBaseID uuid.UUID `json:"knowledgeBaseId" gorm:"column:kb_id;type:uuid;not null;index"` // 归属哪个知识库
	CreatorID       uuid.UUID `gorm:"type:uuid;not null;index"`
	// 2. 文件基本信息
	Name       string `json:"name" gorm:"column:name;type:varchar(255);not null"`          // 文件名: "员工手册.pdf"
	FileType   string `json:"fileType" gorm:"column:file_type;type:varchar(50);not null"`  // 后缀: pdf, docx, md
	Size       int64  `json:"size" gorm:"column:size;type:bigint;not null;default:0"`      // 文件大小(字节)
	TokenCount int    `json:"tokenCount" gorm:"column:token_count;type:integer;default:0"` // 总 Token 数消耗统计
	// 3. 存储与去重
	StorageKey string `json:"storageKey" gorm:"column:storage_key;type:varchar(512);not null"` // S3/OSS 上的路径 key
	FileHash   string `json:"fileHash" gorm:"column:file_hash;type:varchar(64);index"`         // SHA256 Hash，用于防止重复上传
	// 4. 处理状态
	Status       DocumentStatus `json:"status" gorm:"column:status;type:varchar(20);not null;default:'pending';index"`
	ErrorMessage string         `json:"errorMessage" gorm:"column:error_message;type:text"` // 如果失败，存错误堆栈
	// 5. 解析结果元数据 (可选)
	// 存放如: {"page_count": 10, "author": "CEO"}
	MetaInfo JSON `json:"metaInfo" gorm:"column:meta_info;type:jsonb"`
	// 6. 是否启用
	Enabled bool `json:"enabled" gorm:"column:enabled;type:boolean;not null;default:true"` // 软开关，关闭后检索不到
	// 关联
	Chunks []DocumentChunk `json:"chunks,omitempty" gorm:"foreignKey:DocumentID"`
}
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusCompleted  DocumentStatus = "completed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

func (*Document) TableName() string {
	return "documents"
}

// DocumentChunk 存储切分后的片段 (PostgreSQL 侧备份与管理)
type DocumentChunk struct {
	BaseModel
	// 1. 关联关系
	DocumentID      uuid.UUID `json:"documentId" gorm:"column:document_id;type:uuid;not null;index"`
	KnowledgeBaseID uuid.UUID `json:"knowledgeBaseId" gorm:"column:kb_id;type:uuid;not null;index"` // 冗余字段，为了方便按库查询
	// 2. 索引同步 (关键字段)
	// 记录该切片在 ES 中的 ID (_id)，用于后续的更新或删除操作
	ElasticSearchID string `json:"esId" gorm:"column:es_id;type:varchar(100);index"`
	// 3. 内容数据
	ChunkIndex int    `json:"chunkIndex" gorm:"column:chunk_index;type:integer;not null"` // 切片在原文中的顺序 (0, 1, 2...)
	Content    string `json:"content" gorm:"column:content;type:text;not null"`           // 切片文本内容 (PG中存一份用于展示/编辑)
	// 4. 向量化相关
	TokenCount int `json:"tokenCount" gorm:"column:token_count;type:integer"` // 该切片的 Token 数
	// 5. 元数据 (Metadata)
	// 极其重要！这里存放 {"page_num": 1, "heading": "第一章", "image_url": "..."}
	// 这些数据会同步写入 ES 的 metadata 字段，用于 filter
	MetaInfo JSON        `json:"metaInfo" gorm:"column:meta_info;type:jsonb"`
	Status   ChunkStatus `gorm:"column:status;type:varchar(20);not null;default:'pending'"`
}
type ChunkStatus string

const (
	ChunkStatusPending  ChunkStatus = "pending"
	ChunkStatusEmbedded ChunkStatus = "embedded"
	ChunkStatusDeleted  ChunkStatus = "deleted"
	ChunkStatusDisabled ChunkStatus = "disabled"
)

func (*DocumentChunk) TableName() string {
	return "document_chunks"
}
