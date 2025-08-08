package model

import (
	"github.com/lib/pq"
)

type Token struct {
	UserId       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenPayload struct {
	Id    string         `json:"id"`
	Email string         `json:"email"`
	Roles pq.StringArray `json:"role"`
	Phone string         `json:"phone"`
}
