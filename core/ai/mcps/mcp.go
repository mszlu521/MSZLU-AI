package mcps

import (
	"context"
	"fmt"
	"strings"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mszlu521/thunder/ai/einos"
)

func GetEinoBaseTools(ctx context.Context, config *einos.McpConfig) ([]tool.BaseTool, error) {
	headers := make(map[string]string)
	if config.Token != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
	}
	options := transport.WithHeaders(headers)
	url := config.BaseUrl
	//支持streamable http
	var cli *client.Client
	var err error
	if strings.HasSuffix(url, "/sse") {
		cli, err = client.NewSSEMCPClient(url, options)
		if err != nil {
			return nil, err
		}
	} else {
		cli, err = client.NewStreamableHttpClient(url, transport.WithHTTPHeaders(headers))
		if err != nil {
			return nil, err
		}
	}
	err = cli.Start(ctx)
	if err != nil {
		return nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    config.Name,
		Version: config.Version,
	}

	_, err = cli.Initialize(ctx, initRequest)
	if err != nil {
		return nil, err
	}
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: cli})
	if err != nil {
		return nil, err
	}

	return tools, nil
}

func GetMCPTool(ctx context.Context, config *einos.McpConfig) ([]mcp.Tool, error) {
	headers := make(map[string]string)
	if config.Token != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
	}
	options := transport.WithHeaders(headers)
	url := config.BaseUrl
	//支持streamable http
	var cli *client.Client
	var err error
	if strings.HasSuffix(url, "/sse") {
		cli, err = client.NewSSEMCPClient(url, options)
		if err != nil {
			return nil, err
		}
	} else {
		cli, err = client.NewStreamableHttpClient(url, transport.WithHTTPHeaders(headers))
		if err != nil {
			return nil, err
		}
	}
	err = cli.Start(ctx)
	if err != nil {
		return nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    config.Name,
		Version: config.Version,
	}

	_, err = cli.Initialize(ctx, initRequest)
	if err != nil {
		return nil, err
	}
	tools, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}

	return tools.Tools, nil
}
