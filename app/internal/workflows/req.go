package workflows

import (
	"model"

	"github.com/google/uuid"
)

type createWorkflowReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type listReq struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type updateReq struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type saveReq struct {
	Id          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Data        *model.Graph `json:"data"`
	Version     int          `json:"version"`
}

type executeReq struct {
	WorkflowId uuid.UUID    `json:"workflowId"`
	Data       *model.Graph `json:"data"`
}
