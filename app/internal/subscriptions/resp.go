package subscriptions

import (
	"model"

	"github.com/google/uuid"
)

type SubscriptionResponse struct {
	ID            uuid.UUID         `json:"id"`
	UserID        uuid.UUID         `json:"userId"`
	Plan          string            `json:"plan"`
	Duration      string            `json:"duration"`
	PaymentMethod string            `json:"paymentMethod"`
	StartDate     string            `json:"startDate"`
	EndDate       string            `json:"endDate"`
	CreatedAt     string            `json:"createdAt"`
	Configs       *model.PlanConfig `json:"configs"`
	UpdatedAt     string            `json:"updatedAt"`
}
