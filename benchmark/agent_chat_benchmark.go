package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Config 压测配置
type Config struct {
	BaseURL     string        // 基础URL
	Token       string        // JWT Token
	AgentID     string        // Agent ID
	Message     string        // 测试消息
	Concurrency int           // 并发数
	Duration    time.Duration // 压测时长
	Timeout     time.Duration // 请求超时时间
}

// Stats 统计数据
type Stats struct {
	TotalRequests     int64         // 总请求数
	SuccessRequests   int64         // 成功请求数
	FailedRequests    int64         // 失败请求数
	TotalDuration     time.Duration // 总耗时
	MinDuration       time.Duration // 最小耗时
	MaxDuration       time.Duration // 最大耗时
	TotalResponseSize int64         // 总响应大小(bytes)
	mu                sync.RWMutex
}

// RequestResult 单次请求结果
type RequestResult struct {
	Success      bool
	Duration     time.Duration
	Error        string
	ResponseSize int64
}

func main() {
	config := parseFlags()

	fmt.Println("========================================")
	fmt.Println("Agent Chat 接口压测工具")
	fmt.Println("========================================")
	fmt.Printf("目标URL: %s\n", config.BaseURL+"/api/v1/agents/chat")
	fmt.Printf("并发数: %d\n", config.Concurrency)
	fmt.Printf("压测时长: %s\n", config.Duration)
	fmt.Printf("请求超时: %s\n", config.Timeout)
	fmt.Println("========================================")

	stats := runBenchmark(config)
	printResults(stats, config)
}

// eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiJhZWMxMGIyOC1iYjIzLTQ1ZDMtYTY5Zi0yNmQ2YTM2NWEwYTAiLCJ1c2VybmFtZSI6ImFkbWluIiwiZXhwIjoxNzcxODc2MzAyLCJpYXQiOjE3NzE4NTExMDJ9.SUrPkqR1cT0gX1XBZjywi2AVO0agvucYEsDGd0lR240
func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.BaseURL, "url", "http://localhost:8080", "基础URL")
	flag.StringVar(&config.Token, "token", "", "JWT Token (必需)")
	flag.StringVar(&config.AgentID, "agent-id", "", "Agent ID (必需)")
	flag.StringVar(&config.Message, "message", "你好，请介绍一下自己", "测试消息内容")
	flag.IntVar(&config.Concurrency, "c", 10, "并发数")
	flag.DurationVar(&config.Duration, "d", 30*time.Second, "压测时长")
	flag.DurationVar(&config.Timeout, "timeout", 60*time.Second, "单个请求超时时间")

	flag.Parse()

	// 验证必需参数
	if config.Token == "" {
		fmt.Println("错误: 请提供 JWT Token (-token)")
		flag.Usage()
		os.Exit(1)
	}
	if config.AgentID == "" {
		fmt.Println("错误: 请提供 Agent ID (-agent-id)")
		flag.Usage()
		os.Exit(1)
	}

	return config
}

func runBenchmark(config *Config) *Stats {
	stats := &Stats{
		MinDuration: time.Hour, // 初始化为一个很大的值
	}

	// 创建进度控制
	ctx, cancel := make(chan struct{}), func() {}
	closeChan := make(chan struct{})
	go func() {
		time.Sleep(config.Duration)
		close(closeChan)
	}()

	// 创建工作池
	var wg sync.WaitGroup
	resultChan := make(chan RequestResult, config.Concurrency*10)

	// 启动结果收集器
	go collectResults(stats, resultChan)

	// 启动工作协程
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go worker(config, &wg, resultChan, closeChan)
	}

	// 启动进度报告
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				printProgress(stats)
			case <-closeChan:
				ticker.Stop()
				return
			}
		}
	}()

	_ = ctx
	_ = cancel

	// 等待压测完成
	wg.Wait()
	close(resultChan)

	// 等待结果收集完成
	time.Sleep(100 * time.Millisecond)

	return stats
}

func worker(config *Config, wg *sync.WaitGroup, resultChan chan<- RequestResult, stopChan <-chan struct{}) {
	defer wg.Done()

	client := &http.Client{
		Timeout: config.Timeout,
	}

	for {
		select {
		case <-stopChan:
			return
		default:
			result := doRequest(client, config)
			resultChan <- result

			// 检查是否需要停止
			select {
			case <-stopChan:
				return
			default:
			}
		}
	}
}

