package workflows

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/logs"
	"github.com/mszlu521/thunder/req"
	"github.com/mszlu521/thunder/res"
)

type Handler struct {
	service *service
}

func (h *Handler) ListWorkflows(c *gin.Context) {
	var lr listReq
	if err := req.JsonParam(c, &lr); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.listWorkflows(c.Request.Context(), userId, &lr)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) CreateWorkflow(c *gin.Context) {
	var createReq createWorkflowReq
	if err := req.JsonParam(c, &createReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.createWorkflow(c.Request.Context(), userId, &createReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) UpdateWorkflow(c *gin.Context) {
	var ur updateReq
	if err := req.JsonParam(c, &ur); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.updateWorkflow(c.Request.Context(), userId, &ur)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) GetWorkflow(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.getWorkflow(c.Request.Context(), userId, id)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) DeleteWorkflow(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	err := h.service.deleteWorkflow(c.Request.Context(), userId, id)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func (h *Handler) SaveWorkflow(c *gin.Context) {
	var sr saveReq
	if err := req.JsonParam(c, &sr); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	err := h.service.saveWorkflow(c.Request.Context(), userId, &sr)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func (h *Handler) Execute(c *gin.Context) {
	rc := http.NewResponseController(c.Writer)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		//一般不会失败
		logs.Warnf("SetWriteDeadline error: %v", err)
	}

	var reqs executeReq
	if err := req.JsonParam(c, &reqs); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.execute(c.Request.Context(), userId, &reqs)
	if err != nil {
		//这是执行工作流，我们返回一个特定的错误结果
		errResult := make(map[string]any)
		errResult["error"] = err.Error()
		res.Success(c, errResult)
		return
	}
	res.Success(c, resp)
}

func NewHandler() *Handler {
	return &Handler{
		service: newService(),
	}
}
