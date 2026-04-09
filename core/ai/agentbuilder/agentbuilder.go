package agentbuilder

import (
	"app/shared"
	"context"
	"core/ai"
	"core/ai/mcps"
	"core/ai/tools"
	"fmt"
	"model"
	"strings"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	aiModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/ollama/api"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/ai/einos"
	"github.com/mszlu521/thunder/logs"
)

// Context 上下文 包含构建Agent所需的信息
type Context struct {
	ProviderConfig *model.ProviderConfig
	ChatModel      aiModel.ToolCallingChatModel
	Tools          []tool.BaseTool
	Skills         []adk.ChatModelAgentMiddleware
	RAGContext     string
}

// Builder 提供构建Agent相关组件的方法
type Builder struct {
	Loader Loader
}

// Loader 加载器 定义从外部加载资源的接口
type Loader interface {
	GetProviderConfig(ctx context.Context, provider string, modelName string) (*model.ProviderConfig, error)
	SearchKnowledgeBase(ctx context.Context, userId uuid.UUID, query string, kbId uuid.UUID) ([]*shared.SearchKnowledgeBaseResult, error)
}

func NewBuilder() *Builder {
	return &Builder{}
}
func NewBuilderWithLoader(loader Loader) *Builder {
	return &Builder{
		Loader: loader,
	}
}

func (b *Builder) BuildDeepAgentContext(
	ctx context.Context,
	agent *model.Agent,
) (*Context, error) {
	if b == nil {
		return nil, fmt.Errorf("builder is nil")
	}
	if agent == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	//获取模型配置
	providerConfig, err := b.GetProviderConfig(ctx, agent.ModelProvider, agent.ModelName)
	if err != nil {
		return nil, fmt.Errorf("get provider config error: %v", err)
	}
	if providerConfig == nil {
		return nil, fmt.Errorf("provider config is nil")
	}
	//构建chatModel
	chatModel, err := b.BuildToolCallingChatModel(ctx, agent, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("build chat model error: %v", err)
	}
	//构建工具
	allTools := b.BuildTools(agent)
	for _, v := range agent.Workflows {
		workflowTool := ai.NewWorkflowTool(v)
		allTools = append(allTools, workflowTool)
	}
	skills, err := b.BuildSkills(agent)
	if err != nil {
		logs.Errorf("构建skills失败: %v", err)
		return nil, err
	}
	return &Context{
		ProviderConfig: providerConfig,
		ChatModel:      chatModel,
		Tools:          allTools,
		Skills:         skills,
	}, nil
}

// BuildContext 构建上下文
func (b *Builder) BuildContext(
	ctx context.Context,
	agent *model.Agent,
	message string,
	dataChan chan string,
) (*Context, error) {
	if b == nil {
		return nil, fmt.Errorf("builder is nil")
	}
	if agent == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	//获取模型配置
	providerConfig, err := b.GetProviderConfig(ctx, agent.ModelProvider, agent.ModelName)
	if err != nil {
		return nil, fmt.Errorf("get provider config error: %v", err)
	}
	if providerConfig == nil {
		return nil, fmt.Errorf("provider config is nil")
	}
	//构建chatModel
	chatModel, err := b.BuildToolCallingChatModel(ctx, agent, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("build chat model error: %v", err)
	}
	//构建工具
	allTools := b.BuildTools(agent)
	for _, v := range agent.Workflows {
		workflowTool := ai.NewWorkflowTool(v)
		allTools = append(allTools, workflowTool)
	}
	skills, err := b.BuildSkills(agent)
	if err != nil {
		logs.Errorf("构建skills失败: %v", err)
		return nil, err
	}
	//构建ragContext
	ragContext := b.BuildRagContext(ctx, dataChan, message, agent)
	return &Context{
		ProviderConfig: providerConfig,
		ChatModel:      chatModel,
		Tools:          allTools,
		Skills:         skills,
		RAGContext:     ragContext,
	}, nil
}

