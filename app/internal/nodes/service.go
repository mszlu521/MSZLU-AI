package nodes

import (
	"context"
	"core/ai/nodes"
	"fmt"
)

type service struct {
}

func (s *service) testNode(ctx context.Context, reqs TestNodeReq) (*testNodeResp, error) {
	resp := &testNodeResp{}
	nodeType := reqs.NodeType
	var node nodes.WorkflowNode
	switch nodeType {
	case nodes.TextCombine:
		node = nodes.NewTextCombineNode(reqs.NodeData)
	case nodes.TextDisplay:
		node = nodes.NewTextDisplayNode(reqs.NodeData)
	case nodes.HtmlDisplay:
		node = nodes.NewHtmlDisplayNode(reqs.NodeData)
	case nodes.QwenVL:
		node = nodes.NewQwenVLNode(reqs.NodeData)
	default:
		resp.Error = fmt.Sprintf("不支持的节点类型: %s", nodeType)
		return resp, nil
	}
	output, err := node.Invoke(ctx, reqs.InputData)
	if err != nil {
		resp.Error = err.Error()
		return resp, nil
	}
	resp.Output = output
	resp.Success = true
	return resp, nil
}

func newService() *service {
	return &service{}
}
