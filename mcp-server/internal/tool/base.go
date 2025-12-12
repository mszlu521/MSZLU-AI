package tool

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

type MCPTool interface {
	Build() mcp.Tool
	Invoke(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}
