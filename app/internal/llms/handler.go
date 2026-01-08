package llms

import (
	"github.com/gin-gonic/gin"
	"github.com/mszlu521/thunder/req"
	"github.com/mszlu521/thunder/res"
)

type Handler struct {
	service *service
}

func NewHandler() *Handler {
	return &Handler{
		service: newService(),
	}
}
func (h *Handler) CreateProviderConfig(c *gin.Context) {
	var createReq CreateProviderConfigReq
	if err := req.JsonParam(c, &createReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	createResp, err := h.service.createProviderConfig(c.Request.Context(), userID, createReq)
	if err != nil {
		return
	}
	res.Success(c, createResp)
}

func (h *Handler) ListProviderConfigs(c *gin.Context) {
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.listProviderConfigs(c.Request.Context(), userId)
	if err != nil {
		return
	}
	res.Success(c, resp)
}

func (h *Handler) CreateLLM(c *gin.Context) {
	var createReq CreateLLMReq
	if err := req.JsonParam(c, &createReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	createResp, err := h.service.createLLM(c.Request.Context(), userID, createReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, createResp)
}

func (h *Handler) ListLLMs(c *gin.Context) {
	var listReq ListLLMsReq
	if err := req.QueryParam(c, &listReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.listLLMs(c.Request.Context(), userID, listReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) ListLLMAll(c *gin.Context) {
	var listReq ListLLMsReq
	if err := req.QueryParam(c, &listReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.listLLMAll(c.Request.Context(), userID, listReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}
