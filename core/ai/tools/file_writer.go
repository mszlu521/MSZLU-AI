package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mszlu521/thunder/ai/einos"
)

// FileWriteTool 文件写入工具
type FileWriteTool struct {
	baseDir string
}

type FileWriteConfig struct {
	BaseDir string // 基础目录，为空则允许写入任意路径
}

func NewFileWriteTool(c *FileWriteConfig) einos.InvokeParamTool {
	if c == nil {
		c = &FileWriteConfig{}
	}
	return &FileWriteTool{baseDir: c.BaseDir}
}

func (f *FileWriteTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"file_path": {
			Desc:     "要写入的文件完整路径（包括文件名和扩展名）",
			Type:     schema.String,
			Required: true,
		},
		"content": {
			Desc:     "要写入的文件内容",
			Type:     schema.String,
			Required: true,
		},
		"append": {
			Desc: "是否追加模式写入（true 为追加，false 为覆盖）",
			Type: schema.Boolean,
		},
	}
}

func (f *FileWriteTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_writer",
		Desc: "将指定内容写入到文件中，支持各种文件格式（txt、html、json、md 等），可选择覆盖或追加模式",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Desc:     "要写入的文件完整路径（包括文件名和扩展名）",
				Type:     schema.String,
				Required: true,
			},
			"content": {
				Desc:     "要写入的文件内容",
				Type:     schema.String,
				Required: true,
			},
			"append": {
				Desc: "是否追加模式写入（true 为追加，false 为覆盖）",
				Type: schema.Boolean,
			},
		}),
	}, nil
}

// InvokableRun 执行文件写入操作
func (f *FileWriteTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var params map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("解析参数失败：%w", err)
	}

	// 获取文件路径
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return "", fmt.Errorf("file_path 参数为空")
	}

	// 获取文件内容
	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content 参数为空或类型错误")
	}

	// 如果配置了 baseDir，检查路径合法性
	if f.baseDir != "" {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return "", fmt.Errorf("无效的文件路径：%w", err)
		}

		// 如果是相对路径，转换为相对于 baseDir 的路径
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(f.baseDir, filePath)
		} else {
			// 如果是绝对路径，检查是否在 baseDir 内
			absBaseDir, _ := filepath.Abs(f.baseDir)
			if !strings.HasPrefix(absPath, absBaseDir) {
				return "", fmt.Errorf("文件路径超出允许范围：%s (允许范围：%s)", filePath, absBaseDir)
			}
		}
	}

	// 获取写入模式（默认为覆盖）
	appendMode := false
	if appendVal, ok := params["append"].(bool); ok {
		appendMode = appendVal
	}

	// 创建目录（如果不存在）
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败：%w", err)
	}

	// 选择写入模式
	var file *os.File
	var err error

	if appendMode {
		// 追加模式
		file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		// 覆盖模式
		file, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	}

	if err != nil {
		return "", fmt.Errorf("打开文件失败：%w", err)
	}
	defer file.Close()

	// 写入内容
	_, err = file.WriteString(content)
	if err != nil {
		return "", fmt.Errorf("写入文件失败：%w", err)
	}

	// 返回成功信息
	result := map[string]string{
		"status":      "success",
		"message":     fmt.Sprintf("文件写入成功：%s", filePath),
		"file_path":   filePath,
		"write_mode":  "overwrite",
		"content_len": fmt.Sprintf("%d bytes", len(content)),
	}

	if appendMode {
		result["write_mode"] = "append"
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}
