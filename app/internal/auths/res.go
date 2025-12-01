package auths

import "model"

type RegisterResp struct {
	Message string `json:"message"`
}

type LoginResp struct {
	Expire        int64          `json:"expire"`
	Token         string         `json:"token"`
	UserInfo      *model.UserDTO `json:"userInfo"`
	Message       string         `json:"message"`
	RefreshToken  string         `json:"refreshToken"`
	RefreshExpire int64          `json:"refreshExpire"`
}
type VerifyCodeResp struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}
