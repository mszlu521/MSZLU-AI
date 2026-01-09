# MCP（Model Context Protocol）原理与 Go 实现详解

> **MCP（Model Context Protocol）** 是一种用于 **大模型与外部系统进行上下文交互的协议**，它定义了模型如何获取工具、资源、状态，并在一次或多次对话中保持一致的上下文能力。

随着 Agent、RAG、工具调用、工作流的普及，**“如何让模型安全、标准化、可扩展地访问外部能力”** 成为了核心问题，MCP 正是为了解决这一问题而诞生。

------

## 一、为什么需要 MCP？

### 1️⃣ 传统 LLM 的局限

传统的大模型调用方式：

```
User → Prompt → LLM → Response
```

存在明显问题：

- ❌ 模型无法主动访问外部系统
- ❌ 无法获取实时数据
- ❌ 工具调用高度定制，难以复用
- ❌ 上下文与状态无法统一管理

------

### 2️⃣ Agent / 工具调用带来的新问题

当我们引入：

- 搜索工具
- 数据库
- 知识库
- 文件系统
- 业务 API

就会出现：

- 工具接口不统一
- 上下文注入方式混乱
- 前后端难以协同
- Agent 与工具强耦合

👉 **需要一个“模型 ↔ 工具 ↔ 资源”的标准协议**

------

## 二、什么是 MCP？

### MCP 的核心定义

> **MCP 是一个协议层，用于规范大模型如何发现能力、调用工具、读取资源，并在会话中维护上下文状态。**

它不是一个模型，也不是一个框架，而是：

- ✅ **协议**
- ✅ **接口约定**
- ✅ **通信规范**

------

### MCP 解决了什么？

| 问题           | MCP 的解决方式          |
| -------------- | ----------------------- |
| 工具如何暴露   | `tools/list`            |
| 工具如何调用   | `tools/call`            |
| 资源如何访问   | `resources/list / read` |
| 上下文如何维护 | Session / Message       |
| 实时交互       | SSE / Streamable HTTP   |

------

## 三、MCP 的整体架构

```
+-------------------+
|   LLM / Agent     |
+---------+---------+
|
| MCP Request
v
+---------+---------+
|   MCP Server      |
|-------------------|
| Tools             |
| Resources         |
| Context           |
+---------+---------+
|
v
+-------------------+
| External Systems  |
| DB / FS / API     |
+-------------------+
```

------

## 四、MCP 的核心概念

### 1️⃣ Tool（工具）

模型可调用的能力：

```
{
"name": "search_user",
"description": "根据关键词搜索用户",
"input_schema": {
"type": "object",
"properties": {
"keyword": { "type": "string" }
}
}
}
```

------

### 2️⃣ Resource（资源）

模型可读取的上下文数据：

- 文件
- 文档
- 配置
- 知识库

```
{
"uri": "file:///docs/intro.md",
"mimeType": "text/markdown"
}
```

------

### 3️⃣ Transport（通信方式）

MCP 支持多种传输方式：

| 方式            | 说明       |
| --------------- | ---------- |
| stdio           | 本地进程   |
| SSE             | 长连接流式 |
| Streamable HTTP | 单端点流式 |

------

## 五、MCP 的交互流程

### 一个完整的调用流程

```
1. Client → /initialize
2. Client → /tools/list
3. LLM 决定调用工具
4. Client → /tools/call
5. MCP Server 执行工具
6. 返回结果 → LLM
```

------

## 六、用 Go 实现一个最小 MCP Server

下面实现一个**最小可运行的 MCP Server（SSE 版本）**，用于暴露一个简单工具。

------

### 1️⃣ 项目结构

```
mcp-server/
├── main.go
├── tools.go
└── types.go
```

------

### 2️⃣ 定义 MCP 基础结构（types.go）

```
package main

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}
```

------

### 3️⃣ 定义工具（tools.go）

```
package main

func ListTools() []Tool {
	return []Tool{
		{
			Name:        "hello",
			Description: "向用户打招呼",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]string{
						"type": "string",
					},
				},
			},
		},
	}
}

func CallTool(req ToolCallRequest) string {
	if req.Name == "hello" {
		name, _ := req.Arguments["name"].(string)
		return "Hello, " + name
	}
	return "unknown tool"
}
```

------

### 4️⃣ 实现 MCP HTTP Server（main.go）

```
package main

import (
"encoding/json"
"net/http"
)

func main() {
	http.HandleFunc("/tools/list", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ListTools())
	})

	http.HandleFunc("/tools/call", func(w http.ResponseWriter, r *http.Request) {
		var req ToolCallRequest
		json.NewDecoder(r.Body).Decode(&req)
		result := CallTool(req)
		json.NewEncoder(w).Encode(map[string]string{
			"result": result,
		})
	})

	http.ListenAndServe(":8080", nil)
}
```

------

### 5️⃣ 测试调用

```
curl http://localhost:8080/tools/list
curl -X POST http://localhost:8080/tools/call \
-H "Content-Type: application/json" \
-d '{"name":"hello","arguments":{"name":"MCP"}}'
```

------

## 七、MCP 与传统 API 的区别

| 对比项   | REST API  | MCP    |
| -------- | --------- | ------ |
| 调用方   | 人 / 程序 | LLM    |
| 描述方式 | 文档      | Schema |
| 决策者   | 开发者    | 模型   |
| 上下文   | 无        | 强     |
| 扩展性   | 一般      | 极强   |

------

## 八、MCP 的典型应用场景

- 🤖 AI Agent 系统
- 📚 RAG 知识库
- 🔄 工作流编排
- 🧠 多 Agent 协作
- 🛠️ AI 工具市场（Tool Marketplace）

------

## 九、总结

> **MCP 的本质是：让模型成为“会用工具的软件系统”**

它通过协议化、标准化的方式，让：

- 工具可发现
- 能力可组合
- 上下文可持续
- Agent 可扩展

在 Go 生态下，MCP 非常适合用于：

- 高性能 Agent 后端
- 可插拔工具系统
- 企业级 AI 平台