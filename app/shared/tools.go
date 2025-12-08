package shared

import "github.com/google/uuid"

type GetToolsByIdsRequest struct {
	Ids []uuid.UUID `json:"ids"`
}
