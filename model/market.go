package model

import "github.com/google/uuid"

// AgentMarket agent市场模型，用于存储a2a server的信息
type AgentMarket struct {
	Id          uuid.UUID `json:"id" gorm:"column:id;type:uuid;primary_key;comment:ID"`
	URL         string    `json:"url" gorm:"column:url;type:varchar(255);not null;comment:Agent URL地址"`
	Name        string    `json:"name" gorm:"column:name;type:varchar(255);not null;comment:Agent名称"`
	Description string    `json:"description" gorm:"column:description;type:text;comment:Agent描述信息"`
	HandlerPath string    `json:"handlerPath" gorm:"column:handler_path;type:varchar(255);not null;comment:Handler路径"`
}

// TableName 指定表名
func (am *AgentMarket) TableName() string {
	return "agent_market"
}
