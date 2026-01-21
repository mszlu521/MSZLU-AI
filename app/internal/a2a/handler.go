package a2a

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/req"
	"github.com/mszlu521/thunder/res"
)

type Handler struct {
	service *service
}

func (h *Handler) GetAgentCard(c *gin.Context) {
	var r GetAgentCardReq
	if err := req.JsonParam(c, &r); err != nil {
		return
	}
	resp, err := h.service.getAgentCard(r)
	if err != nil {
		return
	}
	res.Success(c, resp)
}

func (h *Handler) SaveAgentCard(c *gin.Context) {
	var r GetAgentCardReq
	if err := req.JsonParam(c, &r); err != nil {
		return
	}
	resp, err := h.service.saveAgentCard(r)
	if err != nil {
		return
	}
	res.Success(c, resp)
}

func (h *Handler) ListAgentMarkets(c *gin.Context) {
	cards, err := h.service.listAgentMarkets()
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, cards)
}

func (h *Handler) Delete(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	if err := h.service.delete(id); err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func NewHandler() *Handler {
	return &Handler{
		service: newService(),
	}
}
