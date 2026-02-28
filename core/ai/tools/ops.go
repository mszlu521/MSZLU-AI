package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mszlu521/thunder/ai/einos"
	"github.com/mszlu521/thunder/logs"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

//这是用于k8s运维相关的工具集合

type K8sClient struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

var k8sClient *K8sClient

func InitK8sClient() error {
	//优先使用集群内的配置
	config, err := rest.InClusterConfig()
	if err != nil {
		//如果报错，则使用本地的kubeconfig
		home := homedir.HomeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")
		//检查环境变量
		if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
			kubeconfig = envConfig
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logs.Errorf("无法创建k8s配置: %v", err)
			return err
		}
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logs.Errorf("无法创建k8s客户端: %v", err)
		return err
	}
	k8sClient = &K8sClient{
		clientset: clientset,
		config:    config,
	}
	return nil
}

func GetK8sClient() *K8sClient {
	return k8sClient
}

//==============接下来创建k8s工具 用于操作k8s============

// K8sResourceQueryTool 这是k8s资源查询工具
type K8sResourceQueryTool struct {
	client *K8sClient
}

func (k *K8sResourceQueryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "k8s_resource_query",
		Desc:        "查询Kubernetes集群中的资源信息，包括Pod、Deployment、Service、Node等",
		ParamsOneOf: schema.NewParamsOneOfByParams(k.Params()),
	}, nil
}

type K8sResourceQueryArgs struct {
	ResourceType  string `json:"resource_type"`
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	LabelSelector string `json:"label_selector"`
}

func (k *K8sResourceQueryTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if k.client == nil {
		return "", fmt.Errorf("k8s客户端未初始化")
	}
	var args K8sResourceQueryArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}
	result, err := k.queryResource(ctx, args)
	if err != nil {
		return "", fmt.Errorf("查询资源失败: %w", err)
	}
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return string(jsonResult), nil
}

func (k *K8sResourceQueryTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"resource_type": {
			Type:     schema.String,
			Required: true,
			Desc:     "Kubernetes资源类型，支持: pod, deployment, service, node, configmap, secret, replicaset, statefulset, daemonset",
		},
		"namespace": {
			Type:     schema.String,
			Required: false,
			Desc:     "命令空间，默认是default",
		},
		"name": {
			Type:     schema.String,
			Required: false,
			Desc:     "资源名称,为空则查询该类型的所有资源",
		},
		"label_selector": {
			Type:     schema.String,
			Required: false,
			Desc:     "标签选择器, 例如: app=myapp,env=prod",
		},
	}
}

