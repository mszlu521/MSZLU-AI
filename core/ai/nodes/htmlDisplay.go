package nodes

import "context"

type HtmlDisplayNode struct {
	data map[string]any
}

func NewHtmlDisplayNode(data map[string]any) *HtmlDisplayNode {
	if data == nil {
		data = make(map[string]any)
	}
	return &HtmlDisplayNode{
		data: data,
	}
}
func (t *HtmlDisplayNode) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	htmlContent, ok := input["htmlContent"]
	if !ok {
		if htmlData, exists := t.data["htmlContent"].(map[string]any)["fieldValue"]; exists {
			htmlContent = htmlData.(string)
		}
	} else {
		htmlContent = htmlContent.(string)
	}
	result := map[string]any{
		"type": HtmlDisplay,
		"output": map[string]any{
			"type":        HtmlDisplay,
			"htmlContent": htmlContent,
		},
	}
	return result, nil
}
