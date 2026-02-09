package interview

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/mszlu521/thunder/logs"
)

// StageAgent 定义面试阶段的元数据结构
type StageAgent struct {
	Name       string    `json:"name"`
	StageType  StageType `json:"stageType"`
	Weight     float64   `json:"weight"`
	Dimensions []string  `json:"dimensions"`
}

type SessionKeyCtxKey struct {
}
type StageType int

const (
	StageCheckResume StageType = iota //简历检查阶段
	StageFirst                        //技术一面
	StageSecond                       //技术二面
	StageFinal                        //终面
	StageHR                           //HR面
)

// StageState 面试会话的完整状态
type StageState struct {
	Stage            int             `json:"stage"` //当前阶段索引
	Round            int             `json:"round"` //当前阶段轮次
	MaxRound         int             `json:"maxRound"`
	History          []QAPair        `json:"history"`
	LastQuestion     string          `json:"lastQuestion"`
	Completed        bool            `json:"completed"`        //当前阶段是否完成
	Score            float64         `json:"score"`            //当前阶段得分
	StageReport      string          `json:"stageReport"`      //当前阶段报告
	ResumeReceived   bool            `json:"resumeReceived"`   //是否已经接收到简历
	ResumeContext    string          `json:"resumeContext"`    //简历内容
	RawInputs        []string        `json:"rawInputs"`        //原始输入记录
	PreStagesSummary []StageSummary  `json:"preStagesSummary"` //之前阶段总结
	StageScores      map[int]float64 `json:"stageScores"`      //各阶段得分
	AwaitingAnswer   bool            `json:"awaitingAnswer"`   //是否正在等待答案
}

type StageSummary struct {
	StageName  string            `json:"stageName"`
	StageType  StageType         `json:"stageType"`
	AvgScore   float64           `json:"avgScore"`
	Strengths  []string          `json:"strengths"`  //候选人优势维度
	Weaknesses []string          `json:"weaknesses"` //候选人劣势维度
	RedFlags   []string          `json:"redFlags"`   //候选人风险点
	Keywords   map[string]string `json:"keywords"`
}
type QAPair struct {
	Question   string             `json:"question"`   // 面试官问题
	Answer     string             `json:"answer"`     // 候选人回答
	Evaluation string             `json:"evaluation"` // 面试官评价
	Scores     map[string]float64 `json:"scores"`     // 面试官打分 各个维度的评分
	Timestamp  int64              `json:"timestamp"`  //回答时间戳
}
type ResumeCheckResult struct {
	IsResume   bool     `json:"isResume"`   //是否是简历
	Confidence float64  `json:"confidence"` //置信度
	Skills     []string `json:"skills"`     //提取的技能列表
	Experience []string `json:"experience"` //工作经历
	Name       string   `json:"name"`       //名字
	Suggestion string   `json:"suggestion"` // 建议
}

type EvalResult struct {
	TotalScore float64            `json:"totalScore"` //总分
	Dimensions map[string]float64 `json:"dimensions"` //各个维度的得分
	Feedback   string             `json:"feedback"`   //评价反馈
	RedFlags   []string           `json:"redFlags"`   //风险点
}
type StateProvider interface {
	//GetAndClearAnswer 获取指定会话的答案并清空答案 (用于中断恢复机制)
	GetAndClearAnswer(sessionId string) (string, bool)
	//GetState 获取指定会话状态
	GetState(sessionId string) *StageState
	//SaveState 保存指定会话状态
	SaveState(sessionId string, state *StageState)
	//ClearState 清空所有会话状态(面试结束调用)
	ClearState(sessionId string)
}

// InterviewStageAgent 需要创建一个用于面试的Agent
// 每个实例负责一个面试阶段
type InterviewStageAgent struct {
	name       string // 面试阶段名称
	stageType  StageType
	stageIndex int
	provider   StateProvider
	llm        model.ToolCallingChatModel
	weight     float64  //阶段权重
	dimensions []string //考察维度
}

func (a *InterviewStageAgent) Name(ctx context.Context) string {
	return a.name
}

func (a *InterviewStageAgent) Description(ctx context.Context) string {
	switch a.stageType {
	case StageFirst:
		return "技术一面官: 考察编程基础与算法（权重20%），完成3题后返回阶段评分"
	case StageSecond:
		return "技术二面官: 考察项目经验与架构（权重20%），完成3题后返回阶段评分"
	case StageFinal:
		return "终面官: 综合素质与软实力（权重20%），完成3题后返回阶段评分"
	case StageHR:
		return "HR面官: 职业规划与价值观（权重20%），完成3题后返回阶段评分"
	default:
		return "面试官: 评估候选人的技能与能力"
	}
}

