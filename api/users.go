package api

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"github.com/triapex/auth/api/response"
	"github.com/triapex/auth/config"
	"github.com/triapex/auth/internal/service"
	"github.com/triapex/auth/logger"
	"github.com/triapex/auth/model"
	"github.com/triapex/auth/utils"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

// UsersController ...
type UsersController struct {
	svc           service.UserService
	OAuthCfg      *config.OAuth2
	EnrollerCfg   *config.Enroller
	RSAPrivateKey *rsa.PrivateKey
	RSAPublicKey  *rsa.PublicKey
	lgr           logger.StructLogger
}

// NewUsersController ...
func NewUsersController(svc service.UserService, oauthCFG *config.OAuth2, enrollerCFG *config.Enroller, rsaPubKey *rsa.PublicKey, rsaPrivateKey *rsa.PrivateKey, lgr logger.StructLogger) *UsersController {
	return &UsersController{
		svc:           svc,
		OAuthCfg:      oauthCFG,
		lgr:           lgr,
		RSAPublicKey:  rsaPubKey,
		RSAPrivateKey: rsaPrivateKey,
		EnrollerCfg:   enrollerCFG,
	}
}

// SetLogger ...
func (uc *UsersController) SetLogger(lgr logger.StructLogger) {
	uc.lgr = lgr
}

type signUpPld struct {
	Email     string         `json:"email"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Password  string         `json:"password"`
	Phone     string         `json:"phone"`
	Type      utils.UserType `json:"type"`
	Roles     []string       `json:"roles"`
}

// SignUpUser ...
func (uc *UsersController) SignUpUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("SignUpUser", tid, "initialize")

	// decode request body to UserInfo
	body := &signUpPld{}
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if body.Roles == nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, "User roles required", nil)
		return
	}

	if ok := utils.IsRolesValid(body.Roles); !ok {
		_ = response.ServeJSON(w, http.StatusBadRequest, "User roles are invalid", nil)
		return
	}

	if len(strings.TrimSpace(body.Password)) < 8 {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Password minimum length should be 8", nil)
		return
	}

	if len(strings.TrimSpace(body.FirstName)) < 2 {
		_ = response.ServeJSON(w, http.StatusBadRequest, "FirstName minimum length should be 4", nil)
		return
	}

	if len(strings.TrimSpace(body.LastName)) < 2 {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Lastname minimum length should be 4", nil)
		return
	}

	if !utils.IsValidPhoneNumber(body.Phone) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Please give valid phone number", nil)
		return
	}

	tokensResponse, err := uc.svc.SignUpUser(ctx, &model.SignupPayload{
		UserInfo: &model.UserInfo{
			FirstName: body.FirstName,
			LastName:  body.LastName,
			Phone:     body.Phone,
			Email:     body.Email,
			Password:  body.Password,
			Type:      body.Type,
			Roles:     body.Roles,
			Verified:  true,
		},
	})
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, utils.SuccessMessage, tokensResponse)
	if err == nil {
		uc.lgr.Println("SignUpUser", tid, "complete!")
	}
}

// Login ...
func (uc *UsersController) Login(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("Login", tid, "initialize")

	// decode request body to UserInfo
	loginRequestBody := &model.LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(loginRequestBody); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if utils.IsValueEmpty(loginRequestBody.Identifier) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Please give valid phone number or mail id", nil)
		return
	}

	if utils.IsValueEmpty(loginRequestBody.Password) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Please give password", nil)
		return
	}

	tokensResponse, err := uc.svc.Login(r.Context(), loginRequestBody)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, "Logged in successful", tokensResponse)
	if err == nil {
		uc.lgr.Println("Login", tid, "complete!")
	}
}

func (uc *UsersController) ssoLogin(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("sso login", tid, "initialize")

	http.Redirect(w, r, uc.OAuthCfg.Oauth2.AuthCodeURL("state-token", oauth2.AccessTypeOffline), http.StatusTemporaryRedirect)
}

func (uc *UsersController) callback(w http.ResponseWriter, r *http.Request) {
	tokensResponse, err := uc.svc.SSOLogin(r.Context(), &model.LoginRequest{
		Identifier: r.URL.Query().Get("code"),
	})
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	resp := map[string]string{
		"access_token":  tokensResponse.AccessToken,
		"refresh_token": tokensResponse.RefreshToken,
		"token_type":    "Bearer",
		//"expires_in":    fmt.Sprintf("%.0f", tokensResponse),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (uc *UsersController) TokenRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("TokenRefresh", tid, "Initialize")

	// decode request body to UserInfo
	tokenRefreshRequest := &model.TokenRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(tokenRefreshRequest); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// validate user field
	if utils.IsValueEmpty(tokenRefreshRequest.Token) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "user token are empty", nil)
		return
	}

	tokensResponseBody, err := uc.svc.TokenRefresh(r.Context(), tokenRefreshRequest)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, "Token refresh successfully", tokensResponseBody)
	if err == nil {
		uc.lgr.Println("TokenRefresh", tid, "complete!")
	}
}

func (uc *UsersController) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("ChangePassword", tid, "Initialize")
	resetPasswordRequest := &model.PasswordResetReq{}
	if err := json.NewDecoder(r.Body).Decode(resetPasswordRequest); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if resetPasswordRequest.NewPassword != resetPasswordRequest.RetypePassword {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Password does no match", resetPasswordRequest)
		return
	}
	err := uc.svc.ResetPassword(r.Context(), resetPasswordRequest.NewPassword, resetPasswordRequest.ID)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	err = response.ServeJSON(w, http.StatusOK, "Password Updated successfully", nil)
	if err == nil {
		uc.lgr.Println("ChangePassword", tid, "complete!")
	}
}

func (uc *UsersController) Enroll(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("Enroll", tid, "Initialize")
	userID := r.Header.Get("UserID")

	user, err := uc.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = uc.svc.EnrollUser(r.Context(), user)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, "User Enrolled Successfully", nil)
	if err == nil {
		uc.lgr.Println("VerifyResetPasswordCode", tid, "complete!")
	}
}

type revokeUserPld struct {
	Reason string `json:"reason"`
}

func (uc *UsersController) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("Enroll", tid, "Initialize")
	userID := r.Header.Get("UserID")

	user, err := uc.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// decode request body to UserInfo
	body := &revokeUserPld{}
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = uc.svc.RevokeUser(r.Context(), user, body.Reason)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, "User Enrolled Successfully", nil)
	if err == nil {
		uc.lgr.Println("VerifyResetPasswordCode", tid, "complete!")
	}
}
