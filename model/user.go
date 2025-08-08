package model

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type GoogleSSOClaims struct {
	Email string `json:"email"`
}

type SignupPayload struct {
	Username string         `json:"username"`
	Phone    string         `json:"phone"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	OrgId    string         `json:"org_id"`
	Roles    pq.StringArray `json:"roles"`
}

// UserInfo define user details
type UserInfo struct {
	gorm.Model
	Username string         `json:"username" gorm:"column:username"`
	Phone    string         `json:"phone" gorm:"column:phone;unique"`
	Email    string         `json:"email" gorm:"column:email;unique"`
	Password string         `json:"password" gorm:"column:password"`
	OrgId    string         `json:"org_id" gorm:"column:org_id"`
	Roles    pq.StringArray `json:"roles" gorm:"type:text[];column:roles"`
}
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type TokenRequestBody struct {
	Token string `json:"token"`
}
