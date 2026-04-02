package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
)

type contextKey string

const UserContextKey contextKey = "user"

// DynamicMiddleware resolves the auth service per request, supporting hot-reload.
// It fails closed when auth service is unavailable and only bypasses auth when explicitly disabled.
func DynamicMiddleware(getService func() *Service) func(http.Handler) http.Handler {
	var mu sync.RWMutex
	var lastGoodSvc *Service

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			svc := getService()

			if svc != nil && svc.IsDisabled() {
				next.ServeHTTP(w, r)
				return
			}

			if svc != nil && svc.IsUsable() {
				mu.Lock()
				lastGoodSvc = svc
				mu.Unlock()
				validateAndServe(svc, next, w, r)
				return
			}

			mu.RLock()
			fallback := lastGoodSvc
			mu.RUnlock()
			if fallback != nil && fallback.IsUsable() {
				validateAndServe(fallback, next, w, r)
				return
			}

			http.Error(w, "Authentication service unavailable", http.StatusServiceUnavailable)
		})
	}
}

// Middleware creates an authentication middleware with a fixed auth service.
func Middleware(authService *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			validateAndServe(authService, next, w, r)
		})
	}
}

// validateAndServe extracts JWT, validates it, adds user to context, and calls next.
func validateAndServe(svc *Service, next http.Handler, w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	var tokenString string

	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString = parts[1]
	} else {
		tokenString = r.URL.Query().Get("token")
		if tokenString == "" {
			http.Error(w, "Authorization header or token query parameter required", http.StatusUnauthorized)
			return
		}
	}

	claims, err := svc.VerifyToken(tokenString)
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
}
