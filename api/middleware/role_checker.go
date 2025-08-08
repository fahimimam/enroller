package middleware

import (
	"github.com/triapex/auth/internal/service"
	"github.com/triapex/auth/utils"
	"net/http"
)

// RoleAuthorizationMiddleware checks if the user has the required roles to access the endpoint
func RoleAuthorizationMiddleware(allowedRoles []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user claims from the context set by the AuthMiddleware
			user, ok := r.Context().Value(UserContextKey).(*service.CustomClaims)
			if !ok || user == nil {
				http.Error(w, "unauthenticated user or invalid context", http.StatusUnauthorized)
				return
			}

			// Check if the user has at least one of the required roles
			if !hasRequiredRole(user.Roles, allowedRoles) {
				http.Error(w, "forbidden: you do not have permission to access this resource", http.StatusForbidden)
				return
			}

			r.Header.Set(utils.UserIDHeader, user.ID)

			// If the user has the required role, proceed to the next middleware or handler
			next.ServeHTTP(w, r)
		})
	}
}

// hasRequiredRole determines whether the user roles overlap with the allowed roles
func hasRequiredRole(userRoles, allowedRoles []string) bool {
	roleSet := make(map[string]bool)
	for _, role := range userRoles {
		roleSet[role] = true
	}

	for _, allowedRole := range allowedRoles {
		if _, exists := roleSet[allowedRole]; exists {
			return true
		}
	}

	return false
}
