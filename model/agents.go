package model

import (
	"time"

	"github.com/google/uuid"
)

type AgentStatus string

var (
	Draft     AgentStatus = "draft"
	Published             = "published"
	Archived              = "archived"
)

type AgentVisibility string

var (
	Private  AgentVisibility = "private"
	Public                   = "public"
	LinkOnly                 = "link_only"
)

// Agent 定义了智能代理的模型
type Agent struct {
	BaseModel
	// CreatorID 创建者ID，标识该agent的创建者
	CreatorID uuid.UUID `json:"creatorId" gorm:"column:creator_id;type:uuid;not null"`
	// Name agent名称
	Name string `json:"name" gorm:"column:name;type:varchar(255);not null"`
	// Description 描述信息
	Description string `json:"description" gorm:"column:description;type:text"`
	// Icon 图标URL或路径
	Icon string `json:"icon" gorm:"column:icon;type:varchar(512)"`
	// SystemPrompt 系统提示词，用于指导AI行为
	SystemPrompt string `json:"systemPrompt" gorm:"column:system_prompt;type:text"`
	// ModelProvider 模型提供商（例如openai）
	ModelProvider string `json:"modelProvider" gorm:"column:model_provider;type:varchar(50);not null;default:'openai'"`
	// ModelName 使用的具体模型名称
	ModelName string `json:"modelName" gorm:"column:model_name;type:varchar(100);not null"`
	// ModelParameters 模型参数配置
	ModelParameters JSON `json:"modelParameters" gorm:"column:model_parameters;type:jsonb"`
	// OpeningDialogue 开场白对话内容
	OpeningDialogue string `json:"openingDialogue" gorm:"column:opening_dialogue;type:text"`
	// SuggestedQuestions 建议问题列表
	SuggestedQuestions JSON `json:"suggestedQuestions" gorm:"column:suggested_questions;type:jsonb"`
	// Version 版本号
	Version uint `json:"version" gorm:"column:version;type:int;not null;default:1"`
	// Status 状态（草稿、发布、归档）
	Status AgentStatus `json:"status" gorm:"column:status;type:varchar(20);not null;default:'draft'"`
	// Visibility 可见性（私有、公开、仅链接）
	Visibility AgentVisibility `json:"visibility" gorm:"column:visibility;type:varchar(20);not null;default:'private'"`
	// InvocationCount 调用次数统计
	InvocationCount uint64 `json:"invocationCount" gorm:"column:invocation_count;type:bigint;not null;default:0"`
	// PublishedAt 发布时间戳
	PublishedAt *time.Time `json:"publishedAt" gorm:"column:published_at;type:timestamptz"`
}

// TableName 返回表名
func (Agent) TableName() string {
	return "agents"
}

// ModelParams 定义了模型参数的结构
type ModelsParams struct {
	// MaxTokens 最大生成长度（单位：Token）。
	// 限制 AI 回复的最大长度，不包含输入的 Prompt 长度。
	// 注意：(Input Tokens + MaxTokens) 不能超过模型的上下文窗口上限。
	MaxTokens int `json:"maxTokens"`
	// 控制模型输出的随机性。
	// - 0.0: 几乎是确定性的，每次运行结果基本相同（适合代码生成、数学解题）。
	// - 1.0+: 增加多样性，甚至可能产生幻觉（适合创意写作）。
	Temperature float64 `json:"temperature"`
	// TopP (核采样 / Nucleus Sampling)
	// 模型只考虑累积概率达到 TopP 的 Token 集合。
	// 例如 0.1 意味着只考虑概率最高的顶层 10% 的词汇。
	// *最佳实践*：一般建议修改 Temperature 或 TopP 其中之一，而不是同时修改。
	TopP float64 `json:"topP"`
	// N (生成数量)
	// 针对同一条提示词，一次性生成多少条独立的回复。
	// 适用于需要从多个结果中进行优选的场景。
	N int `json:"n"`
	// Stop (停止词)
	// 一个字符串或字符串数组。
	// 一旦模型生成的文本中包含该序列，生成过程将立即终止，且返回结果不包含该停止词。
	Stop []any `json:"stop"`
	// PresencePenalty (话题新鲜度惩罚)
	// 基于 Token "是否已经出现过" 进行惩罚（不考虑出现次数）。
	// 值越大，模型越倾向于转换话题，避免在同一个概念上打转。
	PresencePenalty float64 `json:"presencePenalty"`
	// FrequencyPenalty (重复度惩罚)
	// 基于 Token "出现的频次" 进行惩罚。
	// 值越大，模型越排斥逐字逐句地重复之前生成过的文本（减少复读机现象）。
	FrequencyPenalty float64 `json:"frequencyPenalty"`
}

func (j JSON) ToModelParams() ModelsParams {
	params := ModelsParams{}

	if maxTokens, ok := j["maxTokens"].(float64); ok {
		params.MaxTokens = int(maxTokens)
	}

	if temperature, ok := j["temperature"].(float64); ok {
		params.Temperature = temperature
	}

	if topP, ok := j["topP"].(float64); ok {
		params.TopP = topP
	}

	if n, ok := j["n"].(float64); ok {
		params.N = int(n)
	}

	if stop, ok := j["stop"].([]any); ok {
		params.Stop = stop
	}

	if presencePenalty, ok := j["presencePenalty"].(float64); ok {
		params.PresencePenalty = presencePenalty
	}

	if frequencyPenalty, ok := j["frequencyPenalty"].(float64); ok {
		params.FrequencyPenalty = frequencyPenalty
	}

	return params
}

func DefaultAgent(userId uuid.UUID, name string, description string, status AgentStatus) *Agent {
	return &Agent{
		BaseModel: BaseModel{
			ID: uuid.New(),
		},
		CreatorID:   userId,
		Name:        name,
		Description: description,
		Status:      status,
		//这个暂时没用 前端没有实现
		SuggestedQuestions: JSON{},
		OpeningDialogue:    "",
		SystemPrompt:       "",
		ModelProvider:      "",
		ModelName:          "",
		ModelParameters:    JSON{},
		Version:            1,
		Visibility:         Private,
		InvocationCount:    0,
	}
}
