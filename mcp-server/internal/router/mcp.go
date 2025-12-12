package router

import (
	"mcp-server/internal/tool"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type McpRouter struct {
}

func (u *McpRouter) Register(engine *gin.Engine) {
	//需要两个 /sse， /message
	//这里需要使用mcp-go 创建mcp服务
	mcpServer := server.NewMCPServer(
		"mszlu mcp server",
		mcp.LATEST_PROTOCOL_VERSION,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)
	weather := tool.NewWeatherTool(tool.GdApiKey)
	mcpServer.AddTool(weather.Build(), weather.Invoke)
	sseServer := server.NewSSEServer(
		mcpServer,
		server.WithBaseURL("http://localhost:7777"),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
		server.WithKeepAlive(true),
	)
	engine.GET("/sse", gin.WrapH(sseServer.SSEHandler()))
	engine.POST("/message", gin.WrapH(sseServer.MessageHandler()))
}
