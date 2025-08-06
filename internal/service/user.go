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
	SSOLogin(ctx context.Context, loginReq *model.LoginRequest) (*model.Token, error)
	Login(ctx context.Context, loginReq *model.LoginRequest) (*model.Token, error)
	TokenRefresh(ctx context.Context, user *model.TokenRequestBody) (*model.Token, error)
	GetUserByEmailORPhone(ctx context.Context, phone string, email string) (*model.UserInfo, error)
	GetUserByID(ctx context.Context, id string) (*model.UserInfo, error)
	SendResetPasswordCode(ctx context.Context, userInfo *model.UserInfo) (*model.VerificationSuccessResponse, error)
	VerifyResetPasswordCode(ctx context.Context, pinVerifyRequestBody *model.CodeVerifyReq) error
	VerifyOtp(ctx context.Context, pinVerifyRequestBody *model.CodeVerifyReq) error
	ResetPassword(ctx context.Context, password string, id string) error
	SendOtp(ctx context.Context, userInfo *model.UserInfo) (*model.VerificationSuccessResponse, error)
	EnrollUser(ctx context.Context, userInfo *model.UserInfo) error
	RevokeUser(ctx context.Context, userInfo *model.UserInfo, reason string) error
}

//Login(ctx context.Context, user *model.LoginRequest) (*model.Token, error)

// User ...
type User struct {
	userRepo    repo.UserRepo
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
	user, err := u.userRepo.GetUserByPhoneOREmail(ctx, userReq.UserInfo.Phone, userReq.UserInfo.Email)
	if err != nil && !errors.Is(err, infra.ErrNotFound) {
		return nil, err
	}
	if user != nil && user.Verified {
		return nil, errors.New("user already exists")
	}
	if user == nil {
		userReq.UserInfo.Password, err = u.GeneratePasswordHash(userReq.UserInfo.Password)
		if err != nil {
			return nil, err
		}

		if user, err = u.userRepo.CreateUser(ctx, userReq.UserInfo); err != nil {
			return nil, err
		}

	}

	if userReq.UserInfo.Type != user.Type {
		err = u.userRepo.UpdateUserType(ctx, userReq.UserInfo.ID, string(userReq.UserInfo.Type))
	}

	userProfile := &model.Profile{
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		Email:     user.Email,
		Address:   "",
		Type:      user.Type,
	}
	err = u.userRepo.CreateProfile(ctx, userProfile)
	if err != nil {
		return nil, err
	}

	user, err = u.userRepo.GetUserByPhoneOREmail(ctx, userReq.UserInfo.Phone, userReq.UserInfo.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found using email/phone")
	}

	return u.GenerateToken(&model.TokenPayload{
		Id:       user.ID,
		Email:    user.Email,
		Roles:    user.Roles,
		Phone:    user.Phone,
		Type:     user.Type,
		Verified: user.Verified,
	})
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

	if !user.Verified {
		return nil, errors.New("user not verified yet")
	}

	return u.GenerateToken(&model.TokenPayload{
		Id:       user.ID,
		Email:    user.Email,
		Phone:    user.Phone,
		Roles:    user.Roles,
		Type:     user.Type,
		Verified: user.Verified,
	})

	//return nil, nil
}

