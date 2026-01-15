package knowledges

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/errs"
	"github.com/mszlu521/thunder/logs"
	"github.com/mszlu521/thunder/req"
	"github.com/mszlu521/thunder/res"
)

type Handler struct {
	service *service
}

func (h *Handler) CreateKnowledgeBase(c *gin.Context) {
	var createReq createKnowledgeBaseReq
	if err := req.JsonParam(c, &createReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.createKnowledgeBase(c.Request.Context(), userId, createReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) ListKnowledgeBases(c *gin.Context) {
	var params searchReq
	if err := req.JsonParam(c, &params); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.listKnowledgeBases(c.Request.Context(), userId, params.Params)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) GetKnowledgeBase(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.getKnowledgeBase(c.Request.Context(), userId, id)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) UpdateKnowledgeBase(c *gin.Context) {
	var updateReq updateKnowledgeBaseReq
	if err := req.JsonParam(c, &updateReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	resp, err := h.service.updateKnowledgeBase(c.Request.Context(), userId, id, updateReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) DeleteKnowledgeBase(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	err := h.service.deleteKnowledgeBase(c.Request.Context(), userId, id)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func (h *Handler) ListDocuments(c *gin.Context) {
	var params listDocumentReq
	if err := req.QueryParam(c, &params); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	var kbId uuid.UUID
	if err := req.Path(c, "id", &kbId); err != nil {
		return
	}
	resp, err := h.service.listDocuments(c.Request.Context(), userId, kbId, params)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) UploadDocuments(c *gin.Context) {
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	var kbId uuid.UUID
	if err := req.Path(c, "id", &kbId); err != nil {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		res.Error(c, errs.ErrParam)
		return
	}
	resp, err := h.service.uploadDocuments(c.Request.Context(), userId, kbId, file)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) DeleteDocuments(c *gin.Context) {
	var kbId uuid.UUID
	if err := req.Path(c, "id", &kbId); err != nil {
		return
	}
	var documentId uuid.UUID
	if err := req.Path(c, "documentId", &documentId); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	err := h.service.deleteDocuments(c.Request.Context(), userId, kbId, documentId)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func (h *Handler) SearchKnowledgeBase(c *gin.Context) {
	rc := http.NewResponseController(c.Writer)
	//为了防止超时，因为我们加了大模型对话
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		//一般不会失败
		logs.Warnf("SetWriteDeadline error: %v", err)
	}
	var params searchParams
	if err := req.JsonParam(c, &params); err != nil {
		return
	}
	var kbId uuid.UUID
	if err := req.Path(c, "id", &kbId); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.searchKnowledgeBase(c.Request.Context(), userId, kbId, params)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func NewHandler() *Handler {
	return &Handler{
		service: newService(),
	}
}
