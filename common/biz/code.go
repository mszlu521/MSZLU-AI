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

var (
	ErrToolNameExisted     = errs.NewError(30001, "工具名称已存在")
	ErrToolNotExisted      = errs.NewError(30002, "工具不存在")
	ErrMcpConfigNotExisted = errs.NewError(30003, "McpConfig不存在")
	ErrGetMcpTools         = errs.NewError(30004, "获取McpTools失败")
)
var (
	ErrKnowledgeBaseNotFound   = errs.NewError(40001, "知识库不存在")
	FileLoadError              = errs.NewError(40002, "文件加载错误")
	ErrDocumentNotFound        = errs.NewError(40003, "文档不存在")
	ErrEmbeddingConfigNotFound = errs.NewError(40004, "EmbeddingConfig不存在")
	ErrEmbedding               = errs.NewError(40005, "Embedding错误")
	ErrRetriever               = errs.NewError(40006, "Retriever错误")
)
