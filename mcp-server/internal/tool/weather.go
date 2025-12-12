package tool

import (
	"context"
	"core/ai/tools"
	"encoding/json"

	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mszlu521/thunder/ai/einos"
)

type WeatherTool struct {
	ApiKey string
	tool   einos.InvokeParamTool
}

func NewWeatherTool(apiKey string) *WeatherTool {
	return &WeatherTool{ApiKey: apiKey}
}
func (w *WeatherTool) Build() mcp.Tool {
	tool := tools.NewWeatherTool(&tools.WeatherConfig{ApiKey: w.ApiKey})
	w.tool = tool
	//转换为mcp tool
	info, _ := tool.Info(context.Background())
	//这里要加上描述，参数这些才行
	params := tool.Params()
	//做一个转换把 eino的转换为mcp 的
	options := ToMCPOptions(params, info.Desc)
	mcpTool := mcp.NewTool(info.Name, options...)
	return mcpTool
}

func ToMCPOptions(params map[string]*schema.ParameterInfo, desc string) []mcp.ToolOption {
	//先处理描述
	var options []mcp.ToolOption
	options = append(options, mcp.WithDescription(desc))
	//处理参数
	for k, v := range params {
		var propertyOptions []mcp.PropertyOption
		if v.Required {
			propertyOptions = append(propertyOptions, mcp.Required())
		}
		propertyOptions = append(propertyOptions, mcp.Description(v.Desc))
		if v.Enum != nil && len(v.Enum) > 0 {
			propertyOptions = append(propertyOptions, mcp.Enum(v.Enum...))
		}
		//这里要判断不同的参数类型，我们先只处理几个
		switch v.Type {
		case schema.String:
			options = append(options, mcp.WithString(k, propertyOptions...))
		case schema.Number:
			options = append(options, mcp.WithNumber(k, propertyOptions...))
		case schema.Boolean:
			options = append(options, mcp.WithBoolean(k, propertyOptions...))
		case schema.Integer:
			options = append(options, mcp.WithNumber(k, propertyOptions...))
		case schema.Array:
			options = append(options, mcp.WithArray(k, propertyOptions...))
		case schema.Object:
			options = append(options, mcp.WithObject(k, propertyOptions...))

		}
	}
	return options
}

func (w *WeatherTool) Invoke(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params, err := json.Marshal(request.GetArguments())
	if err != nil {
		return nil, err
	}
	invokableRun, err := w.tool.InvokableRun(ctx, string(params))
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(invokableRun), nil
}
