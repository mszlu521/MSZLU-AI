package ai

import (
	"context"
	"core/ai/nodes"
	"encoding/json"
	"fmt"
	"model"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type WorkflowTool struct {
	workflow *model.Workflow
	node     *model.Node
}

func (w *WorkflowTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	// 我们需要把工作流的信息转换成工具的描述
	//这个就是节点和边的数据以及关系以及描述都有
	configJSON, _ := json.Marshal(w.workflow.Data)
	inputParams := make(map[string]*schema.ParameterInfo)
	var startNode *model.Node
	for _, node := range w.workflow.Data.Nodes {
		if node.Type == string(nodes.Start) {
			startNode = node
			break
		}
	}
	if startNode != nil {
		//分析从start开始的所有边
		for _, edge := range w.workflow.Data.Edges {
			if edge.Source == startNode.ID {
				var targetNode *model.Node
				for _, node := range w.workflow.Data.Nodes {
					if node.ID == edge.Target {
						targetNode = node
						break
					}
				}
				//如果目标节点存在且有数据定义，将其做为参数
				if targetNode != nil && targetNode.Data != nil {
					w.node = targetNode
					for key, fieldData := range targetNode.Data {
						if fieldMap, ok := fieldData.(map[string]any); ok {
							if targetNode.Type == string(nodes.QwenVL) {
								//模型参数不作为提供给大模型的参数
								if key == "model" || key == "providers" || key == "promptType" {
									continue
								}
							}
							fieldName := "unknown"
							if name, ok := fieldMap["fieldName"].(string); ok {
								fieldName = name
							}
							desc := fieldName
							if d, ok := fieldMap["fieldDesc"].(string); ok {
								desc = d
							}
							required := false
							if r, ok := fieldMap["required"].(bool); ok {
								required = r
							}
							inputParams[key] = &schema.ParameterInfo{
								Desc:     desc,
								Required: required,
								Type:     schema.String,
							}
						}
					}
				}
			}
		}
	}
	if len(inputParams) == 0 {
		//提供的默认参数
		inputParams["input"] = &schema.ParameterInfo{
			Desc: "工作流执行的输入参数",
			Type: schema.String,
		}
	}
	return &schema.ToolInfo{
		Name: "execute_workflow_" + w.workflow.Name,
		Desc: fmt.Sprintf("执行名为:%s的工作流，工作流描述为:%s。工作流配置%s",
			w.workflow.Name,
			w.workflow.Description,
			string(configJSON)),
		ParamsOneOf: schema.NewParamsOneOfByParams(inputParams),
	}, nil
}
func (w *WorkflowTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	//工作流执行器
	executor := NewWorkflowExecutor()
	executor.initRegistry()
	//解析输入的参数
	var inputParams map[string]any
	err := json.Unmarshal([]byte(argumentsInJSON), &inputParams)
	if err != nil {
		return "", err
	}
	//转换参数
	params := w.ConvertParams(inputParams)
	if w.node != nil {
		for _, v := range w.workflow.Data.Nodes {
			if v.ID == w.node.ID {
				for k, value := range params {
					v.Data[k] = value
				}
			}
		}
	}
	result, err := executor.Execute(w.workflow.Data)
	if err != nil {
		return "", err
	}
	resultJson, _ := json.Marshal(result)
	return string(resultJson), nil
}

func (w *WorkflowTool) ConvertParams(params map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range params {
		result[k] = map[string]any{
			"fieldValue": v,
		}
	}
	return result
}

func NewWorkflowTool(workflow *model.Workflow) *WorkflowTool {
	return &WorkflowTool{
		workflow: workflow,
	}
}
