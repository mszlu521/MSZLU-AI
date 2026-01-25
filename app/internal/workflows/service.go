package workflows

import (
	"common/biz"
	"context"
	"core/ai"
	"model"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/errs"
	"github.com/mszlu521/thunder/logs"
	"github.com/mszlu521/thunder/res"
)

type service struct {
	repo repository
}

func (s *service) createWorkflow(ctx context.Context, userId uuid.UUID, req *createWorkflowReq) (any, error) {
	m := &model.Workflow{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		UserID: userId,
		Data: &model.Graph{
			Nodes: make([]*model.Node, 0),
			Edges: make([]*model.Edge, 0),
		},
		Config: make(model.JSON),
		Type:   model.WorkflowTypeNormal,
		Status: model.WorkflowStatusValid,
	}
	if req.Name == "" {
		m.Name = "新建工作流"
	} else {
		m.Name = req.Name
	}
	if req.Description == "" {
		m.Description = "新建工作流"
	} else {
		m.Description = req.Description
	}
	err := s.repo.createWorkflow(ctx, m)
	if err != nil {
		logs.Errorf("create workflow error: %v", err)
		return m, errs.DBError
	}
	return m, nil
}

func (s *service) listWorkflows(ctx context.Context, userId uuid.UUID, l *listReq) (res.Page, error) {
	if l.Page <= 0 {
		l.Page = 1
	}
	if l.PageSize <= 0 {
		l.PageSize = 10
	}
	list, total, err := s.repo.list(ctx, &Filter{
		UserId: userId,
		Limit:  l.PageSize,
		Offset: (l.Page - 1) * l.PageSize,
	})
	if err != nil {
		logs.Errorf("list workflows error: %v", err)
		return res.Page{}, errs.DBError
	}
	return res.Page{
		List:        list,
		CurrentPage: int64(l.Page),
		PageSize:    int64(l.PageSize),
		Total:       total,
	}, nil
}

func (s *service) updateWorkflow(ctx context.Context, userId uuid.UUID, u *updateReq) (any, error) {
	wf, err := s.repo.getWorkflow(ctx, userId, u.Id)
	if err != nil {
		logs.Errorf("get workflow error: %v", err)
		return nil, errs.DBError
	}
	if u.Name != "" {
		wf.Name = u.Name
	}
	if u.Description != "" {
		wf.Description = u.Description
	}
	err = s.repo.updateWorkflow(ctx, wf)
	if err != nil {
		logs.Errorf("update workflow error: %v", err)
		return nil, errs.DBError
	}
	return wf, nil
}

func (s *service) getWorkflow(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*model.Workflow, error) {
	wf, err := s.repo.getWorkflow(ctx, userId, id)
	if err != nil {
		logs.Errorf("get workflow error: %v", err)
		return nil, errs.DBError
	}
	if wf == nil {
		return nil, biz.ErrWorkflowNotFound
	}
	return wf, nil
}

func (s *service) deleteWorkflow(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	err := s.repo.deleteWorkflow(ctx, userId, id)
	if err != nil {
		logs.Errorf("delete workflow error: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) saveWorkflow(ctx context.Context, userId uuid.UUID, sr *saveReq) error {
	workflow, err := s.getWorkflow(ctx, userId, sr.Id)
	if err != nil {
		logs.Errorf("get workflow error: %v", err)
		return errs.DBError
	}
	if workflow == nil {
		return biz.ErrWorkflowNotFound
	}
	if sr.Name != "" {
		workflow.Name = sr.Name
	}
	if sr.Description != "" {
		workflow.Description = sr.Description
	}
	if sr.Data != nil {
		workflow.Data = sr.Data
	}
	if sr.Version != 0 {
		workflow.Version = sr.Version
	}
	err = s.repo.updateWorkflow(ctx, workflow)
	if err != nil {
		logs.Errorf("update workflow error: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) execute(ctx context.Context, userId uuid.UUID, e *executeReq) (any, error) {
	//执行工作流，我们需要开发一个工作流的执行器，用于执行工作流
	result, err := ai.Executor.Execute(e.Data)
	if err != nil {
		logs.Errorf("execute workflow error: %v", err)
		return nil, err
	}
	return result, nil
}

func newService() *service {
	return &service{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
