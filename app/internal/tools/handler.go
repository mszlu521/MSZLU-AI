package tools

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/req"
	"github.com/mszlu521/thunder/res"
)

type Handler struct {
	service *service
}

func (h *Handler) CreateTool(c *gin.Context) {
	var createReq CreateToolReq
	if err := req.JsonParam(c, &createReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	tool, err := h.service.createTool(c.Request.Context(), userID, createReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, tool)
}

func (h *Handler) ListTools(c *gin.Context) {
	var listReq ListToolsReq
	if err := req.QueryParam(c, &listReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	//判断一下分页 如果没有就查询全部
	tools, err := h.service.listTools(c.Request.Context(), userID, listReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, tools)
}

func (h *Handler) UpdateTool(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	var updateReq UpdateToolReq
	if err := req.JsonParam(c, &updateReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	tool, err := h.service.updateTool(c.Request.Context(), userID, id, updateReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, tool)
}

func (h *Handler) DeleteTool(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	err := h.service.deleteTool(c.Request.Context(), userID, id)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func (h *Handler) TestTool(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	var testReq TestToolReq
	if err := req.JsonParam(c, &testReq); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.testTool(c.Request.Context(), userID, id, testReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) GetMcpTools(c *gin.Context) {
	var mcpId uuid.UUID
	if err := req.Path(c, "mcpId", &mcpId); err != nil {
		return
	}
	userID, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	tools, err := h.service.getMcpTools(c.Request.Context(), userID, mcpId)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, tools)
}

func NewHandler() *Handler {
	return &Handler{
		service: newService(),
	}
}
