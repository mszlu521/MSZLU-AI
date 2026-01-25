package nodes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/schema"
	"github.com/mszlu521/thunder/logs"
)

const Prompt = `# 角色
你是一位资深的前端开发专家，精通HTML5、CSS3和现代前端框架，尤其擅长编写语义化、可维护且响应式的代码。

# 任务
根据我提供的这张网页截图，完整地生成一个独立的HTML文件，包含所有必要的HTML结构和CSS样式。

# 技术栈与要求
1.  **HTML**:
    *   使用语义化的HTML5标签（如 <header>, <nav>, <main>, <section>, <article>, <footer> 等）。
    *   确保HTML结构清晰，逻辑性强，易于阅读。
    *   为重要的元素添加适当的alt属性（图片）和aria标签，以保证可访问性（Accessibility）。

2.  **CSS**:
    *   **优先使用Tailwind CSS来实现所有样式**。将Tailwind的CDN链接添加到HTML的<head>中，并直接在HTML标签的class属性中编写原子化CSS。这是首选方案。

3.  **响应式设计**:
    *   代码必须是完全响应式的，能够在桌面、平板和手机屏幕上都有良好的显示效果。
    *   请使用Tailwind CSS的响应式断点（如 md:, lg:）来处理不同屏幕尺寸下的布局和样式。

4.  **资源处理**:
    *   对于截图中的图片，请使用占位图服务（如 https://placehold.co/）并根据截图中的大致尺寸设置宽高。
    *   对于图标，请使用SVG代码，或者从一个流行的图标库（如Heroicons）中引用。
    *   文本内容请直接从截图中提取，如果看不清，可以使用合适的占位符文本（Lorem Ipsum）。

5.  **代码风格**:
    *   代码需要有良好的格式化和缩进。
    *   在关键的HTML部分或复杂的CSS逻辑处添加简短的注释。

# 输出格式
请将所有代码包裹在一个Markdown代码块中，语言类型为html。
`

type QwenVLNode struct {
	data map[string]any
}

func NewQwenVLNode(data map[string]any) *QwenVLNode {
	if data == nil {
		data = make(map[string]any)
	}
	return &QwenVLNode{
		data: data,
	}
}

