package nodes

import (
	"context"
	"strconv"
)

type TextDisplayNode struct {
	data map[string]any
}

func NewTextDisplayNode(data map[string]any) *TextDisplayNode {
	if data == nil {
		data = make(map[string]any)
	}
	return &TextDisplayNode{
		data: data,
	}
}

func (t *TextDisplayNode) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	title, ok := input["title"]
	if !ok {
		title = t.data["title"].(map[string]any)["fieldValue"].(string)
	} else {
		title = title.(string)
	}
	content, ok := input["content"]
	if !ok {
		content = t.data["content"].(map[string]any)["fieldValue"].(string)
	} else {
		content = content.(string)
	}
	renderMarkdown, ok := input["renderMarkdown"]
	if !ok {
		switch v := t.data["content"].(map[string]any)["fieldValue"].(type) {
		case string:
			renderMarkdown, _ = strconv.ParseBool(v)
		case bool:
			renderMarkdown = v
		default:
			renderMarkdown = false
		}
	} else {
		if renderMarkdown == nil {
			renderMarkdown = false
		} else {
			renderMarkdown = renderMarkdown.(bool)
		}
	}
	return map[string]any{
		"title":          title,
		"content":        content,
		"renderMarkdown": renderMarkdown,
		"output": map[string]any{
			"type":           TextDisplay,
			"title":          title,
			"content":        content,
			"renderMarkdown": renderMarkdown,
		},
	}, nil
}
