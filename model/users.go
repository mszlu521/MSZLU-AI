package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	// Id: 设置为主键，类型为 uuid，让数据库自动生成默认值
	Id uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	// Username: 添加唯一索引，非空
	Username string `json:"username" gorm:"uniqueIndex;not null"`
	// Password: 基本存储
	Password string `json:"password"`
	// Avatar: 头像链接
	Avatar        string           `json:"avatar"`
	Status        StatusEnum       `json:"status" gorm:"type:smallint;default:3"`
	LastLoginTime time.Time        `json:"lastLoginTime"`
	CurrentPlan   SubscriptionPlan `json:"currentPlan" gorm:"type:varchar(20);default:'free'"`
	Email         string           `json:"email" gorm:"type:varchar(100);uniqueIndex;not null"`
	EmailVerified bool             `json:"emailVerified" gorm:"type:boolean;default:false"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

type StatusEnum int

var (
	UserStatusNormal  StatusEnum = 1
	UserStatusDisable StatusEnum = 2
	UserStatusPending StatusEnum = 3 // For users waiting for email verification
)

type UserDTO struct {
	Id            uuid.UUID        `json:"id"`
	Username      string           `json:"username"`
	Avatar        string           `json:"avatar"`
	Status        StatusEnum       `json:"status"`
	LastLoginTime time.Time        `json:"lastLoginTime"`
	CurrentPlan   SubscriptionPlan `json:"currentPlan"`
}
