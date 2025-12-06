package agents

import (
	"app/shared"
	"common/biz"
	"context"
	"core/ai"
	"errors"
	"model"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	aiModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/ollama/api"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/errs"
	"github.com/mszlu521/thunder/event"
	"github.com/mszlu521/thunder/logs"
)

type service struct {
	repo repository
}

func (s *service) createAgent(ctx context.Context, userId uuid.UUID, req CreateAgentReq) (any, error) {
	//子上下文 不能超过10s
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	agent := model.DefaultAgent(userId, req.Name, req.Description, req.Status)
	err := s.repo.createAgent(ctx, agent)
	if err != nil {
		logs.Errorf("创建智能代理失败: %v", err)
		return nil, errs.DBError
	}
	return agent, nil
}

func (s *service) listAgents(ctx context.Context, userID uuid.UUID, req SearchAgentReq) (*ListAgentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	filter := AgentFilter{
		Name:   req.Params.Name,
		Status: req.Params.Status,
		Limit:  req.Params.PageSize,
		Offset: (req.Params.Page - 1) * req.Params.PageSize,
	}
	list, total, err := s.repo.listAgents(ctx, userID, filter)
	if err != nil {
		logs.Errorf("查询智能代理列表失败: %v", err)
		return nil, errs.DBError
	}
	return &ListAgentResponse{
		Agents: list,
		Total:  total,
	}, nil
}

func (s *service) getAgent(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Agent, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	agent, err := s.repo.getAgent(ctx, userID, id)
	if err != nil {
		logs.Errorf("查询智能代理失败: %v", err)
		return nil, errs.DBError
	}
	if agent == nil {
		return nil, biz.AgentNotFound
	}
	return agent, nil
}

func (s *service) updateAgent(ctx context.Context, userId uuid.UUID, req UpdateAgentReq) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	//先查询id是否存在
	agent, err := s.repo.getAgent(ctx, userId, req.ID)
	if err != nil {
		logs.Errorf("查询智能代理失败: %v", err)
		return nil, errs.DBError
	}
	if agent == nil {
		return nil, biz.AgentNotFound
	}
	//对更新的字段进行判断
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.Status != "" {
		agent.Status = req.Status
	}
	if req.SystemPrompt != "" {
		agent.SystemPrompt = req.SystemPrompt
	}
	if req.ModelProvider != "" {
		agent.ModelProvider = req.ModelProvider
	}
	if req.ModelName != "" {
		agent.ModelName = req.ModelName
	}
	if req.ModelParameters != nil {
		agent.ModelParameters = req.ModelParameters
	}
	if req.OpeningDialogue != "" {
		agent.OpeningDialogue = req.OpeningDialogue
	}
	err = s.repo.updateAgent(ctx, agent)
	if err != nil {
		logs.Errorf("更新智能代理失败: %v", err)
		return nil, errs.DBError
	}
	return agent, nil
}

