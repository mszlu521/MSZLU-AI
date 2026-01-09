package agents

import (
	"app/shared"
	"common/biz"
	"context"
	"core/ai"
	"core/ai/mcps"
	"core/ai/tools"
	"encoding/json"
	"errors"
	"fmt"
	"model"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	aiModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/ollama/api"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/ai/einos"
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
	var allTools []tool.BaseTool
	//这里需要把关联的工具添加进去
	allTools = append(allTools, s.buildTools(agent)...)
	//在这里将关联的知识库内容查询出来
	ragContext := s.buildRagContext(ctx, dataChan, message, agent)
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
				"ragContext": ragContext,
				"toolsInfo":  s.formatToolsInfo(allTools),
				"agentsInfo": "",
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
				Tools: allTools,
			},
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

func (s *service) updateAgentTool(ctx context.Context, userID uuid.UUID, agentId uuid.UUID, req UpdateAgentToolReq) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	//先检查agent是否存在
	agent, err := s.repo.getAgent(ctx, userID, agentId)
	if err != nil {
		return nil, errs.DBError
	}
	if agent == nil {
		return nil, biz.AgentNotFound
	}
	if len(req.Tools) <= 0 {
		return nil, biz.ErrToolNotExisted
	}
	//先删除agent现有关联的工具
	err = s.repo.deleteAgentTools(ctx, agentId)
	if err != nil {
		return nil, errs.DBError
	}
	//创建新的关联记录
	var agentTools []*model.AgentTool
	var toolIds []uuid.UUID
	for _, v := range req.Tools {
		toolIds = append(toolIds, v.ID)
	}
	//获取到工具的ID，去工具表查询出对应的工具信息
	toolsList, err := s.getToolsByIds(toolIds)
	for _, t := range toolsList {
		agentTools = append(agentTools, &model.AgentTool{
			AgentID:   agentId,
			ToolID:    t.ID,
			Status:    model.Enabled,
			CreatedAt: time.Now(),
		})
	}
	//批量插入
	err = s.repo.createAgentTools(ctx, agentTools)
	if err != nil {
		logs.Errorf("批量插入agent_tools失败: %v", err)
		return nil, errs.DBError
	}
	return agentTools, nil
}

func (s *service) getToolsByIds(ids []uuid.UUID) ([]*model.Tool, error) {
	//这里我们一会去实现event 获取工具信息
	trigger, err := event.Trigger("getToolsByIds", &shared.GetToolsByIdsRequest{
		Ids: ids,
	})
	return trigger.([]*model.Tool), err
}

func (s *service) buildTools(agent *model.Agent) []tool.BaseTool {
	var agentTools []tool.BaseTool
	for _, v := range agent.Tools {
		//这里面工具的类型有system和mcp两种，我们这里先处理system
		switch v.ToolType {
		case model.SystemToolType:
			systemTool := s.loadSystemTool(v.Name)
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

func (s *service) loadSystemTool(name string) tool.BaseTool {
	return tools.FindTool(name)
}

func (s *service) formatToolsInfo(allTools []tool.BaseTool) string {
	var builder strings.Builder
	builder.WriteString("【可用工具列表】\n")
	for _, t := range allTools {
		info, _ := t.Info(context.Background())
		builder.WriteString(fmt.Sprintf("- name: `%s` \n", info.Name))
		builder.WriteString(fmt.Sprintf("  description: `%s` \n", info.Desc))
		//参数要转成json字符串
		marshal, _ := json.Marshal(info.ParamsOneOf)
		builder.WriteString(fmt.Sprintf("  params: `%s` \n", string(marshal)))
	}
	return builder.String()
}

func (s *service) addAgentKnowledgeBase(ctx context.Context, userId uuid.UUID, agentId uuid.UUID, addReq addAgentKnowledgeBaseReq) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	//先检查agent是否存在
	agent, err := s.repo.getAgent(ctx, userId, agentId)
	if err != nil {
		logs.Errorf("addAgentKnowledgeBase 获取agent失败: %v", err)
		return nil, errs.DBError
	}
	if agent == nil {
		return nil, biz.AgentNotFound
	}
	//先检查知识库是否存在
	kb, err := s.getKnowledgeBase(ctx, userId, addReq.KnowledgeBaseID)
	if err != nil {
		logs.Errorf("addAgentKnowledgeBase 获取知识库失败: %v", err)
		return nil, errs.DBError
	}
	if kb == nil {
		return nil, biz.ErrKnowledgeBaseNotFound
	}
	//查询关联关系是否存在
	exist, err := s.repo.isAgentKnowledgeBaseExist(ctx, agentId, addReq.KnowledgeBaseID)
	if err != nil {
		logs.Errorf("addAgentKnowledgeBase 查询关联关系是否存在失败: %v", err)
		return nil, errs.DBError
	}
	//如果存在 就不需要再次添加了
	if exist {
		return nil, nil
	}
	err = s.repo.createAgentKnowledgeBase(ctx, &model.AgentKnowledgeBase{
		AgentID:         agentId,
		KnowledgeBaseId: addReq.KnowledgeBaseID,
		Status:          model.AgentKnowledgeStatusEnabled,
	})
	if err != nil {
		logs.Errorf("addAgentKnowledgeBase 创建关联关系失败: %v", err)
		return nil, errs.DBError
	}
	return nil, nil
}

func (s *service) getKnowledgeBase(ctx context.Context, userId uuid.UUID, kbId uuid.UUID) (*model.KnowledgeBase, error) {
	trigger, err := event.Trigger("getKnowledgeBase", &shared.GetKnowledgeBaseRequest{
		UserId:          userId,
		KnowledgeBaseId: kbId,
	})
	return trigger.(*model.KnowledgeBase), err
}

func (s *service) deleteAgentKnowledgeBase(ctx context.Context, userID uuid.UUID, agentId uuid.UUID, kbId uuid.UUID) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	err := s.repo.deleteAgentKnowledgeBase(ctx, agentId, kbId)
	if err != nil {
		logs.Errorf("deleteAgentKnowledgeBase 删除关联关系失败: %v", err)
		return nil, errs.DBError
	}
	return nil, nil
}

func (s *service) buildRagContext(ctx context.Context, dataChan chan string, message string, agent *model.Agent) string {
	var ragContext string
	if len(agent.KnowledgeBases) > 0 {
		//从关联的知识库中进行查询
		var allResult []*shared.SearchKnowledgeBaseResult
		for _, v := range agent.KnowledgeBases {
			results, err := s.searchKnowledgeBase(ctx, agent.CreatorID, message, v.ID)
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
				if i >= 3 {
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

func (s *service) searchKnowledgeBase(ctx context.Context, userId uuid.UUID, message string, id uuid.UUID) ([]*shared.SearchKnowledgeBaseResult, error) {
	trigger, err := event.Trigger("searchKnowledgeBase", &shared.SearchKnowledgeBaseRequest{
		UserId:          userId,
		KnowledgeBaseId: id,
		Query:           message,
	})
	if err != nil {
		logs.Errorf("searchKnowledgeBase 搜索知识库失败: %v", err)
		return nil, err
	}
	response := trigger.(*shared.SearchKnowledgeBaseResponse)
	return response.Results, nil
}

func newService() *service {
	return &service{
		repo: newModels(database.GetPostgresDB().GormDB),
	}
}
