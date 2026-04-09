package agents

import (
	"app/shared"
	"common/biz"
	"context"
	"core/ai"
	"core/ai/deepagent"
	"core/ai/interview"
	"core/ai/mcps"
	"core/ai/store"
	"core/ai/tools"
	"encoding/json"
	"errors"
	"fmt"
	"model"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/a2a/client"
	"github.com/cloudwego/eino-ext/a2a/extension/eino"
	"github.com/cloudwego/eino-ext/a2a/transport/jsonrpc"
	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
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
	"gorm.io/gorm"
)

type service struct {
	repo             repository
	stateMutex       sync.RWMutex
	interviewStates  map[string]*interview.StageState //面试状态
	pendingAnswer    map[string]string                //待处理的答案
	waitingStates    map[string]bool
	checkPointStore  compose.CheckPointStore
	deepAgentFactory *deepagent.Factory
}

func (s *service) LoadAgent(ctx context.Context, agentId uuid.UUID) (*model.Agent, error) {
	return s.repo.getAgentById(ctx, agentId)
}

func (s *service) GetProviderConfig(ctx context.Context, provider string, modelName string) (*model.ProviderConfig, error) {
	return s.getProviderConfig(ctx, model.LLMTypeChat, provider, modelName)
}

func (s *service) SearchKnowledgeBase(ctx context.Context, userId uuid.UUID, query string, kbId uuid.UUID) ([]*shared.SearchKnowledgeBaseResult, error) {
	return s.searchKnowledgeBase(ctx, userId, query, kbId)
}

func (s *service) isWaitingForAnswer(sessionId string) bool {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	return s.waitingStates[sessionId]
}
func (s *service) setWaitingState(sessionId string, waiting bool) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	s.waitingStates[sessionId] = waiting
}
func (s *service) GetAndClearAnswer(sessionId string) (string, bool) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	ans, ok := s.pendingAnswer[sessionId]
	if ok {
		delete(s.pendingAnswer, sessionId)
	}
	return ans, true
}

func (s *service) GetState(sessionId string) *interview.StageState {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	if state, ok := s.interviewStates[sessionId]; ok {
		historyCopy := make([]interview.QAPair, len(state.History))
		copy(historyCopy, state.History)
		rawInputsCopy := make([]string, len(state.RawInputs))
		copy(rawInputsCopy, state.RawInputs)
		stageScoresCopy := make(map[int]float64)
		for k, v := range state.StageScores {
			stageScoresCopy[k] = v
		}
		return &interview.StageState{
			Stage:            state.Stage,
			Round:            state.Round,
			MaxRound:         state.MaxRound,
			History:          historyCopy,
			LastQuestion:     state.LastQuestion,
			Completed:        state.Completed,
			Score:            state.Score,
			StageReport:      state.StageReport,
			ResumeContext:    state.ResumeContext,
			ResumeReceived:   state.ResumeReceived,
			RawInputs:        rawInputsCopy,
			PreStagesSummary: state.PreStagesSummary,
			StageScores:      stageScoresCopy,
			AwaitingAnswer:   state.AwaitingAnswer,
		}
	}
	return nil
}

func (s *service) SaveState(sessionId string, state *interview.StageState) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	//深拷贝所有可变字段
	historyCopy := make([]interview.QAPair, len(state.History))
	copy(historyCopy, state.History)
	rawInputsCopy := make([]string, len(state.RawInputs))
	copy(rawInputsCopy, state.RawInputs)
	stageScoresCopy := make(map[int]float64)
	for k, v := range state.StageScores {
		stageScoresCopy[k] = v
	}
	s.interviewStates[sessionId] = &interview.StageState{
		Stage:            state.Stage,
		Round:            state.Round,
		MaxRound:         state.MaxRound,
		History:          historyCopy,
		LastQuestion:     state.LastQuestion,
		Completed:        state.Completed,
		Score:            state.Score,
		StageReport:      state.StageReport,
		ResumeContext:    state.ResumeContext,
		ResumeReceived:   state.ResumeReceived,
		RawInputs:        rawInputsCopy,
		PreStagesSummary: state.PreStagesSummary,
		StageScores:      stageScoresCopy,
		AwaitingAnswer:   state.AwaitingAnswer,
	}
}

func (s *service) ClearState(sessionId string) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	delete(s.interviewStates, sessionId)
}

// 确保service实现了接口
var _ interview.StateProvider = (*service)(nil)

