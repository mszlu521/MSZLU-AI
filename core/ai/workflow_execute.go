package ai

import (
	"context"
	"core/ai/nodes"
	"fmt"
	"model"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/mszlu521/thunder/logs"
)

var Executor *WorkflowExecutor

func Init() {
	Executor = NewWorkflowExecutor()
	Executor.initRegistry()
}

type EdgeKey struct {
	Source string
	Target string
}
type EdgeValues struct {
	SourceHandles []string
	TargetHandles []string
}
type NodeFactory func(data map[string]any) nodes.WorkflowNode
type WorkflowExecutor struct {
	// 节点注册表
	nodeRegistry map[nodes.NodeType]NodeFactory
	//保证只初始化一次
	once sync.Once
}

func NewWorkflowExecutor() *WorkflowExecutor {
	return &WorkflowExecutor{
		nodeRegistry: make(map[nodes.NodeType]NodeFactory),
	}
}

func (w *WorkflowExecutor) initRegistry() {
	w.once.Do(func() {
		w.nodeRegistry[nodes.TextDisplay] = func(data map[string]any) nodes.WorkflowNode {
			return nodes.NewTextDisplayNode(data)
		}
		w.nodeRegistry[nodes.TextCombine] = func(data map[string]any) nodes.WorkflowNode {
			return nodes.NewTextCombineNode(data)
		}
		w.nodeRegistry[nodes.HtmlDisplay] = func(data map[string]any) nodes.WorkflowNode {
			return nodes.NewHtmlDisplayNode(data)
		}
		w.nodeRegistry[nodes.QwenVL] = func(data map[string]any) nodes.WorkflowNode {
			return nodes.NewQwenVLNode(data)
		}
	})
}

func (w *WorkflowExecutor) Execute(data *model.Graph) (map[string]any, error) {
	ctx := context.Background()
	//使用eino框架实现工作流
	wf := compose.NewWorkflow[map[string]any, map[string]any]()

	//构建工作流 开始节点和结束节点是标志，不走节点的执行逻辑 所以要单独处理
	var startNode *model.Node
	var endNode *model.Node
	//构建边关系的映射 source是进入的边 target是出去的边
	sourceMap := make(map[string][]*model.Edge)
	targetMap := make(map[string][]*model.Edge)

	//这里注意 一个节点是有多个source和target
	for _, edge := range data.Edges {
		sourceMap[edge.Target] = append(sourceMap[edge.Target], edge)
		targetMap[edge.Source] = append(targetMap[edge.Source], edge)
	}
	//存储已添加到工作流的节点引用
	nodeRefs := make(map[string]*compose.WorkflowNode)
	hasStartNode := false
	hasEndNode := false
	for _, node := range data.Nodes {
		if node.Type == string(nodes.Start) {
			startNode = node
			hasStartNode = true
			continue
		}
		if node.Type == string(nodes.End) {
			endNode = node
			hasEndNode = true
			continue
		}
		if _, exists := nodeRefs[node.ID]; !exists {
			//从注册表中获取节点
			nodeFactory, ok := w.nodeRegistry[nodes.NodeType(node.Type)]
			if !ok {
				logs.Error("Failed to find node factory for node type: %s", node.Type)
				return nil, fmt.Errorf("不支持的节点类型: %s", node.Type)
			}
			ref := wf.AddLambdaNode(node.ID, compose.InvokableLambda(nodeFactory(node.Data).Invoke))
			nodeRefs[node.ID] = ref
		}
	}
	if !hasStartNode || startNode == nil {
		logs.Error("Workflow must have a start node")
		return nil, fmt.Errorf("工作流必须包含开始节点")
	}
	if !hasEndNode || endNode == nil {
		logs.Error("Workflow must have an end node")
		return nil, fmt.Errorf("工作流必须包含结束节点")
	}
	edgesMap := make(map[EdgeKey]*EdgeValues)
	for _, edge := range data.Edges {
		if edge.Source == startNode.ID {
			nodeRefs[edge.Target].AddInput(compose.START)
		} else if edge.Target == endNode.ID {
			wf.End().AddInput(edge.Source, compose.MapFields(edge.SourceHandle, edge.TargetHandle))
		} else {
			ek := EdgeKey{
				Source: edge.Source,
				Target: edge.Target,
			}
			if edgesMap[ek] == nil {
				edgesMap[ek] = &EdgeValues{
					SourceHandles: []string{edge.SourceHandle},
					TargetHandles: []string{edge.TargetHandle},
				}
			} else {
				edgesMap[ek].SourceHandles = append(edgesMap[ek].SourceHandles, edge.SourceHandle)
				edgesMap[ek].TargetHandles = append(edgesMap[ek].TargetHandles, edge.TargetHandle)
			}
		}
	}
	for key, value := range edgesMap {
		mappings := make([]*compose.FieldMapping, 0)
		for i, sourceHandle := range value.SourceHandles {
			mappings = append(mappings, compose.MapFields(sourceHandle, value.TargetHandles[i]))
		}
		nodeRefs[key.Target].AddInputWithOptions(key.Source, mappings)
	}
	runner, err := wf.Compile(ctx)
	if err != nil {
		logs.Error("Failed to compile workflow: %v", err)
		return nil, err
	}
	params := make(map[string]any)
	result, err := runner.Invoke(ctx, params)
	if err != nil {
		logs.Error("Failed to execute workflow: %v", err)
		return nil, err
	}
	results := make(map[string]any)
	results["result"] = result
	return results, nil
}
