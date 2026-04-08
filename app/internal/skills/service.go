package skills

import (
	"common/biz"
	"common/utils"
	"context"
	"fmt"
	"model"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/logs"
	"gorm.io/gorm"
)

type service struct {
	repo repository
}

func (s *service) createSkill(ctx context.Context, userId uuid.UUID, req CreateSkillReq) (*SkillResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	//先检查技能是否可用
	skillInfo, err := s.validateSkillName(req.Name, req.BaseDir)
	if err != nil {
		logs.Errorf("验证技能失败: %v", err)
		return nil, err
	}
	//检查名称在数据库中是否已经存在
	skill, err := s.repo.findByName(ctx, skillInfo.Name)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	if skill != nil {
		return nil, biz.ErrSkillAlreadyExisted
	}
	//创建技能
	skill = &model.Skill{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		Name:        skillInfo.Name,
		Description: skillInfo.Description,
		BaseDir:     req.BaseDir,
		Status:      model.SkillStatusActive,
		CreatorID:   userId,
		SourceId:    "local",
	}
	err = s.repo.create(ctx, skill)
	if err != nil {
		logs.Errorf("创建技能失败: %v", err)
		return nil, err
	}
	return toSkillResponse(skill), nil
}

func (s *service) validateSkillName(skillName string, baseDir string) (*utils.SkillMetadata, error) {
	if skillName == "" {
		return nil, fmt.Errorf("skill name can not be empty")
	}
	if baseDir == "" {
		return nil, fmt.Errorf("skill base dir can not be empty")
	}
	//检查技能目录是否存在
	skillDIr := filepath.Join(baseDir, skillName)
	skillDirInfo, err := os.Stat(skillDIr)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("skill dir not exists")
		}
		return nil, err
	}
	if !skillDirInfo.IsDir() {
		return nil, fmt.Errorf("skill dir is not a directory")
	}
	//检查SKILL.md是否存在
	skillMdPath := filepath.Join(skillDIr, "SKILL.md")
	skillMdContent, err := os.ReadFile(skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("skill md not exists")
		}
		return nil, err
	}
	//解析SKILL.md 获取元数据
	metadata := utils.ParseSkillMd(string(skillMdContent))
	if metadata == nil {
		return nil, fmt.Errorf("skill md is invalid")
	}
	//验证名称是否一致
	if !strings.EqualFold(metadata.Name, skillName) {
		return nil, fmt.Errorf("skill name is not equal")
	}
	return metadata, nil
}