func (s *service) createAgent(ctx context.Context, userId uuid.UUID, req CreateAgentReq) (any, error) {
	//子上下文 不能超过10s
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	agent := model.DefaultAgent(userId, req.Name, req.Description, req.Status)
	if agent.Mode == "" {
		agent.Mode = req.Mode
	}
	if req.Mode == model.DeepAgentMode && req.DeepConfig != nil {
		agent.DeepConfig = req.DeepConfig
	}
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
	if req.Mode != "" {
		agent.Mode = req.Mode
	}
	if req.DeepConfig != nil {
		agent.DeepConfig = req.DeepConfig
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
		switch agent.Mode {
		case model.DeepAgentMode:
			s.handleDeepAgent(ctx, userID, req, agent, dataChan, errChan)
		case model.GeneralAgentMode:
			if agent.Name == "AI面试" {
				s.handlerInterviewProcess(ctx, userID, req, agent, dataChan, errChan)
				return
			}
			s.handleNormalAgent(ctx, userID, req, agent, dataChan, errChan)
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

func (s *service) buildMainAgent(ctx context.Context, agent *model.Agent, history []*schema.Message, message string, dataChan chan string) (adk.Agent, error) {
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
	for _, v := range agent.Workflows {
		workflowTool := ai.NewWorkflowTool(v)
		allTools = append(allTools, workflowTool)
	}
	skills, err := s.buildSkills(agent)
	if err != nil {
		logs.Errorf("构建skills失败: %v", err)
		return nil, err
	}
	systemPrompt := ai.BaseSystemPrompt
	if agent.Name == "AI运维" || agent.Name == "OpsMaster" {
		systemPrompt = ai.DevOpsSystemPrompt
	}
	//在这里将关联的知识库内容查询出来
	ragContext := s.buildRagContext(ctx, dataChan, message, agent)
	modelAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model:       chatModel,
		Name:        agent.Name,
		Description: agent.Description,
		Instruction: systemPrompt, //这是我们定义的系统提示词
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
				"ragContext":  ragContext,
				"toolsInfo":   s.formatToolsInfo(allTools),
				"agentsInfo":  s.formatAgentsDescription(agent.Agents),
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
				Tools: allTools,
			},
		},
		Handlers: skills,
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

func (s *service) addAgentAgent(ctx context.Context, userId uuid.UUID, request AgentMarketRequest) (any, error) {
	agent, err := s.repo.getAgent(ctx, userId, request.AgentId)
	if err != nil {
		logs.Errorf("addAgentAgent 获取agent失败: %v", err)
		return nil, errs.DBError
	}
	if agent == nil {
		return nil, biz.AgentNotFound
	}
	for _, v := range request.AgentMarketIds {
		aa, err := s.repo.getAgentAgent(ctx, request.AgentId, v)
		if err != nil {
			logs.Errorf("addAgentAgent 获取agent失败: %v", err)
			return nil, errs.DBError
		}
		if aa != nil {
			continue
		}
		aa = &model.AgentAgent{
			AgentId:       request.AgentId,
			AgentMarketId: v,
		}
		err = s.repo.createAgentAgent(ctx, aa)
		if err != nil {
			logs.Errorf("addAgentAgent 创建关联关系失败: %v", err)
			return nil, errs.DBError
		}

	}
	return nil, nil
}

func (s *service) deleteAgentAgent(ctx context.Context, userID uuid.UUID, request DeleteAgentMarketRequest) (any, error) {
	err := s.repo.deleteAgentAgent(ctx, request.AgentId, request.AgentMarketId)
	if err != nil {
		logs.Errorf("deleteAgentAgent 删除关联关系失败: %v", err)
		return nil, errs.DBError
	}
	return nil, nil
}

func (s *service) formatAgentsDescription(agents []*model.AgentMarket) string {
	var builder strings.Builder
	builder.WriteString("【 可调用的智能体列表 】\n")
	for _, v := range agents {
		builder.WriteString(fmt.Sprintf("- name: %s \n", v.Name))
		builder.WriteString(fmt.Sprintf("- desc: %s \n", v.Description))
	}
	return builder.String()
}

func (s *service) addWorkflowToAgent(ctx context.Context, userID uuid.UUID, agentId uuid.UUID, reqs addWorkflowToAgentReq) (any, error) {
	agent, err := s.repo.getAgent(ctx, userID, agentId)
	if err != nil {
		logs.Errorf("addWorkflowToAgent 获取agent失败: %v", err)
		return nil, errs.DBError
	}
	if agent == nil {
		return nil, biz.AgentNotFound
	}
	agentWorkflow, err := s.repo.getAgentWorkflow(ctx, agentId, reqs.WorkflowID)
	if err != nil {
		logs.Errorf("addWorkflowToAgent 获取agent_workflow失败: %v", err)
		return nil, errs.DBError
	}
	if agentWorkflow != nil {
		return nil, nil
	}
	agentWorkflow = &model.AgentWorkflow{
		AgentID:    agentId,
		WorkflowID: reqs.WorkflowID,
		IsDefault:  reqs.IsDefault,
		Priority:   reqs.Priority,
		Status:     reqs.Status,
		CreatedAt:  time.Now(),
	}
	err = s.repo.createAgentWorkflow(ctx, agentWorkflow)
	if err != nil {
		logs.Errorf("addWorkflowToAgent 创建关联关系失败: %v", err)
		return nil, errs.DBError
	}
	return nil, nil
}

func (s *service) deleteWorkflowFromAgent(ctx context.Context, agentId uuid.UUID, workflowId uuid.UUID) error {
	err := s.repo.deleteAgentWorkflow(ctx, agentId, workflowId)
	if err != nil {
		logs.Errorf("deleteWorkflowFromAgent 删除关联关系失败: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) deleteAgent(ctx context.Context, id uuid.UUID) error {
	err := s.repo.transaction(ctx, func(tx *gorm.DB) error {
		err := s.repo.deleteAgent(ctx, id)
		if err != nil {
			return err
		}
		err = s.repo.deleteAgentTools(ctx, id)
		if err != nil {
			return err
		}
		err = s.repo.deleteAgentKnowledgeBaseByAgentId(ctx, id)
		if err != nil {
			return err
		}
		err = s.repo.deleteAgentAgentByAgentId(ctx, id)
		if err != nil {
			return err
		}
		err = s.repo.deleteAgentWorkflowByAgentId(ctx, id)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logs.Errorf("deleteAgent 删除agent失败: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) createSession(ctx context.Context, userId uuid.UUID, param createSessionRequest) (*chatSessionResponse, error) {
	session := &model.ChatSession{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		AgentID: param.AgentID,
		Title:   param.Title,
		UserID:  userId,
	}
	err := s.repo.createSession(ctx, session)
	if err != nil {
		logs.Errorf("createSession 创建session失败: %v", err)
		return nil, errs.DBError
	}
	return toChatSessionResponse(session), nil
}

func (s *service) listSessions(ctx context.Context, userID uuid.UUID, agentId uuid.UUID) ([]*model.ChatSession, error) {
	list, err := s.repo.listSessions(ctx, userID, agentId)
	if err != nil {
		logs.Errorf("listSessions 获取session列表失败: %v", err)
		return nil, errs.DBError
	}
	return list, nil
}

func (s *service) getSessionMessages(ctx context.Context, sessionId uuid.UUID) ([]*chatMessageResponse, error) {
	list, err := s.repo.getSessionMessages(ctx, sessionId)
	if err != nil {
		logs.Errorf("getSessionMessages 获取session消息列表失败: %v", err)
		return nil, errs.DBError
	}
	return toChatMessageResponses(list), nil
}

func (s *service) deleteSession(ctx context.Context, sessionId uuid.UUID) error {
	err := s.repo.transaction(ctx, func(tx *gorm.DB) error {
		err := s.repo.deleteSession(ctx, sessionId)
		if err != nil {
			return err
		}
		err = s.repo.deleteSessionMessages(ctx, sessionId)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logs.Errorf("deleteSession 删除session失败: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) saveChatMessage(sessionId uuid.UUID, message string, roleType schema.RoleType) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	chatMessage := &model.ChatMessage{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		SessionID: sessionId,
		Role:      string(roleType),
		Content:   message,
	}
	err := s.repo.saveChatMessage(ctx, chatMessage)
	if err != nil {
		logs.Errorf("saveChatMessage 保存session消息失败: %v", err)
	}
}

func (s *service) handleNormalAgent(ctx context.Context, userID uuid.UUID, req AgentMessageReq, agent *model.Agent, dataChan chan string, errChan chan error) {
	var session *model.ChatSession
	var err error
	if req.SessionId != nil {
		//使用现有会话
		session, err = s.repo.getSession(ctx, req.SessionId)
		if err != nil {
			logs.Errorf("查询会话失败: %v", err)
			s.sendError(ctx, errChan, err)
			return
		}
	} else {
		//创建新会话
		session = &model.ChatSession{
			BaseModel: model.BaseModel{
				ID: uuid.New(),
			},
			AgentID: agent.ID,
			UserID:  userID,
			Title:   req.Message,
		}
		err = s.repo.createSession(ctx, session)
		if err != nil {
			logs.Errorf("创建会话失败: %v", err)
		} else {
			//通知前端新建了会话，这样前端就会将sessionId携带
			sessionInfo, _ := json.Marshal(map[string]any{
				"action":    "session_created",
				"sessionId": session.ID,
				"title":     session.Title,
			})
			s.sendData(ctx, dataChan, string(sessionInfo))
		}
	}
	//加载历史消息
	var history []*schema.Message
	messages, err := s.repo.getSessionMessages(ctx, session.ID)
	if err != nil {
		logs.Errorf("查询会话历史消息失败: %v", err)
	} else {
		for _, v := range messages {
			switch v.Role {
			case string(schema.User):
				history = append(history, schema.UserMessage(v.Content))
			case string(schema.Assistant):
				history = append(history, schema.AssistantMessage(v.Content, nil))
			case string(schema.System):
				history = append(history, schema.SystemMessage(v.Content))
			}
		}
	}
	//存储消息
	go s.saveChatMessage(session.ID, req.Message, schema.User)
	//我们用eino框架的adk来进行agent开发，所以这里我们需要构建一个主agent
	//因为我们的智能体能添加子智能体，一起协同工作
	mainAgent, err := s.buildMainAgent(ctx, agent, history, req.Message, dataChan)
	if err != nil {
		logs.Errorf("构建主智能体失败: %v", err)
		s.sendError(ctx, errChan, err)
		return
	}
	//构建子Agent
	var subAgents []adk.Agent
	for _, v := range agent.Agents {
		t, err := jsonrpc.NewTransport(ctx, &jsonrpc.ClientConfig{
			BaseURL:     v.URL,
			HandlerPath: v.HandlerPath,
		})
		if err != nil {
			logs.Errorf("构建子智能体失败: %v", err)
			continue
		}
		aClient, err := client.NewA2AClient(ctx, &client.Config{
			Transport: t,
		})
		if err != nil {
			logs.Errorf("构建子智能体失败: %v", err)
			continue
		}
		newAgent, err := eino.NewAgent(ctx, eino.AgentConfig{
			Client: aClient,
		})
		if err != nil {
			logs.Errorf("构建子智能体失败: %v", err)
			continue
		}
		subAgents = append(subAgents, newAgent)
	}
	//构建supervisoragent
	supervisorAgent, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: mainAgent,
		SubAgents:  subAgents,
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
				go s.saveChatMessage(session.ID, msg.Content, schema.Assistant)
				s.sendData(ctx, dataChan, ai.BuildMessage(events.AgentName, msg.ToolName, msg.Content))
			}
		}
	}
}

func (s *service) handlerInterviewProcess(ctx context.Context, userID uuid.UUID, req AgentMessageReq, agent *model.Agent, dataChan chan string, errChan chan error) {
	// 1. 初始化session
	var sessionId uuid.UUID
	var isNewSession bool
	if req.SessionId == nil {
		isNewSession = true
		sessionId = uuid.New()
		session := &model.ChatSession{
			BaseModel: model.BaseModel{
				ID: sessionId,
			},
			AgentID: agent.ID,
			UserID:  userID,
			Title:   "AI面试-" + time.Now().Format("01-02"),
		}
		err := s.repo.createSession(ctx, session)
		if err != nil {
			logs.Errorf("创建会话失败: %v", err)
			s.sendError(ctx, errChan, err)
			return
		}
		sessionInfo, _ := json.Marshal(map[string]any{
			"action":    "session_created",
			"sessionId": session.ID,
			"title":     session.Title,
		})
		s.sendData(ctx, dataChan, string(sessionInfo))
	} else {
		isNewSession = false
		sessionId = *req.SessionId
	}
	//这里我们获取一下用到的chatModel
	llm, err := s.getInterviewLLM(ctx, agent)
	if err != nil {
		logs.Errorf("获取面试模型失败: %v", err)
		s.sendError(ctx, errChan, err)
		return
	}
	//2. 检查简历是否存在合法
	//这里我们需要做一个面试的状态，用于判断面试到了哪一步，因为只有在最开始才检查是否有简历输入
	state := s.GetState(sessionId.String())
	//如果没有接收简历，进行简历检查
	if state == nil || !state.ResumeReceived {
		//这个agent是用于简历识别
		checker := interview.NewInterviewStageAgent(
			"简历识别",
			interview.StageCheckResume,
			0,
			s,
			llm,
			0,
			[]string{
				"简历识别",
			},
		)
		checkResult := checker.CheckIfResume(ctx, req.Message)
		//如果检测到不是简历，发送内容让用户重新上传简历 置信度是0-1 越高代表越符合
		if !checkResult.IsResume || checkResult.Confidence < 0.7 {
			welcomeMsg := "👋 欢迎来到AI面试系统！\n\n"
			if isNewSession {
				welcomeMsg += "为了开始面试，请先发送您的简历内容。\n\n"
			} else {
				welcomeMsg += "检测到您发送的内容可能不是简历格式。\n\n"
			}
			suggestion := "请提供包含以下内容的简历：\\n1. 姓名和联系方式\\n2. 工作经历\\n3. 项目经验\\n4. 专业技能\\n5. 教育背景"
			go s.saveChatMessage(sessionId, welcomeMsg+suggestion, schema.Assistant)
			s.sendData(ctx, dataChan, ai.BuildMessage("AI面试官", "resume_required", welcomeMsg+suggestion))
			return
		}
		//保存简历状态 初始化面试流程
		s.SaveState(sessionId.String(), &interview.StageState{
			Stage:          0,
			Round:          0,
			MaxRound:       3,
			History:        []interview.QAPair{},
			ResumeContext:  req.Message,
			ResumeReceived: true,
			RawInputs:      []string{req.Message},
			StageScores:    make(map[int]float64),
			AwaitingAnswer: false,
		})
		//这里我们需要设置一个等待用户回答的状态，等待用户输入开始
		s.setWaitingState(sessionId.String(), true)
		//发送简历确认消息和面试规则说明
		confirmMsg := fmt.Sprintf("✅ 简历收到！检测到候选人：**%s**\n", checkResult.Name)
		if len(checkResult.Skills) > 0 {
			confirmMsg += fmt.Sprintf("核心技能：%s\n", strings.Join(checkResult.Skills[:min(5, len(checkResult.Skills))], "、"))
		}
		confirmMsg += "\n🎯 面试流程：共4轮（一面基础20% → 二面项目35% → 终面综合25% → HR面20%）\n"
		confirmMsg += "规则：每轮3题，单轮<60分终止，综合≥75分通过。\n\n"
		confirmMsg += "**请回复「开始」启动面试**"
		go s.saveChatMessage(sessionId, confirmMsg, schema.Assistant)
		s.sendData(ctx, dataChan, ai.BuildMessage("AI面试官", "resume_accepted", confirmMsg))
		return
	}
	//3. 简历合法用户输入开始开始面试，检查开始命令是否输入
	if s.isWaitingForAnswer(sessionId.String()) &&
		state.Stage == 0 &&
		state.Round == 0 &&
		len(state.History) == 0 {
		input := strings.TrimSpace(strings.ToLower(req.Message))
		if input != "开始" && input != "start" {
			//开始面试
			s.sendData(ctx, dataChan, ai.BuildMessage("AI面试官", "waiting_started", "等待开始\n请回复[开始]启动面试。"))
			return
		}
		s.setWaitingState(sessionId.String(), false)
		message := ai.BuildMessage("AI面试官", "interview_started", "面试正式开始！\n请认真回答面试官的问题，每轮结束后都会收到评价反馈。\n\n---")
		go s.saveChatMessage(sessionId, message, schema.Assistant)
		s.sendData(ctx, dataChan, message)
	} else if s.isWaitingForAnswer(sessionId.String()) {
		//正常面试流程
		s.savePendingAnswer(sessionId.String(), req.Message)
	}
	//4. 创建多轮面试的智能体
	stages := []interview.StageAgent{
		{
			Name:      "一面官(基础)",
			StageType: interview.StageFirst,
			Weight:    0.20,
			Dimensions: []string{
				"编程基础",
				"算法基础",
				"数据结构",
			},
		},
		{
			Name:      "二面官(项目)",
			StageType: interview.StageSecond,
			Weight:    0.35,
			Dimensions: []string{
				"架构设计",
				"技术深度",
				"项目技术",
				"项目质量",
				"项目经验",
			},
		},
		{
			Name:      "终面官(综合)",
			StageType: interview.StageFinal,
			Weight:    0.25,
			Dimensions: []string{
				"沟通表达",
				"团队协作",
				"文化匹配",
			},
		},
		{
			Name:      "HR面官(综合)",
			StageType: interview.StageHR,
			Weight:    0.20,
			Dimensions: []string{
				"稳定性",
				"职业规划",
				"价值观",
			},
		},
	}
	//5. 构建SequentialAgent
	seqAgent, err := s.buildSequentialAgent(ctx, llm, stages, sessionId.String())
	if err != nil {
		logs.Errorf("创建面试智能体失败: %v", err)
		s.sendError(ctx, errChan, err)
		return
	}
	//sessionKey注入到上下文中
	ctx = context.WithValue(ctx, interview.SessionKeyCtxKey{}, sessionId.String())
	//6. 构建Runner运行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           seqAgent,
		EnableStreaming: true,
		CheckPointStore: s.checkPointStore,
	})
	var userMsg string
	if s.isWaitingForAnswer(sessionId.String()) {
		userMsg = req.Message
	} else if state.Stage == 0 && state.Round == 0 {
		userMsg = fmt.Sprintf("候选人简历内容：\n%s\n\n请从第一阶段开始面试。", state.ResumeContext)
	} else {
		userMsg = req.Message
	}
	iter := runner.Query(ctx, userMsg, adk.WithCheckPointID(sessionId.String()))
	//7. 处理模型返回的数据
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			s.sendData(ctx, dataChan, ai.BuildErrMessage(event.AgentName, event.Err.Error()))
			return
		}
		//这个遇到需要用户回答就会产生中断，这里我们处理中断，等待用户回答
		if event.Action != nil && event.Action.Interrupted != nil {
			s.setWaitingState(sessionId.String(), true)
			round := 0
			question := ""
			if event.Output != nil && event.Output.MessageOutput != nil {
				if msg, err := event.Output.MessageOutput.GetMessage(); err == nil && msg != nil {
					question = msg.Content
				}
			}
			//从中断消息中获取元数据，这个我们到时候发送中断消息时 发送这些内容
			if event.Action.Interrupted.InterruptContexts != nil {
				contexts := event.Action.Interrupted.InterruptContexts
				for _, v := range contexts {
					if v.Info != nil {
						m := v.Info.(map[string]any)
						if q, ok := m["question"].(string); ok {
							question = q
						}
						if r, ok := m["round"].(int); ok {
							round = r
						}
					}
				}
			}
			//发送题目给前端
			msg := ai.BuildMessage(event.AgentName, "interview_wait", fmt.Sprintf("第%d题：%s\n：", round, question))
			go s.saveChatMessage(sessionId, msg, schema.Assistant)
			s.sendData(ctx, dataChan, msg)
			return
		}
		//处理阶段完成事件
		if event.Output != nil && event.Output.CustomizedOutput != nil {
			//这个数据我们在阶段完成时，发送自定义的输出
			if data, ok := event.Output.CustomizedOutput.(map[string]any); ok {
				if stageComplete, ok := data["stage_complete"].(bool); ok && stageComplete {
					score, _ := data["score"].(float64)
					passed, _ := data["passed"].(bool)
					stageName, _ := data["stage_name"].(string)
					currentState := s.GetState(sessionId.String())
					if currentState == nil {
						s.sendError(ctx, errChan, fmt.Errorf("invalid state"))
						return
					}
					//如果未通过
					if !passed {
						//终止面试
						s.terminateInterview(sessionId, stageName, score, currentState.StageScores, stages, dataChan, agent.Name)
						return
					}
					//通过发送阶段完成的消息
					completeMsg := ai.BuildMessage(
						stageName,
						"stage_complete",
						fmt.Sprintf("【%s 完成】\n 阶段评分: %.1f/100", stageName, score))
					go s.saveChatMessage(sessionId, completeMsg, schema.Assistant)
					s.sendData(ctx, dataChan, completeMsg)
					//检查是否完成所有的阶段
					if currentState.Stage >= len(stages) {
						//最终评价
						s.finalizeInterviewResult(sessionId, currentState.StageScores, stages, dataChan, agent.Name)
						return
					}
					//继续执行下一阶段
					if currentState.Stage < len(stages) {
						nextStage := stages[currentState.Stage]
						transitionMsg := ai.BuildMessage(
							agent.Name,
							"stage_transition",
							fmt.Sprintf("⏩ 进入下一阶段：%s\n考察重点: %s", nextStage.Name, strings.Join(nextStage.Dimensions, "，")))
						go s.saveChatMessage(sessionId, transitionMsg, schema.Assistant)
						s.sendData(ctx, dataChan, transitionMsg)
					}
					state = currentState
					continue
				}
			}
		}
		//普通消息
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				s.sendData(ctx, dataChan, ai.BuildErrMessage(event.AgentName, event.Err.Error()))
				return
			}
			if msg != nil && msg.Content != "" {
				out := ai.BuildMessage(event.AgentName, "", msg.Content)
				go s.saveChatMessage(sessionId, out, schema.Assistant)
				s.sendData(ctx, dataChan, out)
			}
		}
	}
	s.setWaitingState(sessionId.String(), false)
	s.ClearState(sessionId.String())
}

