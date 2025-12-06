package ai

import "encoding/json"

// AgentMessage 这个是返回客户端的json消息
type AgentMessage struct {
	Action           string `json:"action"`
	AgentName        string `json:"agentName"`
	ToolName         string `json:"toolName"`
	IsErr            bool   `json:"isErr"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoningContent"`
}

func BuildErrMessage(agentName string, errMsg string) string {
	msg := AgentMessage{
		Action:    "agent_answer", //前端会监听这个action 根据这个action进行消息处理
		AgentName: agentName,
		IsErr:     true,
		Content:   errMsg,
	}
	bytes, _ := json.Marshal(msg)
	return string(bytes)
}

func BuildReasoningMessage(name string, toolName string, reasoningContent string) string {
	msg := AgentMessage{
		Action:           "agent_answer",
		AgentName:        name,
		ToolName:         toolName,
		ReasoningContent: reasoningContent,
	}
	bytes, _ := json.Marshal(msg)
	return string(bytes)
}

func BuildMessage(name string, toolName string, content string) string {
	msg := AgentMessage{
		Action:    "agent_answer",
		AgentName: name,
		ToolName:  toolName,
		Content:   content,
	}
	bytes, _ := json.Marshal(msg)
	return string(bytes)
}
