package tools

import (
	"common/biz"
	"context"
	"core/ai/tools"
	"encoding/json"
	"model"
	"time"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/errs"
	"github.com/mszlu521/thunder/logs"
	"github.com/mszlu521/thunder/res"
)

type service struct {
	repo repository
}

func (s *service) createTool(ctx context.Context, userId uuid.UUID, req CreateToolReq) (*model.Tool, error) {
	//先查询tool名字是否存在 防止重复
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	toolInfo, err := s.repo.getToolByName(ctx, req.Name)
	if err != nil {
		logs.Errorf("get tool by name error: %v", err)
		return nil, errs.DBError
	}
	if toolInfo != nil {
		return nil, biz.ErrToolNameExisted
	}
	//创建tool
	tool := model.Tool{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		ToolType:  req.ToolType,
		IsEnable:  true,
		CreatorID: userId,
	}
	//这个地方我们需要先检查tool是否存在，启动时，我们将tool注册了
	//注意 这个地方 我们只能注册 我们系统中已经开发好的tool
	//这个地方因为有mcp工具的存在，所以这里我们先判断一下
	if req.ToolType == model.McpToolType {
		if req.McpConfig != nil {
			tool.McpConfig = req.McpConfig
		}
		tool.Name = req.Name
		tool.Description = req.Description
	} else {
		//这是系统工具
		invokeParamTool := tools.FindTool(req.Name)
		if invokeParamTool == nil {
			return nil, biz.ErrToolNotExisted
		}
		info, err := invokeParamTool.Info(ctx)
		if err != nil {
			logs.Errorf("get tool info error: %v", err)
			return nil, errs.DBError
		}
		tool.Name = info.Name
		tool.Description = info.Desc
		tool.ParametersSchema = invokeParamTool.Params()
	}
	err = s.repo.createTool(ctx, &tool)
	if err != nil {
		logs.Errorf("create tool error: %v", err)
		return nil, errs.DBError
	}
	return &tool, nil
}

func (s *service) listTools(ctx context.Context, userID uuid.UUID, req ListToolsReq) (*res.Page, error) {
	//构建过滤条件
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	filter := toolFilter{
		Name:     req.Name,
		Limit:    req.PageSize,
		Offset:   (req.Page - 1) * req.PageSize,
		ToolType: req.Type,
	}
	toolList, total, err := s.repo.listTools(ctx, userID, filter)
	if err != nil {
		logs.Errorf("list tools error: %v", err)
		return nil, errs.DBError
	}
	return &res.Page{
		List:        toolList,
		Total:       total,
		CurrentPage: int64(req.Page),
		PageSize:    int64(req.PageSize),
	}, nil
}

func (s *service) updateTool(ctx context.Context, userID uuid.UUID, id uuid.UUID, req UpdateToolReq) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	//先根据id进行查询
	toolInfo, err := s.repo.getTool(ctx, userID, id)
	if err != nil {
		logs.Errorf("get tool error: %v", err)
		return nil, errs.DBError
	}
	if toolInfo == nil {
		return nil, biz.ErrToolNotExisted
	}
	//然后判断名字是否重复
	if req.Name != toolInfo.Name {
		toolInfo1, err := s.repo.getToolByName(ctx, req.Name)
		if err != nil {
			logs.Errorf("get tool by name error: %v", err)
			return nil, errs.DBError
		}
		if toolInfo1 != nil {
			return nil, biz.ErrToolNameExisted
		}
	}
	toolInfo.Name = req.Name
	toolInfo.Description = req.Description
	err = s.repo.updateTool(ctx, toolInfo)
	if err != nil {
		logs.Errorf("update tool error: %v", err)
		return nil, errs.DBError
	}
	return toolInfo, nil
}

func (s *service) deleteTool(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := s.repo.deleteTool(ctx, userID, id)
	if err != nil {
		logs.Errorf("delete tool error: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) testTool(ctx context.Context, userId uuid.UUID, id uuid.UUID, req TestToolReq) (*TestToolResponse, error) {
	//获取 tool
	toolInfo, err := s.repo.getTool(ctx, userId, id)
	if err != nil {
		logs.Errorf("get tool error: %v", err)
		return nil, errs.DBError
	}
	//查找系统中注册的tool
	invokeParamTool := tools.FindTool(toolInfo.Name)
	if invokeParamTool == nil {
		return nil, biz.ErrToolNotExisted
	}
	//参数转换成json
	params, _ := json.Marshal(req.Params)
	result, err := invokeParamTool.InvokableRun(ctx, string(params))
	if err != nil {
		logs.Errorf("invoke tool error: %v", err)
		return &TestToolResponse{
			Message: err.Error(),
			Success: false,
			Data:    nil,
		}, nil
	}
	return &TestToolResponse{
		Message: "success",
		Success: true,
		Data:    result,
	}, nil
}

func newService() *service {
	return &service{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
