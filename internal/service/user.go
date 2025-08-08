package service

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/triapex/auth/config"
	"golang.org/x/crypto/bcrypt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/triapex/auth/internal/infra"
	"github.com/triapex/auth/internal/repo"
	"github.com/triapex/auth/logger"
	"github.com/triapex/auth/model"
	"github.com/triapex/auth/utils"
)

// UserService interface
type UserService interface {
	SignUpUser(ctx context.Context, user *model.SignupPayload) (*model.Token, error)
	SignUpAndEnrollUser(ctx context.Context, user *model.SignupPayload) (string, error)
	SSOLogin(ctx context.Context, loginReq *model.LoginRequest) (*model.Token, error)
	Login(ctx context.Context, loginReq *model.LoginRequest) (*model.Token, error)
	TokenRefresh(ctx context.Context, user *model.TokenRequestBody) (*model.Token, error)
	GetUserByEmailORPhone(ctx context.Context, phone string, email string) (*model.UserInfo, error)
	GetUserByID(ctx context.Context, id string) (*model.UserInfo, error)
	EnrollUser(ctx context.Context, userInfo *model.UserInfo) (string, error)
	RegisterUser(ctx context.Context, userInfo *model.UserInfo) error
	RevokeUser(ctx context.Context, username, reason string) error
}

//Login(ctx context.Context, user *model.LoginRequest) (*model.Token, error)

// User ...
type User struct {
	userRepo    repo.UserRepo
	vaultRepo   repo.VaultRepo
	log         logger.StructLogger
	oAuth       *config.OAuth2
	enrollerCfg *config.Enroller

	PrivateKey           *rsa.PrivateKey
	PublicKey            *rsa.PublicKey
	AccessTokenDuration  time.Duration // token duration in minute
	RefreshTokenDuration time.Duration // refresh token duration in minute
}

// NewUser ...
func NewUser(userRepo repo.UserRepo,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	accessTokenDuration time.Duration,
	RefreshTokenDuration time.Duration,
	lgr logger.StructLogger,
	enrollerCfg *config.Enroller,
	oAuth *config.OAuth2) UserService {
	return &User{
		log:                  lgr,
		userRepo:             userRepo,
		PrivateKey:           privateKey,
		PublicKey:            publicKey,
		AccessTokenDuration:  accessTokenDuration,
		RefreshTokenDuration: RefreshTokenDuration,
		oAuth:                oAuth,
		enrollerCfg:          enrollerCfg,
	}
}

// SetLogger ...
func (u *User) SetLogger(l logger.StructLogger) {
	u.log = l
}

func (u *User) SignUpUser(ctx context.Context, userReq *model.SignupPayload) (*model.Token, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("SignUpUser", tid, "Request for signup from service")

	// user must be created in this block
	// 1.if user already exists in DB, then generate tokens
	// 2. otherwise, create user and generate tokens
	user, err := u.userRepo.GetUserByPhoneOREmail(ctx, userReq.Phone, userReq.Email)
	if err != nil && !errors.Is(err, infra.ErrNotFound) {
		return nil, err
	}
	if user != nil {
		return nil, errors.New("user already exists")
	}
	userReq.Password, err = u.GeneratePasswordHash(userReq.Password)
	if err != nil {
		return nil, err
	}

	if user, err = u.userRepo.CreateUser(ctx, &model.UserInfo{
		Username: userReq.Username,
		Phone:    userReq.Phone,
		Email:    userReq.Email,
		Password: userReq.Password,
		OrgId:    userReq.OrgId,
		Roles:    userReq.Roles,
	}); err != nil {
		return nil, err
	}
	err = u.RegisterUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return u.GenerateToken(&model.TokenPayload{
		Id:    strconv.Itoa(int(user.ID)),
		Email: user.Email,
		Roles: user.Roles,
		Phone: user.Phone,
	})
}
func (u *User) SignUpAndEnrollUser(ctx context.Context, userReq *model.SignupPayload) (string, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("SignUpUser", tid, "Request for signup from service")

	// user must be created in this block
	// 1.if user already exists in DB, then generate tokens
	// 2. otherwise, create user and generate tokens
	user, err := u.userRepo.GetUserByPhoneOREmail(ctx, userReq.Phone, userReq.Email)
	if err != nil && !errors.Is(err, infra.ErrNotFound) {
		return "", err
	}
	if user != nil {
		return "", errors.New("user already exists")
	}
	userReq.Password, err = u.GeneratePasswordHash(userReq.Password)
	if err != nil {
		return "", err
	}

	if user, err = u.userRepo.CreateUser(ctx, &model.UserInfo{
		Username: userReq.Username,
		Phone:    userReq.Phone,
		Email:    userReq.Email,
		Password: userReq.Password,
		OrgId:    userReq.OrgId,
		Roles:    userReq.Roles,
	}); err != nil {
		return "", err
	}

	err = u.RegisterUser(ctx, user)
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "msp-")
	if err != nil {
		return "", err
	}
	tempDir, err = EnrollUser(u.enrollerCfg.Namespace, u.enrollerCfg.Org, u.enrollerCfg.IngressDomain, u.enrollerCfg.TLSCertPath, tempDir, user)
	if err != nil {
		return "", err
	}
	return tempDir, err
}