func (s *service) getInterviewLLM(ctx context.Context, agent *model.Agent) (aiModel.ToolCallingChatModel, error) {
	providerConfig, err := s.getProviderConfig(ctx, model.LLMTypeChat, agent.ModelProvider, agent.ModelName)
	if err != nil {
		return nil, err
	}
	if providerConfig == nil {
		return nil, errors.New("面试模型配置不存在")
	}
	return s.buildToolCallingChatModel(ctx, agent, providerConfig)
}

func (s *service) savePendingAnswer(sessionKey string, message string) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	s.pendingAnswer[sessionKey] = message
}

func (s *service) buildSequentialAgent(ctx context.Context, llm aiModel.ToolCallingChatModel, stages []interview.StageAgent, sessionKey string) (adk.Agent, error) {
	//创建四个阶段的agent
	var subAgents []adk.Agent
	state := s.GetState(sessionKey)
	for i, stage := range stages {
		//如果该阶段已经完成，则跳过
		if state != nil && state.Stage > i {
			continue
		}
		stageAgent := interview.NewInterviewStageAgent(
			stage.Name,
			stage.StageType,
			i,
			s,
			llm,
			stage.Weight,
			stage.Dimensions)
		subAgents = append(subAgents, stageAgent)
	}
	sequentialAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		SubAgents:   subAgents,
		Name:        "AI面试流程",
		Description: "按顺序执行：一面->二面->终面->HR面",
	})
	return sequentialAgent, err
}