func (s *service) listSkills(ctx context.Context, userId uuid.UUID, req SearchSkillReq) (*ListSkillResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	filter := SkillFilter{
		Name:   req.Name,
		Status: model.SkillStatus(req.Status),
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	skills, total, err := s.repo.list(ctx, userId, filter)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	return &ListSkillResponse{
		Skills: toSkillResponses(skills),
		Total:  total,
	}, nil
}

func (s *service) listSkillsAll(ctx context.Context, userId uuid.UUID) ([]*SkillResponse, error) {
	skills, err := s.repo.listAll(ctx, userId)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	return toSkillResponses(skills), nil
}

func (s *service) updateSkill(ctx context.Context, userId uuid.UUID, req UpdateSkillReq) (*SkillResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	skill, err := s.repo.getSkill(ctx, req.ID)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	if skill == nil {
		return nil, biz.ErrSkillNotFound
	}
	if req.Name != "" && req.Name != skill.Name {
		existSkill, err := s.repo.findByName(ctx, req.Name)
		if err != nil {
			logs.Errorf("查询技能失败: %v", err)
			return nil, err
		}
		if existSkill != nil {
			return nil, biz.ErrSkillAlreadyExisted
		}
		skill.Name = req.Name
	}
	if req.Description != "" {
		skill.Description = req.Description
	}
	if req.BaseDir != "" {
		skill.BaseDir = req.BaseDir
	}
	if req.Status != "" {
		skill.Status = model.SkillStatus(req.Status)
	}
	err = s.repo.update(ctx, skill)
	if err != nil {
		logs.Errorf("更新技能失败: %v", err)
		return nil, err
	}
	return toSkillResponse(skill), nil
}

func (s *service) deleteSkill(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	skill, err := s.repo.getSkill(ctx, id)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return err
	}
	if skill == nil {
		return biz.ErrSkillNotFound
	}
	err = s.repo.transaction(ctx, func(tx *gorm.DB) error {
		err := s.repo.delete(ctx, id)
		if err != nil {
			logs.Errorf("删除技能失败: %v", err)
			return err
		}
		//删除agent关联的技能
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *service) createGithubSources(ctx context.Context, userId uuid.UUID, req CreateGithubSourceReq) (*GithubSourceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	//先检查名称在数据库中是否已经存在
	source, err := s.repo.findGithubSourceByName(ctx, req.Name)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	if source != nil {
		return nil, biz.ErrGithubSourceAlreadyExisted
	}
	source = &model.GitHubSource{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		Name:        req.Name,
		Description: req.Description,
		RepoUrl:     req.RepoUrl,
		CreatorID:   userId,
	}
	err = s.repo.createGithubSource(ctx, source)
	if err != nil {
		logs.Errorf("创建技能失败: %v", err)
		return nil, err
	}
	return toGithubSourceResponse(source), nil
}

func (s *service) updateGithubSources(ctx context.Context, userId uuid.UUID, req UpdateGithubSourceReq) (*GithubSourceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	source, err := s.repo.findGithubSourceById(ctx, req.ID)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	if source == nil {
		return nil, biz.ErrGithubSourceNotFound
	}
	if req.Name != "" && req.Name != source.Name {
		existSource, err := s.repo.findGithubSourceByName(ctx, req.Name)
		if err != nil {
			logs.Errorf("查询技能失败: %v", err)
			return nil, err
		}
		if existSource != nil {
			return nil, biz.ErrGithubSourceAlreadyExisted
		}
		source.Name = req.Name
	}
	if req.Description != "" {
		source.Description = req.Description
	}
	if req.RepoUrl != "" {
		source.RepoUrl = req.RepoUrl
	}
	err = s.repo.updateGithubSource(ctx, source)
	if err != nil {
		logs.Errorf("更新技能失败: %v", err)
		return nil, err
	}
	return toGithubSourceResponse(source), nil
}

func (s *service) deleteGithubSources(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	source, err := s.repo.findGithubSourceById(ctx, id)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return err
	}
	if source == nil {
		return biz.ErrGithubSourceNotFound
	}
	err = s.repo.deleteGithubSource(ctx, id)
	if err != nil {
		logs.Errorf("删除技能失败: %v", err)
		return err
	}
	return nil
}

func (s *service) listGithubSources(ctx context.Context, userId uuid.UUID, req SearchGithubSourceReq) (*ListGithubSourceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	filter := GithubSourceFilter{
		Name:   req.Name,
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	sources, total, err := s.repo.listGithubSources(ctx, filter)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	return &ListGithubSourceResponse{
		Sources: toGithubSourceResponses(sources),
		Total:   total,
	}, nil

}

func (s *service) getGithubSource(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*GithubSourceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	source, err := s.repo.findGithubSourceById(ctx, id)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	if source == nil {
		return nil, biz.ErrGithubSourceNotFound
	}
	return toGithubSourceResponse(source), nil
}

func (s *service) installSkill(ctx context.Context, userId uuid.UUID, req InstallSkillReq) (*SkillResponse, error) {
	//检查技能名称是否已经存在
	skill, err := s.repo.findByName(ctx, req.SkillId)
	if err != nil {
		logs.Errorf("查询技能失败: %v", err)
		return nil, err
	}
	if skill != nil {
		return nil, biz.ErrSkillAlreadyExisted
	}
	//克隆github仓库到临时目录
	tempDir := filepath.Join(req.TargetDir, ".temp", req.SkillId)
	if err := os.RemoveAll(tempDir); err != nil {
		logs.Errorf("删除临时目录失败: %v", err)
		return nil, err
	}
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      req.RepoUrl,
		Depth:    1,
		Progress: nil,
	})
	if err != nil {
		logs.Errorf("克隆仓库失败: %v", err)
		return nil, err
	}
	//从临时目录中复制文件到技能目录
	sourcePath := filepath.Join(tempDir, req.RepoPath)
	targetPath := filepath.Join(req.TargetDir, req.SkillId)
	if err := os.RemoveAll(targetPath); err != nil {
		logs.Errorf("删除目标目录失败: %v", err)
		return nil, err
	}
	if err := utils.CopyDir(sourcePath, targetPath); err != nil {
		logs.Errorf("复制文件失败: %v", err)
		return nil, err
	}
	//清理目录
	if err := os.RemoveAll(tempDir); err != nil {
		logs.Errorf("删除临时目录失败: %v", err)
		return nil, err
	}
	var description string
	//读取技能文件
	skillMdPath := filepath.Join(targetPath, "SKILL.md")
	if content, err := os.ReadFile(skillMdPath); err == nil {
		metadata := utils.ParseSkillMd(string(content))
		if metadata != nil && metadata.Name != "" && metadata.Description != "" {
			description = metadata.Description
		} else {
			return nil, fmt.Errorf("invalid skill metadata")
		}
	} else {
		return nil, fmt.Errorf("invalid skill metadata")
	}
	skill = &model.Skill{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		Name:        req.SkillId,
		Description: description,
		BaseDir:     targetPath,
		SourceId:    req.SourceId,
		Status:      model.SkillStatusActive,
		CreatorID:   userId,
	}
	err = s.repo.create(ctx, skill)
	if err != nil {
		logs.Errorf("创建技能失败: %v", err)
		return nil, err
	}
	return toSkillResponse(skill), nil
}

func newService() *service {
	return &service{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