func (k *K8sResourceQueryTool) queryResource(ctx context.Context, args K8sResourceQueryArgs) (interface{}, error) {
	clientset := k.client.clientset
	switch strings.ToLower(args.ResourceType) {
	case "pod", "pods":
		if args.Name != "" {
			pod, err := clientset.CoreV1().Pods(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("无法获取Pod: %w", err)
			}
			return formatPodInfo(pod), nil
		}
		listOptions := metav1.ListOptions{}
		if args.LabelSelector != "" {
			listOptions.LabelSelector = args.LabelSelector
		}
		pods, err := clientset.CoreV1().Pods(args.Namespace).List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("无法获取Pod列表: %w", err)
		}
		var podInfos []map[string]any
		for _, pod := range pods.Items {
			podInfos = append(podInfos, formatPodInfo(&pod))
		}
		return podInfos, nil
	case "deployment", "deployments":
		if args.Name != "" {
			deployment, err := clientset.AppsV1().Deployments(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("无法获取Deployment: %w", err)
			}
			return formatDeploymentInfo(deployment), nil
		}
		listOptions := metav1.ListOptions{}
		if args.LabelSelector != "" {
			listOptions.LabelSelector = args.LabelSelector
		}
		deployments, err := clientset.AppsV1().Deployments(args.Namespace).List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("无法获取Deployment列表: %w", err)
		}
		var deploymentInfos []map[string]any
		for _, deployment := range deployments.Items {
			deploymentInfos = append(deploymentInfos, formatDeploymentInfo(&deployment))
		}
		return deploymentInfos, nil
	case "service", "services":
		if args.Name != "" {
			service, err := clientset.CoreV1().Services(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("无法获取Service: %w", err)
			}
			return formatServiceInfo(service), nil
		}
		listOptions := metav1.ListOptions{}
		if args.LabelSelector != "" {
			listOptions.LabelSelector = args.LabelSelector
		}
		services, err := clientset.CoreV1().Services(args.Namespace).List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("无法获取Service列表: %w", err)
		}
		var serviceInfos []map[string]any
		for _, service := range services.Items {
			serviceInfos = append(serviceInfos, formatServiceInfo(&service))
		}
		return serviceInfos, nil
	case "node", "nodes":
		if args.Name != "" {
			node, err := clientset.CoreV1().Nodes().Get(ctx, args.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("无法获取Node: %w", err)
			}
			return formatNodeInfo(node), nil
		}
		listOptions := metav1.ListOptions{}
		if args.LabelSelector != "" {
			listOptions.LabelSelector = args.LabelSelector
		}
		nodes, err := clientset.CoreV1().Nodes().List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("无法获取Node列表: %w", err)
		}
		var nodeInfos []map[string]any
		for _, node := range nodes.Items {
			nodeInfos = append(nodeInfos, formatNodeInfo(&node))
		}
		return nodeInfos, nil
	case "configmap", "configmaps":
		if args.Name != "" {
			configMap, err := clientset.CoreV1().ConfigMaps(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("无法获取ConfigMap: %w", err)
			}
			return formatConfigMapInfo(configMap), nil
		}
		listOptions := metav1.ListOptions{}
		if args.LabelSelector != "" {
			listOptions.LabelSelector = args.LabelSelector
		}
		configMaps, err := clientset.CoreV1().ConfigMaps(args.Namespace).List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("无法获取ConfigMap列表: %w", err)
		}
		var configMapInfos []map[string]any
		for _, configMap := range configMaps.Items {
			configMapInfos = append(configMapInfos, formatConfigMapInfo(&configMap))
		}
		return configMapInfos, nil
	case "secret", "secrets":
		if args.Name != "" {
			secret, err := clientset.CoreV1().Secrets(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("无法获取Secret: %w", err)
			}
			return formatSecretInfo(secret), nil
		}
		listOptions := metav1.ListOptions{}
		if args.LabelSelector != "" {
			listOptions.LabelSelector = args.LabelSelector
		}
		secrets, err := clientset.CoreV1().Secrets(args.Namespace).List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("无法获取Secret列表: %w", err)
		}
		var secretInfos []map[string]any
		for _, secret := range secrets.Items {
			secretInfos = append(secretInfos, formatSecretInfo(&secret))
		}
		return secretInfos, nil
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", args.ResourceType)
	}

}

func formatSecretInfo(secret *corev1.Secret) map[string]any {
	return map[string]any{
		"name":        secret.Name,
		"namespace":   secret.Namespace,
		"type":        secret.Type,
		"data_keys":   getMapkeysByte(secret.Data),
		"string_keys": getMapkeys(secret.StringData),
		"labels":      secret.Labels,
		"age":         time.Since(secret.CreationTimestamp.Time).Round(time.Second).String(),
	}
}

