package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/triapex/auth/api/response"
	"github.com/triapex/auth/internal/service"
	"net/http"
	"strings"
)

// Key to store user information in the request context
type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware is a middleware to authenticate and authorize users
func AuthMiddleware(publicKey *rsa.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the token from the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				_ = response.ServeJSON(w, http.StatusUnauthorized, "missing authorization header", nil)
				return
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
				_ = response.ServeJSON(w, http.StatusUnauthorized, "invalid authorization header", nil)
				return
			}

			tokenString := tokenParts[1]

			// Parse the token with the public key for verification
			token, err := jwt.ParseWithClaims(tokenString, &service.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
				// Validate the signing method
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return publicKey, nil
			})

			if err != nil {
				_ = response.ServeJSON(w, http.StatusUnauthorized, "invalid token: "+err.Error(), nil)
				return
			}

			// Extract claims if the token is valid
			claims, ok := token.Claims.(*service.CustomClaims)
			if !ok || !token.Valid {
				_ = response.ServeJSON(w, http.StatusUnauthorized, "invalid token claims", nil)
				return
			}

			// Store user data in context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
