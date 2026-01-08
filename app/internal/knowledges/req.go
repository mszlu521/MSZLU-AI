package knowledges

type createKnowledgeBaseReq struct {
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	EmbeddingModelName     string   `json:"embeddingModelName"`
	EmbeddingModelProvider string   `json:"embeddingModelProvider"`
	ChatModelName          string   `json:"chatModelName"`
	ChatModelProvider      string   `json:"chatModelProvider"`
	Tags                   []string `json:"tags"`
}
type updateKnowledgeBaseReq struct {
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	EmbeddingModelName     string   `json:"embeddingModelName"`
	EmbeddingModelProvider string   `json:"embeddingModelProvider"`
	ChatModelName          string   `json:"chatModelName"`
	ChatModelProvider      string   `json:"chatModelProvider"`
	Tags                   []string `json:"tags"`
}
type listReq struct {
	Page     int    `json:"page"`
	PageSize int    `json:"size"`
	Search   string `json:"search"`
}
type searchReq struct {
	Params listReq `json:"params"`
}

type searchParams struct {
	Query string `json:"query"`
}
type listDocumentReq struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"pageSize" form:"pageSize"`
	Search    string `json:"search" form:"search"`
	SortBy    string `json:"sortBy" form:"sortBy"`
	Status    string `json:"status" form:"status"`
	SortOrder string `json:"sortOrder" form:"sortOrder"`
}