func (u *User) SSOLogin(ctx context.Context, loginReq *model.LoginRequest) (*model.Token, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("Login", tid, "Request for login from service")

	token, err := u.oAuth.Oauth2.Exchange(ctx, loginReq.Identifier)
	if err != nil {
		return nil, errors.New("failed to exchange token")
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		//http.Error(w, "No ID token found", http.StatusInternalServerError)
		return nil, errors.New("no ID token found")
	}

	idToken, err := u.oAuth.TokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		//http.Error(w, "Invalid ID token", http.StatusUnauthorized)
		return nil, errors.New("invalid ID token")
	}

	var claims *model.GoogleSSOClaims

	if err := idToken.Claims(&claims); err != nil {
		//http.Error(w, "Failed to parse claims", http.StatusUnauthorized)
		return nil, err
	}

	user, err := u.userRepo.GetUserByPhoneOREmail(ctx, claims.Email, claims.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found using email/phone")
	}

	return u.GenerateToken(&model.TokenPayload{
		Id:    strconv.Itoa(int(user.ID)),
		Email: user.Email,
		Phone: user.Phone,
		Roles: user.Roles,
	})

	//return nil, nil
}

func (u *User) Login(ctx context.Context, loginReq *model.LoginRequest) (*model.Token, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("Login", tid, "Request for login from service")

	// find a user via email or password
	// 1.if found, check password and generate tokens
	// 2. other-wise return error
	user, err := u.userRepo.GetUserByPhoneOREmail(ctx, loginReq.Identifier, loginReq.Identifier)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found using email/phone")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
		return nil, errors.New(fmt.Sprintf("password is not matched: %v", err))
	}

	return u.GenerateToken(&model.TokenPayload{
		Id:    strconv.Itoa(int(user.ID)),
		Email: user.Email,
		Phone: user.Phone,
		Roles: user.Roles,
	})
}

func (u *User) TokenRefresh(ctx context.Context, tokenRefreshReq *model.TokenRequestBody) (*model.Token, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("TokenRefresh", tid, "Request for refresh Token from service")

	// extract user info from requested refresh token
	// then generate new access token and refresh token
	user, err := u.ValidateAndParseToken(tokenRefreshReq.Token)
	if err != nil {
		return nil, err
	}

	return u.GenerateToken(&model.TokenPayload{
		Id:    strconv.Itoa(int(user.ID)),
		Email: user.Email,
		Phone: user.Phone,
		Roles: user.Roles,
	})
}

func (u *User) GetUserByEmailORPhone(ctx context.Context, phone string, email string) (*model.UserInfo, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("GetUserByPhoneOREmail", tid, "Request for Get User By Identifier OR Phone from service")
	return u.userRepo.GetUserByPhoneOREmail(ctx, phone, email)
}
func (u *User) GetUserByID(ctx context.Context, id string) (*model.UserInfo, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("GetUserByPhoneOREmail", tid, "Request for Get User By Identifier OR Phone from service")
	return u.userRepo.GetUserById(ctx, id)
}

func (u *User) EnrollUser(ctx context.Context, userInfo *model.UserInfo) (string, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("EnrollUser", tid, "Request for Enroll User from service")
	tempDir, err := os.MkdirTemp("", "msp-")
	if err != nil {
		return "", err
	}

	return EnrollUser(u.enrollerCfg.Namespace, u.enrollerCfg.Org, u.enrollerCfg.IngressDomain, u.enrollerCfg.TLSCertPath, tempDir, userInfo)
}
func (u *User) RegisterUser(ctx context.Context, userInfo *model.UserInfo) error {
	tid := utils.GetTracingID(ctx)
	u.log.Println("EnrollUser", tid, "Request for Enroll User from service")

	return RegisterUser(u.enrollerCfg.Namespace, u.enrollerCfg.Org, u.enrollerCfg.IngressDomain, u.enrollerCfg.TLSCertPath, u.enrollerCfg.RCAMSPPath, userInfo)

}

func (u *User) RevokeUser(ctx context.Context, username, reason string) error {
	tid := utils.GetTracingID(ctx)
	u.log.Println("EnrollUser", tid, "Request for Enroll User from service")

	// Revoke using RCA admin MSP
	cmd := exec.Command("fabric-ca-client", "revoke",
		"--revoke.name", username,
		"--revoke.reason", reason,
		"--url", fmt.Sprintf("https://%s-%s-ca-ca.%s", u.enrollerCfg.Namespace, u.enrollerCfg.Org, u.enrollerCfg.IngressDomain),
		"--tls.certfiles", u.enrollerCfg.TLSCertPath,
		"--mspdir", u.enrollerCfg.RCAMSPPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("revocation failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}