func (a *InterviewStageAgent) Run(ctx context.Context, input *adk.AgentInput, options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	//这个是核心实现，每问完一个问题，就中断，等待用户回答，然后恢复
	//创建一个异步迭代器，用于返回事件
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		//在协程中执行核心逻辑
		defer gen.Close()
		//会话验证
		sessionKeyVal := ctx.Value(SessionKeyCtxKey{})
		if sessionKeyVal == nil {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("session key not found"),
			})
			return
		}
		sessionKey, ok := sessionKeyVal.(string)
		if !ok || sessionKey == "" {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("invalid session key or empty"),
			})
			return
		}
		state := a.provider.GetState(sessionKey)
		if state == nil {
			state = &StageState{
				Stage:       a.stageIndex,
				Round:       0,
				MaxRound:    3,
				History:     []QAPair{},
				StageScores: make(map[int]float64),
			}
		}
		//防御型检查 确保状态和当前的阶段一致
		if state.Stage != a.stageIndex {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("stage mismatch: expected %d, got %d", a.stageIndex, state.Stage),
			})
			return
		}
		//如果当前处于等待回答状态，尝试获取用户的回答并评估
		if state.AwaitingAnswer && state.Round > 0 {
			if answer, ok := a.provider.GetAndClearAnswer(sessionKey); ok && answer != "" {
				//将回答存入历史记录
				lastIdx := len(state.History) - 1
				state.History[lastIdx].Answer = answer
				state.History[lastIdx].Timestamp = time.Now().Unix()
				//调用大模型进行回答的评估
				eval := a.evaluateWithLLM(ctx, state.History[lastIdx])
				state.History[lastIdx].Evaluation = eval.Feedback
				state.History[lastIdx].Scores = eval.Dimensions
				//更新状态
				state.AwaitingAnswer = false
				a.provider.SaveState(sessionKey, state)
				//发送评价结果
				feedbackMsg := fmt.Sprintf("【第%d题评价】%.1f分 - %s \n", state.Round, eval.TotalScore, eval.Feedback)
				gen.Send(&adk.AgentEvent{
					AgentName: a.name,
					Output: &adk.AgentOutput{
						MessageOutput: &adk.MessageVariant{
							Message: schema.AssistantMessage(feedbackMsg, nil),
						},
						CustomizedOutput: map[string]any{
							"question_evaluate": true,
						},
					},
				})
			} else {
				//恢复执行但未获取到回答，继续等待
				gen.Send(adk.Interrupt(ctx, map[string]any{
					"round":    state.Round,
					"stage":    a.stageIndex,
					"question": state.LastQuestion,
					"awaiting": true,
				}))
				return
			}
		}
		//检查是否3道题都完成了 这里我们每个面试官出3道题
		if state.Round >= state.MaxRound && !state.AwaitingAnswer {
			state.Completed = true
			state.Score = a.calculateScore(state)
			//生成阶段总结报告
			report := a.generateReport(state)
			state.StageReport = report
			state.StageScores[state.Stage] = state.Score
			//判断是否通过
			passed := state.Score >= 60
			if passed {
				//进入下一阶段
				state.Stage++
				state.Round = 0
				state.History = []QAPair{}
				state.AwaitingAnswer = false
			}
			a.provider.SaveState(sessionKey, state)
			//发送阶段完成的事件
			gen.Send(&adk.AgentEvent{
				AgentName: a.name,
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						Message: schema.AssistantMessage(fmt.Sprintf("【阶段完成】，%s 结束，你的分数是%.1f分", a.name, state.Score), nil),
					},
					CustomizedOutput: map[string]any{
						"stage_complete": true,
						"passed":         passed,
						"score":          state.Score,
						"stage_name":     a.name,
						"weight":         a.weight,
						"history_count":  len(state.History),
					},
				},
			})
			return
		}
		//生成下一道面试题
		if !state.AwaitingAnswer {
			state.Round++
			question := a.generateQuestion(state)
			state.LastQuestion = question
			state.History = append(state.History, QAPair{
				Question:  question,
				Timestamp: time.Now().Unix(),
			})
			state.AwaitingAnswer = true
			a.provider.SaveState(sessionKey, state)
			//发送中断事件，等待用户回答
			interrupt := adk.Interrupt(ctx, map[string]any{
				"round":    state.Round,
				"stage":    a.stageIndex,
				"question": state.LastQuestion,
				"awaiting": true,
			})
			interrupt.AgentName = a.name
			interrupt.Output = &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					Message: schema.AssistantMessage(question, nil),
				},
			}
			gen.Send(interrupt)
		}
	}()
	return iter
}

func (a *InterviewStageAgent) Resume(ctx context.Context, info *adk.ResumeInfo, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	//TODO implement me
	panic("implement me")
}

