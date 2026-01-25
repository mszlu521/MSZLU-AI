package nodes

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
)

type TextCombineNode struct {
	data map[string]any
}

func NewTextCombineNode(data map[string]any) *TextCombineNode {
	return &TextCombineNode{
		data: data,
	}
}

func (t *TextCombineNode) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	//这就是节点的实现逻辑
	//这里输入的数据有两个获取途径，一个是通过data字段，一个是通过input字段
	//data字段是节点的配置数据，input字段是节点的输入数据
	var templateStr string
	_, ok := input["template"]
	if !ok {
		templateStr = t.data["template"].(map[string]any)["fieldValue"].(string)
	} else {
		templateStr = input["template"].(string)
	}
	//变量
	variablesMap := make(map[string]any)
	hasVariables := false
	if len(input) > 0 {
		for key, value := range input {
			if key != "template" {
				variablesMap[key] = fmt.Sprintf("%v", value)
				hasVariables = true
			}
		}
	}
	if !hasVariables {
		arr := t.data["variables"].([]any)
		for _, v := range arr {
			m := v.(map[string]any)
			variablesMap[m["fieldName"].(string)] = m["fieldValue"]
		}
	}
	//渲染模版
	var buf bytes.Buffer
	tmpl, err := template.New("TextCombine").Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("parse template error: %v", err)
	}
	err = tmpl.Execute(&buf, variablesMap)
	if err != nil {
		return nil, fmt.Errorf("execute template error: %v", err)
	}
	result := map[string]any{
		"output": buf.String(),
	}
	for k, v := range variablesMap {
		result[k] = v
	}
	return result, nil
}
