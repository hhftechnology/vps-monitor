package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

// Middleware creates an authentication middleware
func Middleware(authService *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			var tokenString string

			if authHeader != "" {
				// Check if the header starts with "Bearer "
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) != 2 || parts[0] != "Bearer" {
					http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
					return
				}
				tokenString = parts[1]
			} else {
				// Try to get token from query parameter (useful for WebSockets)
				tokenString = r.URL.Query().Get("token")
				if tokenString == "" {
					http.Error(w, "Authorization header or token query parameter required", http.StatusUnauthorized)
					return
				}
			}

			claims, err := authService.VerifyToken(tokenString)
			if err != nil {
				if errors.Is(err, ErrTokenExpired) {
					http.Error(w, "Token has expired", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			user := GetUserFromClaims(claims)
			ctx := context.WithValue(r.Context(), UserContextKey, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
