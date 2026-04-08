package skills

import (
	"model"

	"github.com/google/uuid"
)

type SkillResponse struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	BaseDir     string            `json:"baseDir"`
	Status      model.SkillStatus `json:"status"`
	CreatorID   uuid.UUID         `json:"creatorId"`
	CreatedAt   int64             `json:"createdAt"`
	UpdatedAt   int64             `json:"updatedAt"`
}

func toSkillResponse(skill *model.Skill) *SkillResponse {
	return &SkillResponse{
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		BaseDir:     skill.BaseDir,
		Status:      skill.Status,
		CreatorID:   skill.CreatorID,
		CreatedAt:   skill.CreatedAt.UnixMilli(),
		UpdatedAt:   skill.UpdatedAt.UnixMilli(),
	}
}

func toSkillResponses(skills []*model.Skill) []*SkillResponse {
	responses := make([]*SkillResponse, len(skills))
	for i, skill := range skills {
		responses[i] = toSkillResponse(skill)
	}
	return responses
}

type ListSkillResponse struct {
	Total  int64            `json:"total"`
	Skills []*SkillResponse `json:"skills"`
}

type GithubSourceResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RepoUrl     string    `json:"repoUrl"`
	CreatorID   uuid.UUID `json:"creatorId"`
	CreatedAt   int64     `json:"createdAt"`
	UpdatedAt   int64     `json:"updatedAt"`
}

func toGithubSourceResponse(source *model.GitHubSource) *GithubSourceResponse {
	return &GithubSourceResponse{
		ID:          source.ID,
		Name:        source.Name,
		Description: source.Description,
		RepoUrl:     source.RepoUrl,
		CreatorID:   source.CreatorID,
		CreatedAt:   source.CreatedAt.UnixMilli(),
		UpdatedAt:   source.UpdatedAt.UnixMilli(),
	}
}

type ListGithubSourceResponse struct {
	Total   int64                   `json:"total"`
	Sources []*GithubSourceResponse `json:"sources"`
}

func toGithubSourceResponses(sources []*model.GitHubSource) []*GithubSourceResponse {
	responses := make([]*GithubSourceResponse, len(sources))
	for i, source := range sources {
		responses[i] = toGithubSourceResponse(source)
	}
	return responses
}