func getMapkeysByte(data map[string][]byte) []string {
	if data == nil {
		return nil
	}
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

func formatConfigMapInfo(configMap *corev1.ConfigMap) map[string]any {
	return map[string]any{
		"name":      configMap.Name,
		"namespace": configMap.Namespace,
		"data_keys": getMapkeys(configMap.Data),
		"labels":    configMap.Labels,
		"age":       time.Since(configMap.CreationTimestamp.Time).Round(time.Second).String(),
	}
}

func getMapkeys(data map[string]string) []string {
	if data == nil {
		return nil
	}
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

func formatNodeInfo(node *corev1.Node) map[string]any {
	return map[string]any{
		"name":      node.Name,
		"namespace": node.Namespace,
		"labels":    node.Labels,
		"age":       time.Since(node.CreationTimestamp.Time).Round(time.Second).String(),
	}
}

func formatServiceInfo(service *corev1.Service) map[string]any {
	return map[string]any{
		"name":       service.Name,
		"namespace":  service.Namespace,
		"type":       service.Spec.Type,
		"cluster_ip": service.Spec.ClusterIP,
		"ports":      service.Spec.Ports,
		"labels":     service.Labels,
		"age":        time.Since(service.CreationTimestamp.Time).Round(time.Second).String(),
	}
}

func formatDeploymentInfo(deployment *appsv1.Deployment) map[string]any {
	return map[string]any{
		"name":               deployment.Name,
		"namespace":          deployment.Namespace,
		"replicas":           deployment.Status.Replicas,
		"available_replicas": deployment.Status.AvailableReplicas,
		"ready_replicas":     deployment.Status.ReadyReplicas,
		"labels":             deployment.Labels,
		"age":                time.Since(deployment.CreationTimestamp.Time).Round(time.Second).String(),
	}
}

func formatPodInfo(pod *corev1.Pod) map[string]any {
	status := string(pod.Status.Phase)
	if pod.DeletionTimestamp != nil {
		status = "Terminating"
	}
	return map[string]any{
		"name":          pod.Name,
		"namespace":     pod.Namespace,
		"status":        status,
		"node":          pod.Spec.NodeName,
		"labels":        pod.Labels,
		"ip":            pod.Status.PodIP,
		"restart_count": getPodRestartCount(pod),
		"age":           time.Since(pod.CreationTimestamp.Time).Round(time.Second).String(),
	}
}

func getPodRestartCount(pod *corev1.Pod) any {
	var restartCount int32
	for _, status := range pod.Status.ContainerStatuses {
		restartCount += status.RestartCount
	}
	return restartCount
}

func NewK8sResourceQueryTool() einos.InvokeParamTool {
	return &K8sResourceQueryTool{
		client: k8sClient,
	}
}

// ==================== 日志查询工具 ====================

// K8sLogsTool K8s日志查询工具
type K8sLogsTool struct {
	client *K8sClient
}

// NewK8sLogsTool 创建K8s日志查询工具
func NewK8sLogsTool() einos.InvokeParamTool {
	return &K8sLogsTool{
		client: k8sClient,
	}
}

// K8sLogsArgs 日志查询参数
type K8sLogsArgs struct {
	Namespace     string `json:"namespace"`      // 命名空间
	PodName       string `json:"pod_name"`       // Pod名称
	ContainerName string `json:"container_name"` // 容器名称（Pod多容器时需要）
	TailLines     int64  `json:"tail_lines"`     // 返回最后多少行日志，默认100
	Previous      bool   `json:"previous"`       // 是否查看上次容器的日志
	SinceSeconds  int64  `json:"since_seconds"`  // 查看最近多少秒的日志
}

func (k *K8sLogsTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"namespace": {
			Type:     schema.String,
			Desc:     "Pod所在的命名空间",
			Required: true,
		},
		"pod_name": {
			Type:     schema.String,
			Desc:     "Pod名称",
			Required: true,
		},
		"container_name": {
			Type:     schema.String,
			Desc:     "容器名称，当Pod包含多个容器时需要指定",
			Required: false,
		},
		"tail_lines": {
			Type:     schema.Integer,
			Desc:     "返回最后多少行日志，默认100行",
			Required: false,
		},
		"previous": {
			Type:     schema.Boolean,
			Desc:     "是否查看上次崩溃容器的日志",
			Required: false,
		},
		"since_seconds": {
			Type:     schema.Integer,
			Desc:     "查看最近多少秒的日志",
			Required: false,
		},
	}
}

func (k *K8sLogsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "k8s_logs_query",
		Desc:        "查询Kubernetes Pod的日志，支持查看容器日志、历史日志和按时间范围筛选",
		ParamsOneOf: schema.NewParamsOneOfByParams(k.Params()),
	}, nil
}