// finalizeInterviewResult 完成面试并计算最终结果
func (s *service) finalizeInterviewResult(sessionId uuid.UUID, scores map[int]float64, stages []interview.StageAgent, dataChan chan string, name string) {
	var totalScore float64
	var details strings.Builder
	details.WriteString("面试结果详情：\n\n")
	for i, stage := range stages {
		if score, ok := scores[i]; ok {
			weighted := score * stage.Weight
			totalScore += weighted
			details.WriteString(fmt.Sprintf("阶段：%s\n权重：%.1f\n得分：%.1f\n总分：%.1f\n\n", stage.Name, stage.Weight, score, weighted))
		}
	}
	details.WriteString(fmt.Sprintf("总分：%.1f/%.1f\n", totalScore, 100.0))
	passed := totalScore >= 75.0
	var result string
	if passed {
		result = fmt.Sprintf("恭喜，面试通过！(%.1f分) \n\n%s\n\n建议：进入offer审批流程", totalScore, details.String())
	} else {
		result = fmt.Sprintf("面试未通过，请重新准备面试。(%.1f分 < 75分及格线) \n\n%s\n\n建议：加强技术深度", totalScore, details.String())
	}
	out := ai.BuildMessage(name, "interview_complete", result)
	go s.saveChatMessage(sessionId, out, schema.Assistant)
	s.sendData(context.Background(), dataChan, out)
	s.ClearState(sessionId.String())
}