func (a *InterviewStageAgent) CheckIfResume(ctx context.Context, input string) *ResumeCheckResult {
	//判断一下 如果llm没有，使用关键词匹配
	if a.llm == nil {
		isResume := strings.Contains(input, "工作年限") ||
			strings.Contains(input, "专业技能") ||
			strings.Contains(input, "项目经验") ||
			strings.Contains(input, "教育背景")
		return &ResumeCheckResult{
			IsResume:   isResume,
			Confidence: 0.7,
		}
	}
	//如果使用大模型，我们需要构建提示词
	prompt := fmt.Sprintf(`请判断以下文本是否为求职简历。如果是，提取关键信息；如果不是，说明原因。

文本内容：
"""
%s
"""

请严格按照以下JSON格式输出（不要markdown）：
{
 "is_resume": true/false,
 "confidence": 0.95,
 "skills": ["Go", "Redis"],
 "experience": ["字节跳动 3年"],
 "name": "张三",
 "suggestion": "建议补充个人项目经历"
}

判断标准：
1. 包含姓名、联系方式、工作经历/项目经验、教育背景中的至少3项
2. 有明确的时间线（如2021-2024）
3. 有技能列表或专业术语
4. 结构清晰，分段明确`, truncateString(input, 2000))
	msgs := []*schema.Message{
		schema.SystemMessage("你是简历解析专家，准确判断文本是否为标准简历格式。"),
		schema.UserMessage(prompt),
	}
	result, err := a.llm.Generate(ctx, msgs)
	if err != nil {
		return &ResumeCheckResult{
			IsResume:   false,
			Confidence: 0.5,
			Suggestion: "无法识别，请提供一份标准的简历",
		}
	}
	return a.parseResumeCheckResult(result.Content)
}

func (a *InterviewStageAgent) parseResumeCheckResult(content string) *ResumeCheckResult {
	result := &ResumeCheckResult{
		IsResume:   false,
		Confidence: 0.5,
		Suggestion: "请提供一份标准的简历",
		Skills:     []string{},
		Experience: []string{},
	}
	//清理一下可能出现的md代码块标记
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	var data map[string]any
	err := json.Unmarshal([]byte(content), &data)
	if err != nil {
		logs.Errorf("json.Unmarshal 解析失败: %v", err)
		return result
	}
	if v, ok := data["is_resume"].(bool); ok {
		result.IsResume = v
	}
	if v, ok := data["confidence"].(float64); ok {
		result.Confidence = v
	}
	if v, ok := data["suggestion"].(string); ok {
		result.Suggestion = v
	}
	if v, ok := data["skills"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				result.Skills = append(result.Skills, s)
			}
		}
	}
	if v, ok := data["experience"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				result.Experience = append(result.Experience, s)
			}
		}
	}
	return result
}

func (a *InterviewStageAgent) evaluateWithLLM(ctx context.Context, qa QAPair) *EvalResult {
	prompt := fmt.Sprintf(`评价回答（0-100分）：
【问题】%s
【回答】%s（%d字）

JSON格式：{"total_score":85,"dimensions":{"%s":85},"feedback":"评价","red_flags":[]}`,
		qa.Question, qa.Answer, len(qa.Answer), a.dimensions[0])
	msgs := []*schema.Message{
		schema.SystemMessage("严格面试官。90-100优秀，80-89良好，70-79合格，<70不合格。"),
		schema.UserMessage(prompt),
	}
	result, err := a.llm.Generate(ctx, msgs)
	if err != nil {
		logs.Errorf("评价回答失败: %v", err)
		return &EvalResult{
			TotalScore: 75,
			Dimensions: make(map[string]float64),
			Feedback:   "默认评价",
			RedFlags:   []string{},
		}
	}
	return a.parseEvalResult(result.Content)
}

func (a *InterviewStageAgent) parseEvalResult(content string) *EvalResult {
	result := &EvalResult{
		TotalScore: 75,
		Dimensions: make(map[string]float64),
		Feedback:   "默认评价",
		RedFlags:   []string{},
	}
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	var data map[string]any
	err := json.Unmarshal([]byte(content), &data)
	if err != nil {
		logs.Errorf("json.Unmarshal 解析失败: %v", err)
		return result
	}
	if v, ok := data["total_score"].(float64); ok {
		result.TotalScore = v
	}
	if v, ok := data["dimensions"].(map[string]any); ok {
		for k, v := range v {
			if f, ok := v.(float64); ok {
				result.Dimensions[k] = f
			}
		}
	}
	if v, ok := data["feedback"].(string); ok {
		result.Feedback = v
	}
	if v, ok := data["red_flags"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				result.RedFlags = append(result.RedFlags, s)
			}
		}
	}
	return result
}