func (u *User) Login(ctx context.Context, loginReq *model.LoginRequest) (*model.Token, error) {
	tid := utils.GetTracingID(ctx)
	u.log.Println("Login", tid, "Request for login from service")

	// find user via email or password
	// 1.if found, check password and generate tokens
	// 2. other-wise return error
	user, err := u.userRepo.GetUserByPhoneOREmail(ctx, loginReq.Identifier, loginReq.Identifier)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found using email/phone")
	}

	if !user.Verified {
		return nil, errors.New("user not verified yet")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
		return nil, errors.New(fmt.Sprintf("password is not matched: %v", err))
	}

	return u.GenerateToken(&model.TokenPayload{
		Id:       user.ID,
		Email:    user.Email,
		Phone:    user.Phone,
		Type:     user.Type,
		Roles:    user.Roles,
		Verified: user.Verified,
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
		Id:       user.ID,
		Email:    user.Email,
		Phone:    user.Phone,
		Type:     user.Type,
		Roles:    user.Roles,
		Verified: user.Verified,
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

func (u *User) SendResetPasswordCode(ctx context.Context, userInfo *model.UserInfo) (*model.VerificationSuccessResponse, error) {
	/*
		tid := utils.GetTracingID(ctx)
			u.log.Println("SendResetPasswordCode", tid, "Request for send verificationInfo code to mail from service")
			code := utils.GenerateCode(1000, 9999)
			smsRequest := &model.SMSRequest{
				User:   userInfo.Phone,
				Data:   strconv.Itoa(code),
				IsBulk: false,
			}

			if viper.GetString("app.env") == utils.AppEnvTest {
				code = 1234
			} else {
				err := sendMessageToNotification(smsRequest)
				if err != nil {
					return nil, err
				}
			}
			verification := &model.VerificationInfo{
				ID:          userInfo.ID,
				Phone:       userInfo.Phone,
				Code:        strconv.Itoa(code),
				ExpiredTime: time.Now().UTC().Add(time.Minute * 5),
			}
			err := u.userRepo.SendResetPasswordCode(ctx, verification)
			if err != nil {
				return nil, err
			}
			verificationSuccess := &model.VerificationSuccessResponse{
				ID:    userInfo.ID,
				Phone: userInfo.Phone,
			}
			return verificationSuccess, nil
	*/
	return nil, nil
}

func (u *User) VerifyResetPasswordCode(ctx context.Context, pinVerifyRequestBody *model.CodeVerifyReq) error {
	tid := utils.GetTracingID(ctx)
	u.log.Println("VerifyResetPasswordCode", tid, "Request for verify pin code from service")
	verification, err := u.userRepo.VerifyResetPasswordCode(ctx, pinVerifyRequestBody.ID)
	if err != nil {
		return err
	}
	if pinVerifyRequestBody.Code != verification.Code {
		return errors.New("code verification failed")
	}
	if time.Now().UTC().After(verification.ExpiredTime) {
		return errors.New("code Expired")
	}
	return nil
}

func (u *User) ResetPassword(ctx context.Context, password string, id string) error {
	tid := utils.GetTracingID(ctx)
	u.log.Println("ChangePassword", tid, "Request for reset Password from service")
	hashPass, err := u.GeneratePasswordHash(password)
	if err != nil {
		return err
	}
	err = u.userRepo.ResetPassword(ctx, hashPass, id)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) SendOtp(ctx context.Context, userInfo *model.UserInfo) (*model.VerificationSuccessResponse, error) {
	/*
		tid := utils.GetTracingID(ctx)

			u.log.Println("SendOtp", tid, "Send Otp request for login  from service")
			code := strconv.Itoa(utils.GenerateCode(10000, 99999))
			smsRequest := &model.SMSRequest{
				User:   userInfo.Phone,
				Data:   code,
				IsBulk: false,
			}

			if viper.GetString("app.env") == utils.AppEnvTest {
				code = strconv.Itoa(12345)
			} else {
				err := sendMessageToNotification(smsRequest)
				if err != nil {
					return nil, err
				}
			}
			verification := &model.VerificationInfo{
				ID:          userInfo.ID,
				Phone:       userInfo.Phone,
				Code:        code,
				ExpiredTime: time.Now().UTC().Add(time.Minute * 5),
			}
			err := u.userRepo.SendOtp(ctx, verification)
			if err != nil {
				return nil, err
			}
			verificationSuccess := &model.VerificationSuccessResponse{
				ID:    userInfo.ID,
				Phone: userInfo.Phone,
			}
			return verificationSuccess, nil
	*/

	return nil, nil
}

func (u *User) VerifyOtp(ctx context.Context, pinVerifyRequestBody *model.CodeVerifyReq) error {
	tid := utils.GetTracingID(ctx)
	u.log.Println("VerifyOtp", tid, "Request for verify otp from service")
	verification, err := u.userRepo.VerifyOtp(ctx, pinVerifyRequestBody.ID)
	if err != nil {
		return err
	}
	if pinVerifyRequestBody.Code != verification.Code {
		return errors.New("otp verification failed")
	}
	if time.Now().UTC().After(verification.ExpiredTime) {
		return errors.New("otp Expired")
	}
	return u.userRepo.UpdateUserVerificationStatus(ctx, pinVerifyRequestBody.ID, true)
}

func (u *User) EnrollUser(ctx context.Context, userInfo *model.UserInfo) error {
	tid := utils.GetTracingID(ctx)
	u.log.Println("EnrollUser", tid, "Request for Enroll User from service")
	tempDir, err := os.MkdirTemp("", "msp-")
	if err != nil {
		return err
	}

	caAddress := fmt.Sprintf("%s-%s-ca-ca.%s", u.enrollerCfg.Namespace, u.enrollerCfg.Org, u.enrollerCfg.IngressDomain)
	enrollURL := fmt.Sprintf("https://%s:%s@%s", userInfo.Username, userInfo.Password, caAddress)

	cmd := exec.Command("fabric-ca-client", "enroll",
		"--url", enrollURL,
		"--tls.certfiles", u.enrollerCfg.TLSCertPath,
		"--mspdir", tempDir,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("enrollment failed: %v\nOutput: %s", err, string(output))
	}

	if err := u.userRepo.StoreUserMSP(ctx, userInfo.Username, tempDir); err != nil {
		return errors.New("failed to store user msp")
	}

	return nil
}

func (u *User) RevokeUser(ctx context.Context, userInfo *model.UserInfo, reason string) error {
	tid := utils.GetTracingID(ctx)
	u.log.Println("EnrollUser", tid, "Request for Enroll User from service")

	// Revoke using RCA admin MSP
	cmd := exec.Command("fabric-ca-client", "revoke",
		"--revoke.name", userInfo.Username,
		"--revoke.reason", reason,
		"--url", fmt.Sprintf("https://%s-%s-ca-ca.%s", u.enrollerCfg.Namespace, u.enrollerCfg.Org, u.enrollerCfg.IngressDomain),
		"--tls.certfiles", u.enrollerCfg.TLSCertPath,
		"--mspdir", u.enrollerCfg.RCAMSPPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("revocation failed: %v\nOutput: %s", err, string(output))
	}

	if err := u.userRepo.RemoveUserMSP(ctx, userInfo.Username); err != nil {
		return errors.New("failed to store user msp")
	}

	return nil
}
