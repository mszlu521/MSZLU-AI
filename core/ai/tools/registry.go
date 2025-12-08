package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mszlu521/thunder/ai/einos"
)

var _register *Registry

type Registry struct {
	tools []einos.InvokeParamTool
}

func RegisterSystemTools(inputs ...einos.InvokeParamTool) {
	var tools []einos.InvokeParamTool
	tools = append(tools, inputs...)
	_register = &Registry{tools: tools}
}

// FindTool 写一个根据工具名称获取工具
func FindTool(toolName string) einos.InvokeParamTool {
	for _, t := range _register.tools {
		info, _ := t.Info(context.Background())
		if info.Name == toolName {
			return t
		}
	}
	return nil
}

// GetTools 获取所有的工具
func GetTools() []tool.BaseTool {
	var tools []tool.BaseTool
	for _, t := range _register.tools {
		tools = append(tools, t)
	}
	return tools
}
