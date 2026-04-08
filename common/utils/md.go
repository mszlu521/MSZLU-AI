package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// SkillMetadata 技能元数据
type SkillMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

// ParseSkillMd 解析 skill.md 文件内容，提取元数据
func ParseSkillMd(content string) *SkillMetadata {
	metadata := &SkillMetadata{
		Name:        "",
		Description: "",
		Version:     "1.0.0",
		Author:      "",
		Tags:        []string{},
	}

	// 解析 YAML frontmatter (--- ... ---)
	frontmatterMatch := regexp.MustCompile(`^---\s*\n([\s\S]*?)\n---\s*\n`).FindStringSubmatch(content)
	if frontmatterMatch != nil && len(frontmatterMatch) > 1 {
		frontmatter := frontmatterMatch[1]
		lines := strings.Split(frontmatter, "\n")
		for _, line := range lines {
			match := regexp.MustCompile(`^([\w-]+):\s*(.*)$`).FindStringSubmatch(line)
			if match != nil && len(match) > 2 {
				key := strings.TrimSpace(match[1])
				value := strings.TrimSpace(match[2])
				// 去除引号
				value = strings.Trim(value, "\"'")

				switch key {
				case "name":
					metadata.Name = value
				case "description":
					metadata.Description = value
				case "version":
					metadata.Version = value
				case "author":
					metadata.Author = value
				case "tags":
					// 解析 tags 数组（逗号分隔）
					tags := strings.Split(value, ",")
					for _, tag := range tags {
						tag = strings.TrimSpace(tag)
						if tag != "" {
							metadata.Tags = append(metadata.Tags, tag)
						}
					}
				}
			}
		}
	}

	// 如果没有 name，尝试从 Markdown 标题提取
	if metadata.Name == "" {
		titleMatch := regexp.MustCompile(`(?m)^#\s+(.+)$`).FindStringSubmatch(content)
		if titleMatch != nil && len(titleMatch) > 1 {
			metadata.Name = strings.TrimSpace(titleMatch[1])
		}
	}

	// 如果没有 description，尝试从第一段提取
	if metadata.Description == "" {
		// 先去除 YAML frontmatter
		contentWithoutFrontmatter := content
		if frontmatterMatch != nil {
			contentWithoutFrontmatter = content[len(frontmatterMatch[0]):]
		}
		// 按行分割，找到第一个非空且不是标题的行
		lines := strings.Split(contentWithoutFrontmatter, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				if len(trimmed) > 200 {
					trimmed = trimmed[:200]
				}
				metadata.Description = trimmed
				break
			}
		}
	}

	return metadata
}

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