func (t *QwenVLNode) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	//这个是视觉模型，我们需要先获取到模型的信息，最后调用模型实现内容的生成
	var model string
	if inputModel, exists := input["model"]; exists {
		model = inputModel.(string)
	} else {
		dataModel, exist := t.data["model"]
		if exist {
			modelData, ok := dataModel.(map[string]any)
			if ok {
				model = modelData["fieldValue"].(string)
			}
		} else {
			return nil, fmt.Errorf("no model")
		}
	}
	//获取图片
	var imageUrl string
	if inputImage, exists := input["image"]; exists {
		imageUrl = inputImage.(string)
	} else {
		dataImageUrl, exist := t.data["image"]
		if exist {
			imageUrlData, ok := dataImageUrl.(map[string]any)
			if ok {
				imageUrl = imageUrlData["fieldValue"].(string)
			}
		}
	}
	if imageUrl == "" {
		if inputImageUrl, exists := input["imageUrl"]; exists {
			imageUrl = inputImageUrl.(string)
		} else {
			dataImageUrl, exist := t.data["imageUrl"]
			if exist {
				imageUrlData, ok := dataImageUrl.(map[string]any)
				if ok {
					imageUrl = imageUrlData["fieldValue"].(string)
				}
			}
		}
	}
	//模型提供商
	var provider map[string]any
	if inputProvider, exists := input["providers"]; exists {
		provider = inputProvider.(map[string]any)
	} else {
		dataProvider, exist := t.data["providers"]
		if exist {
			providerData, ok := dataProvider.(map[string]any)
			if ok {
				provider = providerData
			} else {
				return nil, fmt.Errorf("no providers")
			}
		} else {
			return nil, fmt.Errorf("no providers")
		}
	}
	var promptType string
	if inputPromptType, exists := input["promptType"]; exists {
		promptType = inputPromptType.(string)
	} else {
		dataPromptType, exist := t.data["promptType"]
		if exist {
			promptTypeData, ok := dataPromptType.(map[string]any)
			if ok {
				promptType = promptTypeData["fieldValue"].(string)
			}
		}
	}
	//把图片转成base64
	var err error
	imageUrl, err = t.processImage(imageUrl)
	if err != nil {
		logs.Errorf("process image error: %v", err)
		return nil, err
	}
	//获取厂商名字
	providerName, ok := provider["provider"].(string)
	if !ok {
		m, ok := provider["fieldValue"].(map[string]any)
		if ok {
			providerName = m["provider"].(string)
		}
	}
	apiBase, ok := provider["apiBase"].(string)
	if !ok {
		m, ok := provider["fieldValue"].(map[string]any)
		if ok {
			apiBase = m["apiBase"].(string)
		}
	}
	apiKey, ok := provider["apiKey"].(string)
	if !ok {
		m, ok := provider["fieldValue"].(map[string]any)
		if ok {
			apiKey = m["apiKey"].(string)
		}
	}
	var response *schema.Message
	if providerName == "ollama" {
		chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			Model:   model,
			BaseURL: apiBase,
		})
		if err != nil {
			return nil, err
		}
		//提示词
		messages := t.processPrompt(promptType, imageUrl)
		response, err = chatModel.Generate(ctx, messages)
		if err != nil {
			logs.Errorf("ollama generate error: %v", err)
			return nil, err
		}
	} else if providerName == "qwen" {
		chatModel, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
			Model:   model,
			BaseURL: apiBase,
			APIKey:  apiKey,
		})
		if err != nil {
			return nil, err
		}
		messages := t.processPrompt(promptType, imageUrl)
		response, err = chatModel.Generate(ctx, messages)
		if err != nil {
			return nil, err
		}
	} else {
		chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:   model,
			BaseURL: apiBase,
			APIKey:  apiKey,
		})
		if err != nil {
			return nil, err
		}
		messages := t.processPrompt(promptType, imageUrl)
		response, err = chatModel.Generate(ctx, messages)
		if err != nil {
			return nil, err
		}
	}
	if response == nil {
		return nil, fmt.Errorf("no response")
	}
	result := map[string]any{
		"type":   "qwenVL",
		"output": response.Content,
	}
	return result, nil
}

func (t *QwenVLNode) processImage(url string) (string, error) {
	//判断是本地文件还是网络URL
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url, nil
	}
	//本地文件转为base64
	absPaht, err := filepath.Abs(url)
	if err != nil {
		return "", err
	}
	//读取文件
	data, err := ioutil.ReadFile(absPaht)
	if err != nil {
		return "", err
	}
	contentType := http.DetectContentType(data)
	encodeToString := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, encodeToString)
	return dataURL, nil
}

func (t *QwenVLNode) processPrompt(promptType string, imageUrl string) []*schema.Message {
	if promptType == "image" {
		imageMsg := []map[string]string{
			{
				"type":  "image",
				"image": imageUrl,
			},
			{
				"type": "text",
				"text": Prompt,
			},
		}
		imageMsgBytes, _ := json.Marshal(imageMsg)
		messages := []*schema.Message{
			{
				Role:    schema.User,
				Content: string(imageMsgBytes),
			},
		}
		return messages
	}
	return []*schema.Message{
		{
			Role: schema.User,
			MultiContent: []schema.ChatMessagePart{
				{
					Type: schema.ChatMessagePartTypeText,
					Text: Prompt,
				},
				{
					Type: schema.ChatMessagePartTypeImageURL,
					ImageURL: &schema.ChatMessageImageURL{
						URL: imageUrl,
					},
				},
			},
		},
	}
}
