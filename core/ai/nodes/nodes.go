package nodes

import "context"

// fieldName: '文本标题',
//
//	fieldValue: '文本标题',
//	fieldDesc: '文本标题',
//	fieldType: 'div',
//	list: false,
//	options: [],
//	show: true,
//	required: true,
//

type NodeType string

const (
	Start       NodeType = "start"
	End         NodeType = "end"
	TextCombine NodeType = "textCombine"
	TextDisplay NodeType = "textDisplay"
	HtmlDisplay NodeType = "htmlDisplay"
	QwenVL      NodeType = "qwenVL"
)

// WorkflowNode
type WorkflowNode interface {
	//Invoke 接收一个输入 返回一个输出
	Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
}
