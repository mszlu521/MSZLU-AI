package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var ApiKey = ""

// WeatherTool 天气查询工具 使用高德天气API
type WeatherTool struct {
	apiKey string
}

type WeatherConfig struct {
	ApiKey string
}

func NewWeatherTool(c *WeatherConfig) *WeatherTool {
	if c == nil {
		panic("WeatherConfig is nil")
	}
	return &WeatherTool{apiKey: c.ApiKey}
}

func (w *WeatherTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"city": {
			Desc:     "需要查询天气的城市名称或区域编码",
			Type:     schema.String,
			Required: true,
		},
		"extensions": {
			Desc: "气象类型：base(实况天气)/all(预报天气)",
			Type: schema.String,
			Enum: []string{"base", "all"},
		},
	}
}

func (w *WeatherTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_weather",
		Desc: "查询指定城市的天气信息，使用高德天气API",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {
				Desc:     "需要查询天气的城市名称或区域编码",
				Type:     schema.String,
				Required: true,
			},
			"extensions": {
				Desc: "气象类型：base(实况天气)/all(预报天气)",
				Type: schema.String,
				Enum: []string{"base", "all"},
			},
		}),
	}, nil
}

// InvokableRun 这是执行天气查询的函数
func (w *WeatherTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	//根据高德天气API 发起GET请求 获取查询结果
	//argumentsInJSON 这是传入的参数
	var params map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", err
	}
	//获取我们需要的参数信息
	city, ok := params["city"].(string)
	if !ok {
		return "", fmt.Errorf("city is required")
	}
	//先构建请求参数
	queryParams := url.Values{}
	queryParams.Set("key", w.apiKey)
	queryParams.Set("city", city)
	//可选
	if extensions, ok := params["extensions"].(string); ok {
		queryParams.Set("extensions", extensions)
	} else {
		//给个默认查询值
		queryParams.Set("extensions", "base")
	}
	//返回的格式为json
	queryParams.Set("output", "JSON")
	baseUrl := "https://restapi.amap.com/v3/weather/weatherInfo"
	fullUrl := fmt.Sprintf("%s?%s", baseUrl, queryParams.Encode())
	//发送http请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullUrl, nil)
	if err != nil {
		return "", err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	//读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	//判断响应码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http request failed with status code %d", resp.StatusCode)
	}
	return string(body), nil
}