func doRequest(client *http.Client, config *Config) RequestResult {
	start := time.Now()
	result := RequestResult{}

	// 构建请求体
	reqBody := map[string]interface{}{
		"agentId": config.AgentID,
		"message": config.Message,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		result.Error = fmt.Sprintf("marshal error: %v", err)
		return result
	}

	// 创建请求
	url := config.BaseURL + "/api/v1/agents/chat"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Sprintf("create request error: %v", err)
		return result
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.Token)
	req.Header.Set("Accept", "text/event-stream")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request error: %v", err)
		return result
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Sprintf("status code: %d, body: %s", resp.StatusCode, string(body))
		return result
	}

	// 读取 SSE 流
	var responseSize int64
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				result.Error = fmt.Sprintf("read stream error: %v", err)
				return result
			}
			break
		}
		responseSize += int64(len(line))

		// 检查是否收到 [DONE] 标记
		if bytes.Contains([]byte(line), []byte("[DONE]")) {
			break
		}
	}

	result.Success = true
	result.Duration = time.Since(start)
	result.ResponseSize = responseSize

	return result
}

func collectResults(stats *Stats, resultChan <-chan RequestResult) {
	for result := range resultChan {
		atomic.AddInt64(&stats.TotalRequests, 1)
		atomic.AddInt64(&stats.TotalResponseSize, result.ResponseSize)

		if result.Success {
			atomic.AddInt64(&stats.SuccessRequests, 1)
			stats.mu.Lock()
			stats.TotalDuration += result.Duration
			if result.Duration < stats.MinDuration {
				stats.MinDuration = result.Duration
			}
			if result.Duration > stats.MaxDuration {
				stats.MaxDuration = result.Duration
			}
			stats.mu.Unlock()
		} else {
			atomic.AddInt64(&stats.FailedRequests, 1)
		}
	}
}

func printProgress(stats *Stats) {
	total := atomic.LoadInt64(&stats.TotalRequests)
	success := atomic.LoadInt64(&stats.SuccessRequests)
	failed := atomic.LoadInt64(&stats.FailedRequests)

	fmt.Printf("[进度] 总请求: %d | 成功: %d | 失败: %d | 成功率: %.2f%%\n",
		total, success, failed, float64(success)/float64(total)*100)
}

func printResults(stats *Stats, config *Config) {
	total := atomic.LoadInt64(&stats.TotalRequests)
	success := atomic.LoadInt64(&stats.SuccessRequests)
	failed := atomic.LoadInt64(&stats.FailedRequests)
	totalSize := atomic.LoadInt64(&stats.TotalResponseSize)

	fmt.Println("\n========================================")
	fmt.Println("压测结果")
	fmt.Println("========================================")
	fmt.Printf("总请求数:       %d\n", total)
	fmt.Printf("成功请求:       %d\n", success)
	fmt.Printf("失败请求:       %d\n", failed)
	fmt.Printf("成功率:         %.2f%%\n", float64(success)/float64(total)*100)

	if total > 0 {
		// QPS 计算
		qps := float64(total) / config.Duration.Seconds()
		fmt.Printf("QPS:            %.2f\n", qps)

		// 吞吐量 (KB/s)
		throughput := float64(totalSize) / 1024 / config.Duration.Seconds()
		fmt.Printf("吞吐量:         %.2f KB/s\n", throughput)
	}

	if success > 0 {
		stats.mu.RLock()
		avgDuration := stats.TotalDuration / time.Duration(success)
		minDuration := stats.MinDuration
		maxDuration := stats.MaxDuration
		stats.mu.RUnlock()

		fmt.Println("\n--- 响应时间统计 ---")
		fmt.Printf("平均响应时间:   %s\n", avgDuration)
		fmt.Printf("最小响应时间:   %s\n", minDuration)
		fmt.Printf("最大响应时间:   %s\n", maxDuration)

		// 估算 TTFB (Time To First Byte) - 流式接口的特殊指标
		fmt.Printf("\n流式接口说明:\n")
		fmt.Printf("  - 响应时间表示完整接收所有流数据的时间\n")
		fmt.Printf("  - 由于是 SSE 流式接口，实际用户体验会更流畅\n")
	}

	fmt.Println("========================================")
}
