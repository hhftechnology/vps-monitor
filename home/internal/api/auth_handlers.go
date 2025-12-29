package api

import (
	"encoding/json"
	"net/http"

	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

type AuthHandlers struct {
	authService *auth.Service
}

func NewAuthHandlers(authService *auth.Service) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
	}
}

func (ah *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var loginReq models.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if loginReq.Username == "" || loginReq.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if err := ah.authService.ValidateCredentials(loginReq.Username, loginReq.Password); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := ah.authService.GenerateToken(loginReq.Username)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := models.LoginResponse{
		Token: token,
		User: models.User{
			Username: loginReq.Username,
			Role:     "admin",
		},
	}

	WriteJsonResponse(w, http.StatusOK, response)
}

// GetMe returns the current authenticated user's information
func (ah *AuthHandlers) GetMe(w http.ResponseWriter, r *http.Request) {
	userValue := r.Context().Value(auth.UserContextKey)
	if userValue == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	user, ok := userValue.(models.User)
	if !ok {
		http.Error(w, "Invalid user data in context", http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}