func (s *service) terminateInterview(sessionId uuid.UUID, stageName string, failScore float64, scores map[int]float64, stages []interview.StageAgent, dataChan chan string, name string) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("❌ 【%s 未通过】\n 阶段评分: %.1f/100\n\n", stageName, failScore))
	if len(scores) > 1 {
		//列出已完成的阶段得分
		result.WriteString("已完成阶段得分：\n")
		for i, stage := range stages {
			if score, ok := scores[i]; ok && stage.Name != stageName {
				result.WriteString(fmt.Sprintf("阶段：%s\n得分：%.1f\n\n", stage.Name, score))
			}
		}
	}
	result.WriteString("\n建议：针对薄弱的点，加强技术深度，重新准备面试。")
	out := ai.BuildMessage(name, "interview_terminated", result.String())
	go s.saveChatMessage(sessionId, out, schema.Assistant)
	s.sendData(context.Background(), dataChan, out)
	s.ClearState(sessionId.String())
}

func (s *service) buildSkills(agent *model.Agent) ([]adk.ChatModelAgentMiddleware, error) {
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
	//if agent.Name == "git提交" {
	//	backend, err := skill.NewLocalBackend(&skill.LocalBackendConfig{
	//		BaseDir: "D:\\ai\\mszlu\\mszlu-im\\.roo\\skills",
	//	})
	//	if err != nil {
	//		logs.Errorf("创建技能后端失败：%v", err)
	//		return nil, err
	//	}
	//	list, err := backend.List(context.Background())
	//	if err != nil {
	//		logs.Errorf("获取技能列表失败：%v", err)
	//		return nil, err
	//	}
	//	var skills []adk.AgentMiddleware
	//	for _, sk := range list {
	//		middleware, err := skill.New(context.Background(), &skill.Config{
	//			Backend:       backend,
	//			SkillToolName: &sk.Name,
	//			UseChinese:    true,
	//		})
	//		if err != nil {
	//			logs.Errorf("创建技能失败：%v", err)
	//			return nil, err
	//		}
	//		skills = append(skills, middleware)
	//	}
	//	return skills, nil
	//}
	//return []adk.AgentMiddleware{}, nil
}

