package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time      `json:"createdAt" gorm:"column:created_at;not null"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"column:updated_at;not null"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"column:deleted_at;index"`
}

type JSON map[string]interface{}

// Scan 实现 sql.Scanner 接口，用于从数据库读取 JSON 数据
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}
	// 将数据库中的值转换为字节切片
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSON", value)
	}

	// 解析 JSON 数据
	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Value 实现 driver.Valuer 接口，用于将 JSON 数据写入数据库
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return make(JSON), nil
	}

	// 将 JSON 转换为字节切片
	return json.Marshal(j)
}