func (k *K8sLogsTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if k.client == nil {
		return "", fmt.Errorf("K8s客户端未初始化")
	}

	var args K8sLogsArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	if args.TailLines == 0 {
		args.TailLines = 100
	}

	logOptions := &corev1.PodLogOptions{
		TailLines: &args.TailLines,
		Previous:  args.Previous,
	}

	if args.ContainerName != "" {
		logOptions.Container = args.ContainerName
	}

	if args.SinceSeconds > 0 {
		logOptions.SinceSeconds = &args.SinceSeconds
	}

	req := k.client.clientset.CoreV1().Pods(args.Namespace).GetLogs(args.PodName, logOptions)
	logsData, err := req.Do(ctx).Raw()
	if err != nil {
		return fmt.Sprintf("获取日志失败: %v", err), nil
	}

	return string(logsData), nil
}

// ==================== 资源操作工具 ====================

// K8sResourceActionTool K8s资源操作工具
type K8sResourceActionTool struct {
	client *K8sClient
}

// NewK8sResourceActionTool 创建K8s资源操作工具
func NewK8sResourceActionTool() einos.InvokeParamTool {
	return &K8sResourceActionTool{
		client: k8sClient,
	}
}

// K8sResourceActionArgs 资源操作参数
type K8sResourceActionArgs struct {
	Action       string `json:"action"`        // 操作: restart, scale, delete
	ResourceType string `json:"resource_type"` // 资源类型: deployment, pod
	Namespace    string `json:"namespace"`     // 命名空间
	Name         string `json:"name"`          // 资源名称
	Replicas     int32  `json:"replicas"`      // 副本数（scale操作使用）
}

func (k *K8sResourceActionTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"action": {
			Type:     schema.String,
			Desc:     "操作类型: restart(重启), scale(扩缩容), delete(删除)",
			Required: true,
		},
		"resource_type": {
			Type:     schema.String,
			Desc:     "资源类型: deployment, pod",
			Required: true,
		},
		"namespace": {
			Type:     schema.String,
			Desc:     "命名空间",
			Required: true,
		},
		"name": {
			Type:     schema.String,
			Desc:     "资源名称",
			Required: true,
		},
		"replicas": {
			Type:     schema.Integer,
			Desc:     "副本数，仅scale操作需要",
			Required: false,
		},
	}
}

func (k *K8sResourceActionTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "k8s_resource_action",
		Desc:        "操作Kubernetes资源，支持重启Deployment、扩缩容、删除Pod等操作",
		ParamsOneOf: schema.NewParamsOneOfByParams(k.Params()),
	}, nil
}

func (k *K8sResourceActionTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if k.client == nil {
		return "", fmt.Errorf("K8s客户端未初始化")
	}

	var args K8sResourceActionArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	switch strings.ToLower(args.Action) {
	case "restart":
		return k.restartDeployment(ctx, args)
	case "scale":
		return k.scaleDeployment(ctx, args)
	case "delete":
		return k.deleteResource(ctx, args)
	default:
		return "", fmt.Errorf("不支持的操作: %s", args.Action)
	}
}