func (s *service) addSkillToAgent(ctx context.Context, userID uuid.UUID, agentId uuid.UUID, reqs AddAgentSkillReq) (any, error) {
	//检查agent是否存在
	agent, err := s.repo.getAgent(ctx, userID, agentId)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, biz.AgentNotFound
	}
	//直接关联技能
	err = s.repo.transaction(ctx, func(tx *gorm.DB) error {
		for _, skillID := range reqs.SkillIDs {
			//检查是否已经存在关联
			existed, err := s.repo.getAgentSkill(ctx, agentId, skillID)
			if err != nil {
				return err
			}
			if existed == nil {
				//如果不存在就创建新的关联
				agentSkill := &model.AgentSkill{
					AgentID:   agentId,
					SkillID:   skillID,
					Status:    "active",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := s.repo.saveAgentSkill(ctx, agentSkill)
				if err != nil {
					logs.Errorf("保存技能关联失败：%v", err)
					return err
				}
			} else {
				//如果已经存在，防止是未激活状态，则激活
				existed.Status = "active"
				existed.UpdatedAt = time.Now()
				err := s.repo.updateAgentSkill(ctx, existed)
				if err != nil {
					logs.Errorf("更新技能关联失败：%v", err)
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		logs.Errorf("添加技能失败：%v", err)
		return nil, err
	}

	return nil, nil
}

func (s *service) deleteSkillFromAgent(ctx context.Context, userID uuid.UUID, agentId uuid.UUID, skillId uuid.UUID) error {
	//检查agent是否存在
	agent, err := s.repo.getAgent(ctx, userID, agentId)
	if err != nil {
		return err
	}
	if agent == nil {
		return biz.AgentNotFound
	}
	err = s.repo.deleteAgentSkill(ctx, agentId, skillId)
	if err != nil {
		logs.Errorf("删除技能失败：%v", err)
		return err
	}
	return nil
}

func (s *service) deleteAgentTool(ctx context.Context, userID uuid.UUID, agentId uuid.UUID, toolId uuid.UUID) error {
	err := s.repo.deleteAgentTool(ctx, agentId, toolId)
	if err != nil {
		logs.Errorf("删除工具失败：%v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) handleDeepAgent(ctx context.Context, userID uuid.UUID, req AgentMessageReq, agent *model.Agent, dataChan chan string, errChan chan error) {
	s.handleAgentMessage(ctx, userID, req, agent, dataChan, errChan, func(ctx context.Context, session *model.ChatSession, dataChan chan string, errChan chan error) error {
		factory := s.deepAgentFactory
		deepAgent, err := factory.Create(ctx, &deepagent.UniversalDeepAgentConfig{
			Name:           agent.Name,
			Description:    agent.Description,
			SubAgentLoader: s,
			Agent:          agent,
			SystemPrompt:   agent.SystemPrompt,
		})
		if err != nil {
			logs.Errorf("创建深度代理失败：%v", err)
			return fmt.Errorf("创建深度代理失败")
		}
		eventChan, err := deepAgent.ChatStream(ctx, req.Message)
		if err != nil {
			logs.Errorf("深度代理聊天失败：%v", err)
			return fmt.Errorf("深度代理聊天失败")
		}
		subAgentName := make(map[string]string)
		for eve := range eventChan {
			if eve.Err != nil {
				logs.Errorf("深度代理聊天失败：%v", eve.Err)
				s.sendData(ctx, dataChan, ai.BuildErrMessage(agent.Name, eve.Err.Error()))
				continue
			}
			if eve.Output != nil {
				if eve.Output.MessageOutput != nil {
					msg, err := eve.Output.MessageOutput.GetMessage()
					if err == nil && msg != nil {
						if msg.Role == schema.Tool {
							toolName, ok := subAgentName[msg.ToolCallID]
							if !ok {
								toolName = msg.ToolName
							}
							responseMsg := ai.BuildMessage(agent.Name, toolName, msg.Content)
							s.sendData(ctx, dataChan, responseMsg)
							if msg.Content != "" {
								go s.saveChatMessage(session.ID, responseMsg, schema.Assistant)
							}
						} else if len(msg.ToolCalls) > 0 {
							for _, tc := range msg.ToolCalls {
								name := tc.Function.Name
								if name == "task" {
									//有子agent执行
									args := tc.Function.Arguments
									if args != "" {
										var subAgent deepagent.SubAgent
										err = json.Unmarshal([]byte(args), &subAgent)
										if err != nil {
											logs.Errorf("解析子代理参数失败：%v", err)
											continue
										}
										if subAgent.SubagentType != "" {
											tn := "子Agent- " + subAgent.SubagentType
											subAgentName[tc.ID] = tn
										}
									}
								}
							}
							s.handleDeepAgentMessage(ctx, agent, msg, dataChan)
							if msg.Content != "" {
								toolCallMsg := ai.BuildMessage(agent.Name, msg.ToolName, msg.Content)
								go s.saveChatMessage(session.ID, toolCallMsg, schema.Assistant)
							}
						} else if msg.Content != "" {
							normalMsg := ai.BuildMessage(agent.Name, msg.ToolName, msg.Content)
							s.sendData(ctx, dataChan, normalMsg)
							if msg.Content != "" {
								go s.saveChatMessage(session.ID, normalMsg, schema.Assistant)
							}
						}
					}
					if eve.Output.CustomizedOutput != nil {
						//如果有自定义的输出在这里处理
					}
				}
				//处理Action事件
				if eve.Action != nil {
					if eve.Action.TransferToAgent != nil {
						transferContent := fmt.Sprintf("正在将任务转交给%s",
							eve.Action.TransferToAgent.DestAgentName)
						transferMsg := ai.BuildMessage(agent.Name, "transfer_to_agent", transferContent)
						s.sendData(ctx, dataChan, transferMsg)
						go s.saveChatMessage(session.ID, transferMsg, schema.Assistant)
					}
					if eve.Action.Interrupted != nil {
						//中断事件
						if len(eve.Action.Interrupted.InterruptContexts) > 0 {
							for _, interruptCtx := range eve.Action.Interrupted.InterruptContexts {
								if interruptCtx.Info != nil {
									info := interruptCtx.Info.(map[string]any)
									if question, ok := info["question"].(string); ok {
										msg := ai.BuildMessage(agent.Name, "interrupt", question)
										s.sendData(ctx, dataChan, msg)
										break
									}
								}
							}
						}
						return nil
					}
					if eve.Action.Exit {
						//退出事件
						exitMsg := ai.BuildMessage(agent.Name, "exit", "任务执行完成")
						s.sendData(ctx, dataChan, exitMsg)
						go s.saveChatMessage(session.ID, exitMsg, schema.Assistant)
					}
				}
				//推理内容
				if eve.Output != nil && eve.Output.MessageOutput != nil {
					msg, err := eve.Output.MessageOutput.GetMessage()
					if err == nil && msg != nil && msg.ReasoningContent != "" {
						reasonMsg := ai.BuildMessage(agent.Name, msg.ToolName, msg.ReasoningContent)
						s.sendData(ctx, dataChan, reasonMsg)
					}
				}
			}
		}
		return nil
	})
}

type agentExecutionHandler func(ctx context.Context, session *model.ChatSession, dataChan chan string, errChan chan error) error

func (s *service) handleAgentMessage(
	ctx context.Context,
	userID uuid.UUID,
	req AgentMessageReq,
	agent *model.Agent,
	dataChan chan string,
	errChan chan error,
	executor agentExecutionHandler) {
	var session *model.ChatSession
	var err error
	if req.SessionId != nil {
		//使用现有会话
		session, err = s.repo.getSession(ctx, req.SessionId)
		if err != nil {
			logs.Errorf("查询会话失败: %v", err)
			s.sendError(ctx, errChan, err)
			return
		}
	} else {
		//创建新会话
		session = &model.ChatSession{
			BaseModel: model.BaseModel{
				ID: uuid.New(),
			},
			AgentID: agent.ID,
			UserID:  userID,
			Title:   req.Message,
		}
		err = s.repo.createSession(ctx, session)
		if err != nil {
			logs.Errorf("创建会话失败: %v", err)
		} else {
			//通知前端新建了会话，这样前端就会将sessionId携带
			sessionInfo, _ := json.Marshal(map[string]any{
				"action":    "session_created",
				"sessionId": session.ID,
				"title":     session.Title,
			})
			s.sendData(ctx, dataChan, string(sessionInfo))
		}
	}
	//加载历史消息
	var history []*schema.Message
	messages, err := s.repo.getSessionMessages(ctx, session.ID)
	if err != nil {
		logs.Errorf("查询会话历史消息失败: %v", err)
	} else {
		for _, v := range messages {
			switch v.Role {
			case string(schema.User):
				history = append(history, schema.UserMessage(v.Content))
			case string(schema.Assistant):
				history = append(history, schema.AssistantMessage(v.Content, nil))
			case string(schema.System):
				history = append(history, schema.SystemMessage(v.Content))
			}
		}
	}
	//存储消息
	go s.saveChatMessage(session.ID, req.Message, schema.User)
	err = executor(ctx, session, dataChan, errChan)
	if err != nil {
		logs.Errorf("执行任务失败: %v", err)
		s.sendError(ctx, errChan, err)
		return
	}
}

func (s *service) createNewSession(
	ctx context.Context,
	req AgentMessageReq,
	userID uuid.UUID,
	agent *model.Agent,
	dataChan chan string,
	errChan chan error) (*model.ChatSession, error) {
	session := &model.ChatSession{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		AgentID: agent.ID,
		UserID:  userID,
		Title:   req.Message,
	}
	err := s.repo.createSession(ctx, session)
	if err != nil {
		logs.Errorf("创建会话失败: %v", err)
	} else {
		//通知前端新建了会话，这样前端就会将sessionId携带
		sessionInfo, _ := json.Marshal(map[string]any{
			"action":    "session_created",
			"sessionId": session.ID,
			"title":     session.Title,
		})
		s.sendData(ctx, dataChan, string(sessionInfo))
	}
	return session, err
}

func (s *service) handleDeepAgentMessage(ctx context.Context, agent *model.Agent, msg adk.Message, dataChan chan string) {
	if msg.Content == "" {
		return
	}
	s.sendData(ctx, dataChan, ai.BuildMessage(agent.Name, msg.ToolName, msg.Content))
}
func newService() *service {
	factory := deepagent.NewFactory()
	return &service{
		repo:             newModels(database.GetPostgresDB().GormDB),
		checkPointStore:  store.NewInMemoryStore(),
		pendingAnswer:    make(map[string]string),
		waitingStates:    make(map[string]bool),
		interviewStates:  make(map[string]*interview.StageState),
		deepAgentFactory: factory,
	}
}
