package utils

import (
	"fmt"
	"regexp"
	"strings"
)

func ExtractTitle(content string, mark string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s+(.*)`, mark))
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func SplitByHeading(content string, mark string) []string {
	//正则匹配
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s+`, mark))
	//获取所有匹配项的起始索引
	indices := re.FindAllStringIndex(content, -1)
	if len(indices) == 0 {
		return []string{content}
	}
	var chunks []string
	//处理第一个标题出现之前的内容
	if indices[0][0] > 0 {
		preHeaderContent := strings.TrimSpace(content[:indices[0][0]])
		if preHeaderContent != "" {
			chunks = append(chunks, preHeaderContent)
		}
	}
	//遍历索引 按区间划分内容
	for i := 0; i < len(indices); i++ {
		start, end := indices[i][0], len(content)
		//如果后面还有标题，当前块的终点是下一个标题的起始索引
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		chunks = append(chunks, strings.TrimSpace(content[start:end]))
	}
	return chunks
}

func SplitTextByLength(content string, limit int, overlap int) []string {
	if len(content) <= limit {
		return []string{content}
	}
	return SplitByWindow(content, limit, overlap)
}
func SplitByWindow(content string, maxSize int, overlap int) []string {
	var chunks []string
	runes := []rune(content)
	if len(runes) <= maxSize {
		return []string{content}
	}
	step := maxSize - overlap
	for i := 0; i < len(runes); i += step {
		end := i + maxSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}
