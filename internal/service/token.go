package service

import (
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/golang-jwt/jwt/v5"
	"github.com/triapex/auth/model"
	"github.com/triapex/auth/utils"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"strconv"
	"time"
)

type CustomClaims struct {
	jwt.RegisteredClaims
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	OrgID  string   `json:"org_id"`
	UserID string   `json:"user_id"`
}

func (u *User) GenerateToken(user *model.TokenPayload) (*model.Token, error) {
	randStr := u.GenerateRandomString(32)

	// generate an access token
	accessToken, err := u.generateToken(user, randStr)
	if err != nil {
		return nil, err
	}

	// generate a refresh token
	refreshToken, err := u.generateRefreshToken(user, randStr)
	if err != nil {
		return nil, err
	}

	return &model.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserId:       user.Id,
	}, nil
}

func (u *User) generateToken(tPayload *model.TokenPayload, randStr string) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)

	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(u.AccessTokenDuration * time.Minute)),
			Issuer:    utils.JWTTokenIssuer,
		},
		Email:  tPayload.Email,
		Roles:  tPayload.Roles,
		UserID: tPayload.Id,
	}
	token.Claims = claims

	tokenString, err := token.SignedString(u.PrivateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (u *User) generateRefreshToken(tPayload *model.TokenPayload, randStr string) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)

	payload, err := json.Marshal(tPayload)
	if err != nil {
		return "", nil
	}

	claims := jwt.MapClaims{
		"jti":     randStr,
		"exp":     time.Now().Add(u.RefreshTokenDuration * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
		"aud":     tPayload.Id,
		"payload": string(payload),
	}
	token.Claims = claims

	tokenString, err := token.SignedString(u.PrivateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (u *User) ValidateAndParseToken(tokenString string) (*model.UserInfo, error) {
	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return u.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	userInfo := &model.UserInfo{}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		var payload string
		if val, ok := claims["payload"]; ok {
			if val, ok := val.(string); ok {
				payload = val
			}
		}
		data := &model.TokenPayload{}
		if err := json.Unmarshal([]byte(payload), data); err != nil {
			return nil, err
		}
		val, _ := strconv.Atoi(data.Id)
		userInfo.ID = uint(val)
		userInfo.Email = data.Email
		userInfo.Phone = data.Phone
	}

	return userInfo, nil
}

func (u *User) GeneratePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (u *User) GenerateRandomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}
