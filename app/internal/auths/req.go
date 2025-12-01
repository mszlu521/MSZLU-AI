package auths

type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
}

type VerifyEmailReq struct {
	Token string `json:"token" form:"token" binding:"required"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type ForgetPasswordReq struct {
	Email string `json:"email" binding:"required"`
}
type VerifyCodeReq struct {
	Code  string `json:"code" binding:"required"`
	Email string `json:"email" binding:"required"`
}

type ResetPasswordReq struct {
	Token       string `json:"token" binding:"required"`
	Email       string `json:"email" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}
