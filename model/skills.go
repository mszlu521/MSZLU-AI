package model

import (
	"github.com/google/uuid"
)

// SkillStatus 技能状态
type SkillStatus string

const (
	SkillStatusActive   SkillStatus = "active"
	SkillStatusInactive SkillStatus = "inactive"
)

// Skill 技能定义
type Skill struct {
	BaseModel
	// Name 技能名称（唯一）
	Name string `json:"name" gorm:"column:name;type:varchar(100);not null;uniqueIndex"`
	// Description 技能描述
	Description string `json:"description" gorm:"column:description;type:text"`
	// BaseDir 技能文件基础目录
	BaseDir string `json:"baseDir" gorm:"column:base_dir;type:varchar(512);not null"`
	//SourceId 技能来源，本地来源就写localhost，github来源就写对应的source
	SourceId string `json:"sourceId" gorm:"column:source_id;type:varchar(255);not null"`
	// Status 状态
	Status SkillStatus `json:"status" gorm:"column:status;type:varchar(20);not null;default:'active'"`
	// CreatorID 创建者ID
	CreatorID uuid.UUID `json:"creatorId" gorm:"column:creator_id;type:uuid;not null"`
}

// TableName 返回表名
func (Skill) TableName() string {
	return "skills"
}

// SkillWithAgentStatus 带 Agent 关联状态的技能
type SkillWithAgentStatus struct {
	Skill
	IsAssociated bool `json:"isAssociated"`
}

// GitHubSource GitHub 源定义
type GitHubSource struct {
	BaseModel
	// Name 源名称（唯一）
	Name string `json:"name" gorm:"column:name;type:varchar(100);not null;uniqueIndex"`
	// RepoUrl GitHub 仓库 URL
	RepoUrl string `json:"repoUrl" gorm:"column:repo_url;type:varchar(512);not null"`
	// Description 源描述
	Description string `json:"description" gorm:"column:description;type:text"`
	// CreatorID 创建者 ID
	CreatorID uuid.UUID `json:"creatorId" gorm:"column:creator_id;type:uuid;not null"`
}

// TableName 返回表名
func (GitHubSource) TableName() string {
	return "github_sources"
}