func (b *Builder) GetProviderConfig(ctx context.Context, provider string, modelName string) (*model.ProviderConfig, error) {
	if b.Loader != nil {
		return b.Loader.GetProviderConfig(ctx, provider, modelName)
	}
	return nil, fmt.Errorf("loader is nil")
}

func (b *Builder) BuildToolCallingChatModel(ctx context.Context, agent *model.Agent, config *model.ProviderConfig) (aiModel.ToolCallingChatModel, error) {
	var chatModel aiModel.ToolCallingChatModel
	var err error
	modelParams := agent.ModelParameters.ToModelParams()
	temperature := float32(modelParams.Temperature)
	topP := float32(modelParams.TopP)
	maxTokens := modelParams.MaxTokens
	if config.Provider == model.OllamaProvider {
		chatModel, err = ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			Model:   agent.ModelName,
			BaseURL: config.APIBase,
			Options: &api.Options{
				Temperature: temperature,
				TopP:        topP,
				Runner: api.Runner{
					NumCtx: maxTokens,
				},
			},
		})
	} else if config.Provider == model.OpenAIProvider {
		chatModel, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:               agent.ModelName,
			BaseURL:             config.APIBase,
			APIKey:              config.APIKey,
			MaxCompletionTokens: &maxTokens,
			Temperature:         &temperature,
			TopP:                &topP,
		})
	} else if config.Provider == model.QwenProvider {
		chatModel, err = qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
			Model:       agent.ModelName,
			BaseURL:     config.APIBase,
			APIKey:      config.APIKey,
			MaxTokens:   &maxTokens,
			Temperature: &temperature,
			TopP:        &topP,
		})
	} else {
		//默认用openai，大部分厂商都支持openai的方式
		chatModel, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:               agent.ModelName,
			BaseURL:             config.APIBase,
			APIKey:              config.APIKey,
			MaxCompletionTokens: &maxTokens,
			Temperature:         &temperature,
			TopP:                &topP,
		})
	}

	return chatModel, err
}

func (b *Builder) BuildTools(agent *model.Agent) []tool.BaseTool {
	var agentTools []tool.BaseTool
	for _, v := range agent.Tools {
		//这里面工具的类型有system和mcp两种，我们这里先处理system
		switch v.ToolType {
		case model.SystemToolType:
			systemTool := b.loadSystemTool(v.Name)
			if systemTool == nil {
				logs.Warnf("加载系统工具时，找不到工具: %v", v.Name)
				continue
			}
			agentTools = append(agentTools, systemTool)
		case model.McpToolType:
			//获取到mcp的所有tools，并且需要转换为eino的tool
			mcpConfig := einos.McpConfig{
				BaseUrl: v.McpConfig.Url,
				Token:   v.McpConfig.CredentialType,
				Name:    "mszlu-AI",
				Version: "1.0.0",
			}
			baseTools, err := mcps.GetEinoBaseTools(context.Background(), &mcpConfig)
			if err != nil {
				logs.Errorf("获取mcp tools失败: %v", err)
				continue
			}
			agentTools = append(agentTools, baseTools...)
		default:
			logs.Warnf("未知的工具类型: %v", v.ToolType)

		}
	}
	return agentTools
}

func (b *Builder) loadSystemTool(name string) tool.BaseTool {
	return tools.FindTool(name)
}

