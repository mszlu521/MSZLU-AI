package nodes

type testNodeResp struct {
	Success bool           `json:"success"`
	Output  map[string]any `json:"output"`
	Error   string         `json:"error"`
}
