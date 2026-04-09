package deepagent

import (
	"app/shared"
	"context"
	"core/ai/agentbuilder"
	"fmt"
	"model"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	aiModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/logs"
)

type SubAgentLoader interface {
	LoadAgent(ctx context.Context, agentId uuid.UUID) (*model.Agent, error)
	GetProviderConfig(ctx context.Context, provider string, modelName string) (*model.ProviderConfig, error)
	SearchKnowledgeBase(ctx context.Context, userId uuid.UUID, query string, kbId uuid.UUID) ([]*shared.SearchKnowledgeBaseResult, error)
}
type UniversalDeepAgent struct {
	agent        adk.ResumableAgent
	config       *model.DeepAgentConfig
	chatModel    aiModel.ToolCallingChatModel
	subAgents    []adk.Agent
	systemPrompt string
	loader       SubAgentLoader
	builder      *agentbuilder.Builder
}

func (a *UniversalDeepAgent) createSubAgents(ctx context.Context, cfg *UniversalDeepAgentConfig) ([]adk.Agent, error) {
	var subAgents []adk.Agent
	if len(a.config.SubAgentIDs) > 0 {
		return a.loadSubAgentsFromDB(ctx, cfg)
	}
	return subAgents, nil
}

func (a *UniversalDeepAgent) loadSubAgentsFromDB(ctx context.Context, cfg *UniversalDeepAgentConfig) ([]adk.Agent, error) {
	var subAgents []adk.Agent
	if a.loader == nil {
		return nil, fmt.Errorf("loader is nil")
	}
	for _, agentId := range a.config.SubAgentIDs {
		agent, err := cfg.SubAgentLoader.LoadAgent(ctx, agentId)
		if err != nil {
			logs.Errorf("loadSubAgentsFromDB 加载子代理失败: %v", err)
			return nil, err
		}
		if agent == nil {
			return nil, fmt.Errorf("agent %s not found", agentId)
		}
		agentContext, err := a.builder.BuildDeepAgentContext(ctx, agent)
		if err != nil {
			logs.Errorf("loadSubAgentsFromDB 构建skills失败: %v", err)
			return nil, err
		}
		chatModelAgentConfig, err := a.builder.CreateChatModelAgentConfig(ctx, agent, nil, agentContext)
		if err != nil {
			logs.Errorf("loadSubAgentsFromDB 创建ChatModelAgentConfig失败: %v", err)
			return nil, err
		}
		modelAgent, err := adk.NewChatModelAgent(ctx, chatModelAgentConfig)
		if err != nil {
			logs.Errorf("loadSubAgentsFromDB 创建ChatModelAgent失败: %v", err)
			return nil, err
		}
		subAgents = append(subAgents, modelAgent)
	}
	return subAgents, nil
}

type UniversalDeepAgentConfig struct {
	Name           string
	Description    string
	APIKey         string
	BaseURL        string
	SubAgentLoader SubAgentLoader
	Agent          *model.Agent
	SystemPrompt   string
}

func NewUniversalDeepAgent(
	ctx context.Context,
	cfg *UniversalDeepAgentConfig,
	deepConfig *model.DeepAgentConfig) (*UniversalDeepAgent, error) {
	builder := agentbuilder.NewBuilderWithLoader(cfg.SubAgentLoader)
	agentContext, err := builder.BuildDeepAgentContext(ctx, cfg.Agent)
	if err != nil {
		logs.Errorf("NewUniversalDeepAgent 构建上下文失败: %v", err)
		return nil, err
	}
	agent := &UniversalDeepAgent{
		config:    deepConfig,
		chatModel: agentContext.ChatModel,

		loader:       cfg.SubAgentLoader,
		builder:      builder,
		systemPrompt: cfg.SystemPrompt,
	}
	subAgents, err := agent.createSubAgents(ctx, cfg)
	if err != nil {
		return nil, err
	}
	agent.subAgents = subAgents
	backend, err := local.NewBackend(context.Background(), &local.Config{})
	if err != nil {
		logs.Errorf("NewUniversalDeepAgent 创建backend失败: %v", err)
		return nil, err
	}
	deepCfg := &deep.Config{
		Name:        cfg.Agent.Name,
		Description: cfg.Agent.Description,
		ChatModel:   agent.chatModel,
		Instruction: cfg.Agent.SystemPrompt,
		SubAgents:   subAgents,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentContext.Tools,
			},
		},
		Handlers:          agentContext.Skills,
		MaxIteration:      deepConfig.MaxIterations,
		WithoutWriteTodos: deepConfig.EnableTodos,
		Backend:           backend,
	}
	resumableAgent, err := deep.New(ctx, deepCfg)
	if err != nil {
		logs.Errorf("NewUniversalDeepAgent 创建resumableAgent失败: %v", err)
		return nil, err
	}
	agent.agent = resumableAgent
	return agent, nil
}

func (a *UniversalDeepAgent) ChatStream(ctx context.Context, message string) (<-chan *adk.AgentEvent, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           a.agent,
		EnableStreaming: true,
	})
	iter := runner.Query(ctx, message)
	eventChan := make(chan *adk.AgentEvent)
	go func() {
		defer close(eventChan)
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				select {
				case <-ctx.Done():
					return
				case eventChan <- event:
				}
				return
			}
			select {
			case <-ctx.Done():
				return
			case eventChan <- event:
			}
		}
	}()
	return eventChan, nil
}
