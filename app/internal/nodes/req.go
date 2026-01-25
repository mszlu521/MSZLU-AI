package nodes

import "core/ai/nodes"

type TestNodeReq struct {
	NodeType  nodes.NodeType         `json:"nodeType"`
	NodeData  map[string]interface{} `json:"nodeData"`
	InputData map[string]interface{} `json:"inputData"`
}