func (k *K8sResourceActionTool) restartDeployment(ctx context.Context, args K8sResourceActionArgs) (string, error) {
	if strings.ToLower(args.ResourceType) != "deployment" {
		return "", fmt.Errorf("restart操作仅支持deployment类型")
	}

	// 通过更新annotation触发重启
	deployment, err := k.client.clientset.AppsV1().Deployments(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("获取Deployment失败: %w", err)
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = k.client.clientset.AppsV1().Deployments(args.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return "", fmt.Errorf("重启Deployment失败: %w", err)
	}

	return fmt.Sprintf("成功触发Deployment %s/%s 的重启", args.Namespace, args.Name), nil
}

func (k *K8sResourceActionTool) scaleDeployment(ctx context.Context, args K8sResourceActionArgs) (string, error) {
	if strings.ToLower(args.ResourceType) != "deployment" {
		return "", fmt.Errorf("scale操作仅支持deployment类型")
	}

	scale, err := k.client.clientset.AppsV1().Deployments(args.Namespace).GetScale(ctx, args.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("获取Deployment Scale失败: %w", err)
	}

	scale.Spec.Replicas = args.Replicas
	_, err = k.client.clientset.AppsV1().Deployments(args.Namespace).UpdateScale(ctx, args.Name, scale, metav1.UpdateOptions{})
	if err != nil {
		return "", fmt.Errorf("扩缩容失败: %w", err)
	}

	return fmt.Sprintf("成功将Deployment %s/%s 调整为 %d 个副本", args.Namespace, args.Name, args.Replicas), nil
}

func (k *K8sResourceActionTool) deleteResource(ctx context.Context, args K8sResourceActionArgs) (string, error) {
	var err error

	switch strings.ToLower(args.ResourceType) {
	case "pod":
		err = k.client.clientset.CoreV1().Pods(args.Namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "deployment":
		err = k.client.clientset.AppsV1().Deployments(args.Namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	default:
		return "", fmt.Errorf("delete操作仅支持pod或deployment类型")
	}

	if err != nil {
		return "", fmt.Errorf("删除资源失败: %w", err)
	}

	return fmt.Sprintf("成功删除 %s %s/%s", args.ResourceType, args.Namespace, args.Name), nil
}

// ==================== 集群健康检查工具 ====================

// K8sHealthCheckTool K8s健康检查工具
type K8sHealthCheckTool struct {
	client *K8sClient
}

// NewK8sHealthCheckTool 创建K8s健康检查工具
func NewK8sHealthCheckTool() einos.InvokeParamTool {
	return &K8sHealthCheckTool{
		client: k8sClient,
	}
}

func (k *K8sHealthCheckTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"namespace": {
			Type:     schema.String,
			Desc:     "检查的命名空间，为空则检查所有命名空间",
			Required: false,
		},
		"check_type": {
			Type:     schema.String,
			Desc:     "检查类型: all, pods, nodes, events",
			Required: false,
		},
	}
}

func (k *K8sHealthCheckTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "k8s_health_check",
		Desc:        "检查Kubernetes集群健康状态，包括节点状态、Pod状态、异常事件等",
		ParamsOneOf: schema.NewParamsOneOfByParams(k.Params()),
	}, nil
}

func (k *K8sHealthCheckTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if k.client == nil {
		return "", fmt.Errorf("K8s客户端未初始化")
	}

	var args struct {
		Namespace string `json:"namespace"`
		CheckType string `json:"check_type"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	if args.CheckType == "" {
		args.CheckType = "all"
	}

	result := make(map[string]interface{})

	switch args.CheckType {
	case "nodes":
		nodesHealth, err := k.checkNodes(ctx)
		if err != nil {
			result["nodes_error"] = err.Error()
		} else {
			result["nodes"] = nodesHealth
		}
	case "pods":
		podsHealth, err := k.checkPods(ctx, args.Namespace)
		if err != nil {
			result["pods_error"] = err.Error()
		} else {
			result["pods"] = podsHealth
		}
	case "events":
		eventsHealth, err := k.checkEvents(ctx, args.Namespace)
		if err != nil {
			result["events_error"] = err.Error()
		} else {
			result["events"] = eventsHealth
		}
	case "all":
		nodesHealth, err := k.checkNodes(ctx)
		if err != nil {
			result["nodes_error"] = err.Error()
		} else {
			result["nodes"] = nodesHealth
		}

		podsHealth, err := k.checkPods(ctx, args.Namespace)
		if err != nil {
			result["pods_error"] = err.Error()
		} else {
			result["pods"] = podsHealth
		}

		eventsHealth, err := k.checkEvents(ctx, args.Namespace)
		if err != nil {
			result["events_error"] = err.Error()
		} else {
			result["events"] = eventsHealth
		}
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return string(jsonResult), nil
}

func (k *K8sHealthCheckTool) checkNodes(ctx context.Context) (map[string]interface{}, error) {
	nodes, err := k.client.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"total": len(nodes.Items),
	}

	var readyNodes, notReadyNodes int
	var nodeDetails []map[string]interface{}

	for _, node := range nodes.Items {
		nodeInfo := map[string]interface{}{
			"name":   node.Name,
			"labels": node.Labels,
		}

		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status == corev1.ConditionTrue {
					readyNodes++
					nodeInfo["status"] = "Ready"
				} else {
					notReadyNodes++
					nodeInfo["status"] = "NotReady"
					nodeInfo["reason"] = condition.Reason
					nodeInfo["message"] = condition.Message
				}
			}
		}
		nodeDetails = append(nodeDetails, nodeInfo)
	}

	result["ready"] = readyNodes
	result["not_ready"] = notReadyNodes
	result["details"] = nodeDetails

	return result, nil
}

func (k *K8sHealthCheckTool) checkPods(ctx context.Context, namespace string) (map[string]interface{}, error) {
	listOptions := metav1.ListOptions{}

	pods, err := k.client.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"total": len(pods.Items),
	}

	var running, pending, failed, succeeded, unknown int
	var failedPods []map[string]interface{}

	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			running++
		case corev1.PodPending:
			pending++
		case corev1.PodFailed:
			failed++
			failedPods = append(failedPods, formatPodInfo(&pod))
		case corev1.PodSucceeded:
			succeeded++
		default:
			unknown++
		}
	}

	result["running"] = running
	result["pending"] = pending
	result["failed"] = failed
	result["succeeded"] = succeeded
	result["unknown"] = unknown
	result["failed_pods"] = failedPods

	return result, nil
}

func (k *K8sHealthCheckTool) checkEvents(ctx context.Context, namespace string) (map[string]interface{}, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "type=Warning",
	}

	if namespace != "" {
		events, err := k.client.clientset.CoreV1().Events(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}

		var warningEvents []map[string]interface{}
		for _, event := range events.Items {
			warningEvents = append(warningEvents, map[string]interface{}{
				"type":       event.Type,
				"reason":     event.Reason,
				"message":    event.Message,
				"count":      event.Count,
				"object":     fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
				"first_seen": event.FirstTimestamp.Format(time.RFC3339),
				"last_seen":  event.LastTimestamp.Format(time.RFC3339),
			})
		}

		return map[string]interface{}{
			"warning_count": len(warningEvents),
			"warnings":      warningEvents,
		}, nil
	}

	// 获取所有命名空间的事件
	allNamespaces, err := k.client.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var allWarningEvents []map[string]interface{}
	for _, ns := range allNamespaces.Items {
		events, err := k.client.clientset.CoreV1().Events(ns.Name).List(ctx, listOptions)
		if err != nil {
			continue
		}
		for _, event := range events.Items {
			allWarningEvents = append(allWarningEvents, map[string]interface{}{
				"namespace": event.Namespace,
				"type":      event.Type,
				"reason":    event.Reason,
				"message":   event.Message,
				"count":     event.Count,
				"object":    fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			})
		}
	}

	return map[string]interface{}{
		"warning_count": len(allWarningEvents),
		"warnings":      allWarningEvents,
	}, nil
}
