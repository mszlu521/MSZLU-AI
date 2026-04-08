package skills

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

func (m *models) listGithubSources(ctx context.Context, filter GithubSourceFilter) ([]*model.GitHubSource, int64, error) {
	var sources []*model.GitHubSource
	var total int64
	query := m.db.WithContext(ctx).Model(&model.GitHubSource{})
	if filter.Name != "" {
		query = query.Where("name like ?", "%"+filter.Name+"%")
	}
	query = query.Count(&total)
	query = query.Limit(filter.Limit).Offset(filter.Offset).Find(&sources)
	return sources, total, query.Error
}

type GithubSourceFilter struct {
	Name   string
	Limit  int
	Offset int
}

func (m *models) deleteGithubSource(ctx context.Context, id uuid.UUID) error {
	return m.db.WithContext(ctx).Delete(&model.GitHubSource{}, id).Error
}

func (m *models) findGithubSourceById(ctx context.Context, id uuid.UUID) (*model.GitHubSource, error) {
	var source model.GitHubSource
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&source).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &source, err
}

func (m *models) updateGithubSource(ctx context.Context, source *model.GitHubSource) error {
	return m.db.WithContext(ctx).Save(source).Error
}

func (m *models) findGithubSourceByName(ctx context.Context, name string) (*model.GitHubSource, error) {
	var source model.GitHubSource
	err := m.db.WithContext(ctx).Where("name = ?", name).First(&source).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &source, err
}

func (m *models) createGithubSource(ctx context.Context, source *model.GitHubSource) error {
	return m.db.WithContext(ctx).Create(source).Error
}

func (m *models) transaction(ctx context.Context, f func(tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(f)
}

func (m *models) delete(ctx context.Context, id uuid.UUID) error {
	return m.db.WithContext(ctx).Delete(&model.Skill{}, id).Error
}

func (m *models) getSkill(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	var skill model.Skill
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&skill).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &skill, err
}

func (m *models) update(ctx context.Context, skill *model.Skill) error {
	return m.db.WithContext(ctx).Save(skill).Error
}

func (m *models) listAll(ctx context.Context, userId uuid.UUID) ([]*model.Skill, error) {
	var skills []*model.Skill
	return skills, m.db.WithContext(ctx).Where("creator_id = ?", userId).Find(&skills).Error
}

func (m *models) list(ctx context.Context, userId uuid.UUID, filter SkillFilter) ([]*model.Skill, int64, error) {
	var skills []*model.Skill
	var total int64
	query := m.db.WithContext(ctx).Model(&model.Skill{}).Where("creator_id = ?", userId)
	if filter.Name != "" {
		query = query.Where("name like ?", "%"+filter.Name+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	query = query.Count(&total)
	query = query.Limit(filter.Limit).Offset(filter.Offset).Find(&skills)
	return skills, total, query.Error
}

type SkillFilter struct {
	Name   string
	Status model.SkillStatus
	Limit  int
	Offset int
}

func (m *models) findByName(ctx context.Context, name string) (*model.Skill, error) {
	var skill model.Skill
	err := m.db.WithContext(ctx).Where("name = ?", name).First(&skill).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &skill, err
}

func (m *models) create(ctx context.Context, skill *model.Skill) error {
	return m.db.WithContext(ctx).Create(skill).Error
}

func newModels(db *gorm.DB) *models {
	return &models{
		db: db,
	}
}