func (s *service) agentMessage(ctx context.Context, userID uuid.UUID, req AgentMessageReq) (<-chan string, <-chan error) {
	dataChan := make(chan string)
	errChan := make(chan error)
	go func() {
		//defer中 关闭channel和处理错误
		defer func() {
			if err := recover(); err != nil {
				logs.Errorf("处理智能代理消息失败: %v", err)
				select {
				case errChan <- errors.New("internal server error"):
				case <-ctx.Done():
					logs.Warnf("发送取消 context Done")
				}
			}
			close(dataChan)
			close(errChan)
		}()
		//先获取agent
		agent, err := s.repo.getAgent(ctx, userID, req.AgentID)
		if err != nil {
			logs.Errorf("查询智能代理失败: %v", err)
			//告诉客户端,这里我们封装一下消息
			s.sendError(ctx, errChan, err)
			return
		}
		//我们用eino框架的adk来进行agent开发，所以这里我们需要构建一个主agent
		//因为我们的智能体能添加子智能体，一起协同工作
		mainAgent, err := s.buildMainAgent(ctx, agent, req.Message, dataChan)
		if err != nil {
			logs.Errorf("构建主智能体失败: %v", err)
			s.sendError(ctx, errChan, err)
			return
		}
		//构建supervisoragent
		supervisorAgent, err := supervisor.New(ctx, &supervisor.Config{
			Supervisor: mainAgent,
			SubAgents:  []adk.Agent{
				//这里可以添加多个子智能体
			},
		})
		if err != nil {
			logs.Errorf("构建supervisorAgent失败: %v", err)
			s.sendError(ctx, errChan, err)
			return
		}
		//构建Runner
		runner := adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           supervisorAgent,
			EnableStreaming: true,
		})
		iter := runner.Query(ctx, req.Message)
		for {
			//处理大模型返回的数据
			events, ok := iter.Next()
			if !ok {
				break
			}
			//检查context是否已经取消
			select {
			case <-ctx.Done():
				logs.Warnf("客户端取消了请求")
				return
			default:
			}
			//判断有没有错误
			if events.Err != nil {
				//这里我们已经能拿到agent的信息了，所以这里我们封装成json返给客户端
				//这是属于某个agent执行的错误
				//证明模型返回了错误，将错误返回给客户端
				s.sendData(ctx, dataChan, ai.BuildErrMessage(events.AgentName, events.Err.Error()))
				return
			}
			//判断有没有内容生成
			if events.Output != nil && events.Output.MessageOutput != nil {
				msg, err := events.Output.MessageOutput.GetMessage()
				if err != nil {
					logs.Errorf("获取模型返回内容失败: %v", err)
					s.sendError(ctx, errChan, err)
					return
				}
				if msg.Content == "" && msg.ReasoningContent == "" {
					continue
				}
				if msg.ReasoningContent != "" {
					//思考内容
					s.sendData(ctx, dataChan, ai.BuildReasoningMessage(events.AgentName, msg.ToolName, msg.ReasoningContent))
				}
				logs.Infof("Agent名称[%s], 工具名称:[%s], 模型返回内容: %s", events.AgentName, msg.ToolName, msg.Content)
				if msg.Content != "" {
					s.sendData(ctx, dataChan, ai.BuildMessage(events.AgentName, msg.ToolName, msg.Content))
				}
			}
		}
	}()
	return dataChan, errChan
}

func (s *service) sendError(ctx context.Context, errChan chan error, err error) {
	select {
	case errChan <- err:
	case <-ctx.Done():
		logs.Warnf("发送取消 context Done")
	}
}

func (s *service) buildMainAgent(ctx context.Context, agent *model.Agent, message string, dataChan chan string) (adk.Agent, error) {
	//构建主智能体
	//首先需要获取到agent的模型配置信息
	providerConfig, err := s.getProviderConfig(ctx, model.LLMTypeChat, agent.ModelProvider, agent.ModelName)
	if err != nil {
		return nil, errs.DBError
	}
	if providerConfig == nil {
		return nil, biz.ErrProviderConfigNotFound
	}
	//构建chatmodel，因为这里有很多厂商，所以这里要适配
	chatModel, err := s.buildToolCallingChatModel(ctx, agent, providerConfig)
	if err != nil {
		logs.Errorf("构建chatmodel失败: %v", err)
		return nil, err
	}
	modelAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model:       chatModel,
		Name:        agent.Name,
		Description: agent.Description,
		Instruction: ai.BaseSystemPrompt, //这是我们定义的系统提示词
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			//这是在最终发送大模型前做一些处理 一般是重新构建系统提示词
			template := prompt.FromMessages(schema.FString, schema.SystemMessage(ai.BaseSystemPrompt))
			messages, err2 := template.Format(ctx, map[string]any{
				"role":       agent.SystemPrompt,
				"ragContext": "",
				"toolsInfo":  "",
				"agentsInfo": "",
			})
			if err2 != nil {
				logs.Errorf("格式化模板失败: %v", err2)
				return nil, err2
			}
			messages = append(messages, input.Messages...)
			return messages, nil
		},
	})
	if err != nil {
		logs.Errorf("构建ChatModelAgent失败: %v", err)
		return nil, err
	}
	return modelAgent, nil
}

func (s *service) getProviderConfig(ctx context.Context, chat model.LLMType, provider string, name string) (*model.ProviderConfig, error) {
	//这个需要调用llms服务 所以我们需要定义event事件
	trigger, err := event.Trigger("getProviderConfig", &shared.GetProviderConfigsRequest{
		Provider:  provider,
		ModelName: name,
		LLMType:   chat,
	})
	if err != nil {
		logs.Errorf("触发getProviderConfig事件失败: %v", err)
		return nil, errs.DBError
	}
	return trigger.(*model.ProviderConfig), nil
}

func (s *service) buildToolCallingChatModel(ctx context.Context, agent *model.Agent, config *model.ProviderConfig) (aiModel.ToolCallingChatModel, error) {
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

func (s *service) sendData(ctx context.Context, dataChan chan string, data string) {
	select {
	case dataChan <- data:
	case <-ctx.Done():
		logs.Warnf("sendData 发送取消 context Done")
	}
}

func newService() *service {
	return &service{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
