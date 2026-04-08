package skills

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

func (h *Handler) CreateSkill(c *gin.Context) {
	var createReq CreateSkillReq
	if err := req.JsonParam(c, &createReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.createSkill(c.Request.Context(), userId, createReq)
	if err != nil {
		logs.Errorf("create skill error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, resp)

}

func (h *Handler) ListSkills(c *gin.Context) {
	var listReq SearchSkillReq
	if err := req.JsonParam(c, &listReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	skills, err := h.service.listSkills(c.Request.Context(), userId, listReq)
	if err != nil {
		logs.Errorf("list skills error: %v", err)
		res.Error(c, err)
	}
	res.Success(c, skills)
}

func (h *Handler) ListSkillsAll(c *gin.Context) {
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	skills, err := h.service.listSkillsAll(c.Request.Context(), userId)
	if err != nil {
		logs.Errorf("list skills error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, skills)
}

func (h *Handler) UpdateSkill(c *gin.Context) {
	var updateReq UpdateSkillReq
	if err := req.JsonParam(c, &updateReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.updateSkill(c.Request.Context(), userId, updateReq)
	if err != nil {
		logs.Errorf("update skill error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) DeleteSkill(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	if err := h.service.deleteSkill(c.Request.Context(), userId, id); err != nil {
		logs.Errorf("delete skill error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func (h *Handler) CreateGithubSources(c *gin.Context) {
	var createReq CreateGithubSourceReq
	if err := req.JsonParam(c, &createReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.createGithubSources(c.Request.Context(), userId, createReq)
	if err != nil {
		logs.Errorf("create github sources error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) UpdateGithubSources(c *gin.Context) {
	var updateReq UpdateGithubSourceReq
	if err := req.JsonParam(c, &updateReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.updateGithubSources(c.Request.Context(), userId, updateReq)
	if err != nil {
		logs.Errorf("update github sources error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) DeleteGithubSources(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	if err := h.service.deleteGithubSources(c.Request.Context(), userId, id); err != nil {
		logs.Errorf("delete github sources error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, nil)
}

func (h *Handler) ListGithubSources(c *gin.Context) {
	var listReq SearchGithubSourceReq
	if err := req.JsonParam(c, &listReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	githubSources, err := h.service.listGithubSources(c.Request.Context(), userId, listReq)
	if err != nil {
		logs.Errorf("list github sources error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, githubSources)
}

func (h *Handler) GetGithubSource(c *gin.Context) {
	var id uuid.UUID
	if err := req.Path(c, "id", &id); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	githubSource, err := h.service.getGithubSource(c.Request.Context(), userId, id)
	if err != nil {
		logs.Errorf("get github source error: %v", err)
		res.Error(c, err)
		return
	}
	res.Success(c, githubSource)
}

func (h *Handler) InstallSkill(c *gin.Context) {
	rc := http.NewResponseController(c.Writer)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		//一般不会失败
		logs.Warnf("SetWriteDeadline error: %v", err)
	}
	var installReq InstallSkillReq
	if err := req.JsonParam(c, &installReq); err != nil {
		return
	}
	userId, ok := req.GetUserIdUUID(c)
	if !ok {
		return
	}
	resp, err := h.service.installSkill(c.Request.Context(), userId, installReq)
	if err != nil {
		logs.Errorf("install skill error: %v", err)
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
