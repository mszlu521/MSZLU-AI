package main

import (
	"context"
	"core/ai/tools"
	"fmt"

	"github.com/cloudwego/eino-ext/a2a/extension/eino"
	"github.com/cloudwego/eino-ext/a2a/transport/jsonrpc"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	hertzServer "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/mszlu521/thunder/config"
)

func main() {
	//加载etc/config.yml中的配置
	config.Init()
	conf := config.GetConfig()
	addr := fmt.Sprintf("%s:%d", conf.Server.GetHost(), conf.Server.GetPort())
	h := hertzServer.Default(
		hertzServer.WithHostPorts(addr),
		hertzServer.WithSenseClientDisconnection(true),
	)
	ctx := context.Background()
	r, err := jsonrpc.NewRegistrar(ctx, &jsonrpc.ServerConfig{
		Router:        h,
		HandlerPath:   "/a2a",
		AgentCardPath: nil,
	})
	if err != nil {
		panic(err)
	}
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: "http://127.0.0.1:11434",
		Model:   "modelscope.cn/Qwen/Qwen3-32B-GGUF:latest",
	})
	if err != nil {
		panic(err)
	}
	//这个a2a服务 我们还是用天气查询服务来演示
	weatherTool := tools.NewWeatherTool(&tools.WeatherConfig{
		ApiKey: tools.ApiKey,
	})
	toolsNodeConfig := compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{
			weatherTool,
		},
	}
	chatModelAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "高德天气查询智能体",
		Description: "一个可以查询天气的智能体",
		Instruction: "你是一个天气助手，请使用高德天气API查询天气信息",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: toolsNodeConfig,
		},
	})
	if err != nil {
		panic(err)
	}
	err = eino.RegisterServerHandlers(ctx, chatModelAgent, &eino.ServerConfig{
		Registrar: r,
		URL:       "http://localhost:8777",
	})
	if err != nil {
		panic(err)
	}
	err = h.Run()
	if err != nil {
		panic(err)
	}
}
