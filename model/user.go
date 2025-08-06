package model

import (
	"github.com/lib/pq"
	"github.com/triapex/auth/utils"
	"gorm.io/gorm"
	"time"
)

type GoogleSSOClaims struct {
	Email string `json:"email"`
}

type SignupPayload struct {
	UserInfo *UserInfo `json:"user_info"`
	OrgId    uint      `json:"org_id"`
}

// UserInfo define user details
type UserInfo struct {
	gorm.Model
	FirstName string         `json:"first_name" gorm:"column:first_name"`
	LastName  string         `json:"last_name" gorm:"column:last_name"`
	Username  string         `json:"username" gorm:"column:username"`
	Phone     string         `json:"phone" gorm:"column:phone;unique"`
	Email     string         `json:"email" gorm:"column:email;unique"`
	Password  string         `json:"password" gorm:"column:password"`
	Type      utils.UserType `json:"type" gorm:"column:type"`
	Roles     pq.StringArray `json:"roles" gorm:"type:text[];column:roles"`
	Verified  bool           `json:"verified" gorm:"column:verified"`
}

type Profile struct {
	gorm.Model
	FirstName string         `json:"first_name" gorm:"column:first_name"`
	LastName  string         `json:"last_name" gorm:"column:last_name"`
	Phone     string         `json:"phone" gorm:"column:phone"`
	Email     string         `json:"email" gorm:"column:email"`
	Address   string         `json:"address" gorm:"column:address"`
	Type      utils.UserType `json:"type" gorm:"column:type"`
}

type LoginRequest struct {
	Identifier string         `json:"identifier"`
	Password   string         `json:"password"`
	Type       utils.UserType `json:"type" gorm:"column:type"`
}

type TokenRequestBody struct {
	Token string `json:"token"`
}

type VerificationCodeReq struct {
	Identifier string         `json:"identifier"`
	Type       utils.UserType `json:"type" gorm:"column:type"`
}

type VerificationInfo struct {
	ID          string    `json:"id" gorm:"column:_id"`
	Phone       string    `json:"phone"`
	Code        string    `json:"code"`
	ExpiredTime time.Time `json:"expired_time"`
}

type VerificationSuccessResponse struct {
	ID    string `json:"id" gorm:"column:_id"`
	Phone string `json:"phone"`
}

type CodeVerifyReq struct {
	ID   string `json:"id" gorm:"column:_id"`
	Code string `json:"code"`
}

type PasswordResetReq struct {
	ID             string `json:"id" gorm:"column:_id"`
	NewPassword    string `json:"new_password"`
	RetypePassword string `json:"retype_password"`
}

type SMSRequest struct {
	User   interface{} `json:"user,omitempty" gorm:"column:user,omitempty"`
	Data   string      `json:"data,omitempty" gorm:"column:data,omitempty"`
	IsBulk bool        `gorm:"column:is_bulk,omitempty" gorm:"column:is_bulk,omitempty"`
}
