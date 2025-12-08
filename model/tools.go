package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type ToolType string

const (
	McpToolType    ToolType = "mcp"
	SystemToolType          = "system"
)

// Tool 定义了工具的模型
type Tool struct {
	BaseModel
	// 添加索引，通常查询工具会根据创建者查询
	CreatorID uuid.UUID `json:"creatorId" gorm:"type:uuid;index;not null"`
	// 名称通常需要唯一性或者作为查找条件，限制长度
	Name        string   `json:"name" gorm:"size:255;not null;index"`
	Description string   `json:"description" gorm:"type:text"`
	ToolType    ToolType `json:"toolType" gorm:"size:50;not null"`
	IsEnable    bool     `json:"isEnable" gorm:"default:true"`
	// 显式指定 type:jsonb，PostgreSQL 能够更好地查询和存储
	ParametersSchema ParametersSchema `json:"parametersSchema" gorm:"type:jsonb"`
	// 指针类型允许存 NULL
	McpConfig *McpConfig `json:"mcpConfig" gorm:"type:jsonb"`
	// 关联关系
	// 注意：如果你需要在 agent_tools 中存储额外字段（如 Status），
	// 在 GORM 代码逻辑中可能需要使用 SetupJoinTable，或者将 Many2Many 改为 HasMany AgentTools
	Agents []Agent `json:"agents" gorm:"many2many:agent_tools;"`
}

// TableName 返回表名
func (Tool) TableName() string {
	return "tools"
}

type McpConfig struct {
	// sse 等
	Type string `json:"type,omitempty"`

	Url string `json:"url,omitempty"`

	AuthenticationRequired bool `json:"authenticationRequired,omitempty"`

	CredentialType string `json:"credentialType,omitempty"`
}

// Value - 实现 driver.Valuer 接口
// 当 GORM 保存数据时，会调用此方法将 McpConfig 转换为数据库能识别的类型
func (c McpConfig) Value() (driver.Value, error) {
	// 使用 json.Marshal 将结构体编码为 JSON 字节切片
	return json.Marshal(c)
}

// Scan - 实现 sql.Scanner 接口
// 当 GORM 查询数据时，会调用此方法将从数据库读取的值填充到 McpConfig 中
func (c *McpConfig) Scan(value interface{}) error {
	// 首先，检查从数据库接收到的值是否为 nil
	if value == nil {
		// 如果是 nil, 我们也将结构体指针设为 nil
		// 注意: Scan 方法的接收者是指针，但在这里我们不能直接将 c 设为 nil
		// 实际上，如果数据库值为 NULL, GORM 在调用 Scan 之前就会处理好指针为 nil 的情况。
		// 这个检查主要是为了健壮性。
		return nil
	}

	// 数据库驱动通常返回 []byte 类型给 json/jsonb 字段
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	// 使用 json.Unmarshal 将 JSON 字节切片解码到结构体中
	return json.Unmarshal(bytes, c)
}

type ParametersSchema map[string]*schema.ParameterInfo

// Value - 实现 driver.Valuer 接口，用于将 JSONSchema 存入数据库
func (s ParametersSchema) Value() (driver.Value, error) {
	// 将结构体序列化为 JSON 字节
	return json.Marshal(s)
}

// Scan - 实现 sql.Scanner 接口，用于从数据库读取并解析到 JSONSchema
func (s *ParametersSchema) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	// 将 JSON 字节反序列化到结构体中
	return json.Unmarshal(bytes, s)
}
