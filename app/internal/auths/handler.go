package auths

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
func (h *Handler) Register(c *gin.Context) {
	var regReq RegisterReq
	if err := req.JsonParam(c, &regReq); err != nil {
		return
	}
	resp, err := h.service.register(regReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	//获取参数
	var verifyReq VerifyEmailReq
	if err := req.QueryParam(c, &verifyReq); err != nil {
		return
	}
	_, err := h.service.verifyEmail(verifyReq.Token)
	if err != nil {
		res.Error(c, err)
		return
	}
	//这个地方应该跳转到登录页面
	c.Redirect(302, "http://localhost:5173/login")
}

func (h *Handler) Login(c *gin.Context) {
	var loginReq LoginReq
	if err := req.JsonParam(c, &loginReq); err != nil {
		return
	}
	resp, err := h.service.login(loginReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var refreshReq RefreshTokenReq
	if err := req.JsonParam(c, &refreshReq); err != nil {
		return
	}
	resp, err := h.service.refreshToken(refreshReq.RefreshToken)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var forgetReq ForgetPasswordReq
	if err := req.JsonParam(c, &forgetReq); err != nil {
		return
	}
	resp, err := h.service.forgotPassword(forgetReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) VerifyCode(c *gin.Context) {
	var verifyCodeReq VerifyCodeReq
	if err := req.JsonParam(c, &verifyCodeReq); err != nil {
		return
	}
	resp, err := h.service.verifyCode(verifyCodeReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var resetReq ResetPasswordReq
	if err := req.JsonParam(c, &resetReq); err != nil {
		return
	}
	resp, err := h.service.resetPassword(c, resetReq)
	if err != nil {
		res.Error(c, err)
		return
	}
	res.Success(c, resp)
}
