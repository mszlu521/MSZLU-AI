package skills

import (
	"context"
	"model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository interface {
	findByName(ctx context.Context, name string) (*model.Skill, error)
	create(ctx context.Context, skill *model.Skill) error
	list(ctx context.Context, userId uuid.UUID, filter SkillFilter) ([]*model.Skill, int64, error)
	listAll(ctx context.Context, userId uuid.UUID) ([]*model.Skill, error)
	getSkill(ctx context.Context, id uuid.UUID) (*model.Skill, error)
	update(ctx context.Context, skill *model.Skill) error
	transaction(ctx context.Context, f func(tx *gorm.DB) error) error
	delete(ctx context.Context, id uuid.UUID) error
	findGithubSourceByName(ctx context.Context, name string) (*model.GitHubSource, error)
	createGithubSource(ctx context.Context, source *model.GitHubSource) error
	findGithubSourceById(ctx context.Context, id uuid.UUID) (*model.GitHubSource, error)
	updateGithubSource(ctx context.Context, source *model.GitHubSource) error
	deleteGithubSource(ctx context.Context, id uuid.UUID) error
	listGithubSources(ctx context.Context, filter GithubSourceFilter) ([]*model.GitHubSource, int64, error)
}