func (b *Builder) BuildSkills(agent *model.Agent) ([]adk.ChatModelAgentMiddleware, error) {
	skills := agent.Skills
	if len(skills) == 0 {
		return []adk.ChatModelAgentMiddleware{}, nil
	}
	var middlewares []adk.ChatModelAgentMiddleware
	//加载技能
	//我们按baseDir分组，避免重复创建
	dirToSkills := make(map[string][]*model.Skill)
	for _, sk := range skills {
		if sk.BaseDir != "" {
			dirToSkills[sk.BaseDir] = append(dirToSkills[sk.BaseDir], sk)
		}
	}
	//为每个baseDir创建一个backend并加载skill
	for baseDir, sls := range dirToSkills {
		backend, _ := local.NewBackend(context.Background(), &local.Config{})
		bc, err := skill.NewBackendFromFilesystem(context.Background(), &skill.BackendFromFilesystemConfig{
			Backend: backend,
			BaseDir: baseDir,
		})
		if err != nil {
			logs.Errorf("创建技能后端失败：%v", err)
			continue
		}
		for _, sk := range sls {
			middleware, err := skill.NewMiddleware(context.Background(), &skill.Config{
				Backend:       bc,
				SkillToolName: &sk.Name,
			})
			if err != nil {
				logs.Errorf("创建技能失败：%v", err)
				continue
			}
			middlewares = append(middlewares, middleware)
		}
	}
	return middlewares, nil
}

func (b *Builder) BuildRagContext(ctx context.Context, dataChan chan string, message string, agent *model.Agent) string {
	var ragContext string
	if len(agent.KnowledgeBases) > 0 {
		//从关联的知识库中进行查询
		var allResult []*shared.SearchKnowledgeBaseResult
		for _, v := range agent.KnowledgeBases {
			results, err := b.Loader.SearchKnowledgeBase(ctx, agent.CreatorID, message, v.ID)
			if err != nil {
				logs.Errorf("searchKnowledgeBase 搜索知识库失败: %v", err)
				continue
			}
			allResult = append(allResult, results...)
		}
		if len(allResult) > 0 {
			var contextBuilder strings.Builder
			contextBuilder.WriteString("【 参考以下知识库内容回答问题 】\n")
			for i, v := range allResult {
				//为了防止内容过长，这里只取前几位的结果
				//这个数字根据实际进行调整
				if i >= 1 {
					break
				}
				contextBuilder.WriteString(fmt.Sprintf("%d.  %s \n", i+1, v.Content))
			}
			ragContext = contextBuilder.String()
			//知识库查询出来的内容，我们发送到前端进行展示
			//toolName使用知识库的名称
			var names strings.Builder
			for _, v := range agent.KnowledgeBases {
				names.WriteString(v.Name + "\t")
			}
			buildMessage := ai.BuildMessage(agent.Name, names.String(), ragContext)
			dataChan <- buildMessage
		}
	}
	return ragContext
}

func (b *Builder) selectSystemPrompt(agent *model.Agent) string {
	systemPrompt := ai.BaseSystemPrompt
	if agent.Name == "AI运维" || agent.Name == "OpsMaster" {
		systemPrompt = ai.DevOpsSystemPrompt
	}
	return systemPrompt
}

func (b *Builder) CreateChatModelAgentConfig(
	ctx context.Context,
	agent *model.Agent,
	history []*schema.Message,
	agentCtx *Context) (*adk.ChatModelAgentConfig, error) {
	systemPrompt := b.selectSystemPrompt(agent)
	return &adk.ChatModelAgentConfig{
		Model:       agentCtx.ChatModel,
		Name:        agent.Name,
		Description: agent.Description,
		Instruction: systemPrompt,
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			optional := false
			if len(history) == 0 {
				optional = true
			}
			//这是在最终发送大模型前做一些处理 一般是重新构建系统提示词
			template := prompt.FromMessages(schema.FString,
				schema.SystemMessage(systemPrompt),
				schema.MessagesPlaceholder("history_key", optional),
			)
			messages, err2 := template.Format(ctx, map[string]any{
				"role":        agent.SystemPrompt,
				"ragContext":  agentCtx.RAGContext,
				"history_key": history,
			})
			if err2 != nil {
				logs.Errorf("格式化模板失败: %v", err2)
				return nil, err2
			}
			messages = append(messages, input.Messages...)
			return messages, nil
		},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentCtx.Tools,
			},
		},
		Handlers: agentCtx.Skills,
	}, nil
}
