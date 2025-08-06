package model

import (
	"github.com/lib/pq"
	"github.com/triapex/auth/utils"
)

type Token struct {
	UserId       uint   `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenPayload struct {
	Id       uint           `json:"id"`
	Email    string         `json:"email"`
	Roles    pq.StringArray `json:"role"`
	Phone    string         `json:"phone"`
	Type     utils.UserType `json:"type"`
	Verified bool           `json:"verified"`
}
