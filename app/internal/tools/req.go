package tools

import "model"

type CreateToolReq struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	ToolType    model.ToolType   `json:"toolType"`
	IsEnable    bool             `json:"isEnable"`
	McpConfig   *model.McpConfig `json:"mcpConfig"`
}

type ListToolsReq struct {
	Name     string         `json:"name" form:"name"`
	Type     model.ToolType `json:"type" form:"type"`
	Page     int            `json:"page" form:"page"`
	PageSize int            `json:"pageSize" form:"pageSize"`
}

type UpdateToolReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TestToolReq struct {
	Params map[string]interface{} `json:"params"`
}
