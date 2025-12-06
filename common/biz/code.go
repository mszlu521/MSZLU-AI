package biz

import "github.com/mszlu521/thunder/errs"

var (
	ErrUserNameExisted  = errs.NewError(10001, "用户名已存在")
	ErrEmailExisted     = errs.NewError(10002, "邮箱已存在")
	ErrPasswordFormat   = errs.NewError(10003, "密码格式错误")
	ErrTokenInvalid     = errs.NewError(10004, "token无效")
	ErrUserNotFound     = errs.NewError(10005, "用户不存在")
	ErrEmailNotVerified = errs.NewError(10006, "邮箱未验证")
	ErrPasswordInvalid  = errs.NewError(10007, "密码错误")
	ErrTokenGen         = errs.NewError(10008, "token生成失败")
	ErrCodeGen          = errs.NewError(10009, "验证码生成失败")
	ErrCodeInvalid      = errs.NewError(10010, "验证码错误")
	ErrEmailNotMatch    = errs.NewError(10011, "邮箱不匹配")
)
var (
	AgentNotFound             = errs.NewError(20001, "Agent不存在")
	ErrProviderConfigNotFound = errs.NewError(20002, "ProviderConfig不存在")
)
