package skills

import "github.com/google/uuid"

type CreateSkillReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	BaseDir     string `json:"baseDir"`
}

type SearchSkillReq struct {
	Name     string `json:"name"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Status   string `json:"status"`
}

type UpdateSkillReq struct {
	ID          uuid.UUID `json:"id" binding:"required"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BaseDir     string    `json:"baseDir"`
	Status      string    `json:"status"`
}

type CreateGithubSourceReq struct {
	Name        string `json:"name" binding:"required"`
	RepoUrl     string `json:"repoUrl" binding:"required"`
	Description string `json:"description"`
}

type UpdateGithubSourceReq struct {
	ID          uuid.UUID `json:"id" binding:"required"`
	Name        string    `json:"name"`
	RepoUrl     string    `json:"repoUrl"`
	Description string    `json:"description"`
}

type SearchGithubSourceReq struct {
	Name     string `json:"name"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type InstallSkillReq struct {
	SkillId   string `json:"skillId" binding:"required"`
	SourceId  string `json:"sourceId" binding:"required"`
	TargetDir string `json:"targetDir" binding:"required"`
	RepoUrl   string `json:"repoUrl" binding:"required"`
	RepoPath  string `json:"repoPath" binding:"required"`
}