func (a *InterviewStageAgent) generateQuestion(state *StageState) string {
	//构建完整的上下文，帮助生成的问题更符合要求
	fullContext := a.buildInterviewContext(state)
	//构建提示词
	prompt := fmt.Sprintf(`你是%s（第%d轮，第%d题/共%d题）。

%s

【重要规则】
1. 必须结合简历：从简历中挑选一个具体经历/技术点切入，禁止凭空捏造问题。
2. 可以参考前几轮的评价来决定问题的难易程度和考察重点。
3. 当前必须生成第%d题。
4. 禁止在问题前添加"第X题"或"Q1"等题号前缀，直接写问题内容。
5. 只出一道题。

请生成第%d题：`,
		a.name,
		state.Stage+1, state.Round, state.MaxRound,
		fullContext,
		state.Round, state.Round,
	)
	msgs := []*schema.Message{
		schema.SystemMessage(fmt.Sprintf("你是一位资深面试官(%s)。", a.name)),
		schema.UserMessage(prompt),
	}
	result, err := a.llm.Generate(context.Background(), msgs)
	if err != nil || result.Content == "" {
		logs.Errorf("生成问题失败: %v", err)
		return "这里可以写一些默认问题"
	}
	return result.Content
}

func (a *InterviewStageAgent) buildInterviewContext(state *StageState) string {
	var contextBuilder strings.Builder
	if state.ResumeContext != "" {
		//这是候选人简历
		contextBuilder.WriteString(fmt.Sprintf("【候选人简历】%s\n", state.ResumeContext))
		contextBuilder.WriteString("\n")
	}
	//阶段总结
	if len(state.PreStagesSummary) > 0 {
		contextBuilder.WriteString("【前面面试阶段评价】\n")
		for _, summary := range state.PreStagesSummary {
			contextBuilder.WriteString(fmt.Sprintf("* 【%s】（%.1f分）\n", summary.StageName, summary.AvgScore))
		}
		contextBuilder.WriteString("\n")
	}
	//添加本轮的面试记录
	if len(state.History) > 0 {
		contextBuilder.WriteString("【本轮面试记录】\n")
		for i, qa := range state.History {
			contextBuilder.WriteString(fmt.Sprintf("Q%d: %s\n", i+1, qa.Question))
			if qa.Answer != "" {
				contextBuilder.WriteString(fmt.Sprintf("A%d: %s\n", i+1, truncateString(qa.Answer, 100)))
				if qa.Evaluation != "" {
					contextBuilder.WriteString(fmt.Sprintf("评价：%.1f分: %s\n", qa.Scores["total"], qa.Evaluation))
				}
			}
		}
		contextBuilder.WriteString("\n")
	}
	//本轮考察重点
	contextBuilder.WriteString("【本轮考察重点】\n")
	for _, dimension := range a.dimensions {
		contextBuilder.WriteString(fmt.Sprintf("* %s\n", dimension))
	}
	contextBuilder.WriteString("\n")
	return contextBuilder.String()
}

func (a *InterviewStageAgent) calculateScore(state *StageState) float64 {
	if len(state.History) == 0 {
		return 0
	}
	var total float64
	var count int
	//累加所有维度的分数
	for _, qa := range state.History {
		if qa.Scores != nil {
			for _, score := range qa.Scores {
				total += score
				count++
			}
		}
	}
	if count == 0 {
		return 70 //默认分数
	}
	avg := total / float64(count)
	//危险信号扣分
	for _, qa := range state.History {
		if strings.Contains(qa.Evaluation, "抄袭") ||
			strings.Contains(qa.Evaluation, "虚假") {
			avg -= 20
		}
	}
	if avg < 0 {
		return 0
	}
	if avg > 100 {
		return 100
	}
	return avg
}

func (a *InterviewStageAgent) generateReport(state *StageState) string {
	var strengths []string
	var weaknesses []string
	for _, qa := range state.History {
		for dim, score := range qa.Scores {
			if score >= 85 {
				strengths = append(strengths, dim)
			} else if score < 70 {
				weaknesses = append(weaknesses, dim)
			}
		}
	}
	report := fmt.Sprintf("%s完成，得分%.1f分", a.name, state.Score)
	if len(strengths) > 0 {
		report += fmt.Sprintf("\n\n【强项】%s", strings.Join(strengths, "、"))
	}
	if len(weaknesses) > 0 {
		report += fmt.Sprintf("\n\n【弱项】%s", strings.Join(weaknesses, "、"))
	}
	return report
}

// truncateString 截取字符串 避免过长
func truncateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}
	return input[:maxLength] + "..."
}
func NewInterviewStageAgent(
	name string,
	stageType StageType,
	stageIndex int,
	provider StateProvider,
	llm model.ToolCallingChatModel,
	weight float64,
	dimensions []string) *InterviewStageAgent {
	return &InterviewStageAgent{
		name:       name,
		stageType:  stageType,
		stageIndex: stageIndex,
		provider:   provider,
		llm:        llm,
		weight:     weight,
		dimensions: dimensions,
	}
}
