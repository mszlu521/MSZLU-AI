package utils

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

var (
	tkm     *tiktoken.Tiktoken
	tkmOnce sync.Once
)

// GetTokenCount 计算文本的 Token 数
func GetTokenCount(text string) int {
	tkmOnce.Do(func() {
		// 生产环境建议指定编码方式，cl100k_base 是目前主流模型最常用的
		encoding := "cl100k_base"
		// 默认编码
		tke, err := tiktoken.GetEncoding(encoding)
		if err != nil {
			return
		}
		tkm = tke
	})

	if tkm == nil {
		// 极端情况下的兜底：如果是中文，约 0.6 字符一个 token；英文约 4 字符一个 token。
		// 这里采用保守估计：字符数 / 1.5
		return len([]rune(text))
	}
	// 真正的分词计算
	tokens := tkm.Encode(text, nil, nil)
	return len(tokens)
}
