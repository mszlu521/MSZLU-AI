package model

type SubscriptionPlan string

const (
	FreePlan       SubscriptionPlan = "free"       // 免费版
	BasicPlan      SubscriptionPlan = "basic"      // 基础版
	ProPlan        SubscriptionPlan = "pro"        // 高级版
	EnterprisePlan SubscriptionPlan = "enterprise" // 企业版
)

type PlanConfig struct {
	MaxAgents            int64 `json:"maxAgents"`
	MaxWorkflows         int64 `json:"maxWorkflows"`
	MaxKnowledgeBaseSize int64 `json:"maxKnowledgeBaseSize"`
}

type PaymentDuration string

const (
	Monthly   PaymentDuration = "month"   // 月付
	Quarterly PaymentDuration = "quarter" // 季付
	Yearly    PaymentDuration = "year"    // 年付
)

type PaymentMethod string

const (
	WeChatPay PaymentMethod = "wechat" // 微信支付
)
