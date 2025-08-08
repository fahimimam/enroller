package api

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"github.com/go-chi/chi/v5"
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
	userSvc       service.UserService
	vaultSvc      service.VaultService
	OAuthCfg      *config.OAuth2
	EnrollerCfg   *config.Enroller
	RSAPrivateKey *rsa.PrivateKey
	RSAPublicKey  *rsa.PublicKey
	lgr           logger.StructLogger
}

// NewUsersController ...
func NewUsersController(userSvc service.UserService, vaultSvc service.VaultService, oauthCFG *config.OAuth2, enrollerCFG *config.Enroller, rsaPubKey *rsa.PublicKey, rsaPrivateKey *rsa.PrivateKey, lgr logger.StructLogger) *UsersController {
	return &UsersController{
		userSvc:       userSvc,
		vaultSvc:      vaultSvc,
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
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Phone    string   `json:"phone"`
	OrgId    string   `json:"orgId"`
	Roles    []string `json:"roles"`
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

	if utils.IsInValidEmail(body.Email) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Please give valid email id", nil)
	}

	if !utils.IsValidPhoneNumber(body.Phone) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Please give valid phone number", nil)
		return
	}

	tokensResponse, err := uc.userSvc.SignUpUser(ctx, &model.SignupPayload{
		Username: body.Username,
		Phone:    body.Phone,
		Email:    body.Email,
		Password: body.Password,
		OrgId:    body.OrgId,
		Roles:    body.Roles,
	})
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, utils.SuccessMessage, map[string]string{
		"access_token":  tokensResponse.AccessToken,
		"refresh_token": tokensResponse.RefreshToken,
		"username":      body.Username,
		"password":      body.Password, // TODO: Should I send back the password?!
		"token_type":    "Bearer",
	})
	if err == nil {
		uc.lgr.Println("SignUpUser", tid, "complete!")
	}
}
func (uc *UsersController) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("SignUpUser", tid, "initialize")
	userID := r.Header.Get(utils.UserIDHeader)

	if userID == "" {
		_ = response.ServeJSON(w, http.StatusBadRequest, "User id is empty", nil)
		return
	}

	user, _ := uc.userSvc.GetUserByID(ctx, userID)
	if user == nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, "User not found", nil)
		return
	}

	err := uc.userSvc.RegisterUser(ctx, user)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, utils.SuccessMessage, "User Registered Successfully")
	if err == nil {
		uc.lgr.Println("Register", tid, "complete!")
	}
}
func (uc *UsersController) SignUpAndRegisterUser(w http.ResponseWriter, r *http.Request) {
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

	if utils.IsInValidEmail(body.Email) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Please give valid email id", nil)
	}

	if !utils.IsValidPhoneNumber(body.Phone) {
		_ = response.ServeJSON(w, http.StatusBadRequest, "Please give valid phone number", nil)
		return
	}

	tokensResponse, err := uc.userSvc.SignUpAndRegisterUser(ctx, &model.SignupPayload{
		Username: body.Username,
		Phone:    body.Phone,
		Email:    body.Email,
		Password: body.Password,
		OrgId:    body.OrgId,
		Roles:    body.Roles,
	})
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, utils.SuccessMessage, map[string]string{
		"access_token":  tokensResponse.AccessToken,
		"refresh_token": tokensResponse.RefreshToken,
		"username":      body.Username,
		"password":      body.Password, // TODO: Should I send back the password?!
		"token_type":    "Bearer",
	})
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

	tokensResponse, err := uc.userSvc.Login(r.Context(), loginRequestBody)
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
	tokensResponse, err := uc.userSvc.SSOLogin(r.Context(), &model.LoginRequest{
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

	tokensResponseBody, err := uc.userSvc.TokenRefresh(r.Context(), tokenRefreshRequest)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, "Token refresh successfully", tokensResponseBody)
	if err == nil {
		uc.lgr.Println("TokenRefresh", tid, "complete!")
	}
}

func (uc *UsersController) Enroll(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("Enroll", tid, "Initialize")
	userID := r.Header.Get("UserID")

	user, err := uc.userSvc.GetUserByID(r.Context(), userID)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	tempDir, err := uc.userSvc.EnrollUser(r.Context(), user)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := uc.vaultSvc.StoreUserMSP(ctx, user.Username, tempDir); err != nil {
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
	usernameToRemove := chi.URLParam(r, "username")
	user, err := uc.userSvc.GetUserByID(r.Context(), userID)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if user == nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, "User not found", nil)
		return
	}

	//TODO: Add ROLE validation if needed

	// decode request body to UserInfo
	body := &revokeUserPld{}
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = uc.userSvc.RevokeUser(r.Context(), usernameToRemove, body.Reason)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := uc.vaultSvc.RemoveUserMSP(ctx, usernameToRemove); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, "User Enrolled Successfully", nil)
	if err == nil {
		uc.lgr.Println("VerifyResetPasswordCode", tid, "complete!")
	}
}

func (uc *UsersController) GetMsp(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tid := utils.GetTracingID(ctx)
	uc.lgr.Println("Enroll", tid, "Initialize")
	userID := r.Header.Get("UserID")
	username := chi.URLParam(r, "username")
	user, err := uc.userSvc.GetUserByID(r.Context(), userID)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if user == nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, "User not found", nil)
		return
	}

	//TODO: Add ROLE validation if needed

	// decode request body to UserInfo
	body := &revokeUserPld{}
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	mspResp, err := uc.vaultSvc.GetUserMsp(ctx, username)
	if err != nil {
		_ = response.ServeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = response.ServeJSON(w, http.StatusOK, "User Enrolled Successfully", mspResp)
	if err == nil {
		uc.lgr.Println("VerifyResetPasswordCode", tid, "complete!")
	}
}
