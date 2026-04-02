package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateCredentialsSupportsBcrypt(t *testing.T) {
	hash, err := HashPassword("super-secret")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	svc := &Service{
		jwtSecret:         []byte("jwt-secret"),
		adminUsername:     "admin",
		adminPasswordHash: hash,
		tokenExpiration:   time.Hour,
	}

	if err := svc.ValidateCredentials("admin", "super-secret"); err != nil {
		t.Fatalf("expected credentials to validate, got %v", err)
	}
	if err := svc.ValidateCredentials("admin", "wrong"); err == nil {
		t.Fatalf("expected invalid credentials error for wrong password")
	}
}

func TestValidateCredentialsRejectsNonBcryptHashes(t *testing.T) {
	svc := &Service{
		jwtSecret:         []byte("jwt-secret"),
		adminUsername:     "admin",
		adminPasswordHash: "legacy-not-bcrypt-hash",
		tokenExpiration:   time.Hour,
	}

	if err := svc.ValidateCredentials("admin", "legacy-pass"); err == nil {
		t.Fatalf("expected non-bcrypt password hash to be rejected")
	}
}

func TestDynamicMiddlewareFailsClosedWhenUnavailable(t *testing.T) {
	handler := DynamicMiddleware(func() *Service { return nil })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestDynamicMiddlewareBypassesExplicitDisabledService(t *testing.T) {
	handler := DynamicMiddleware(func() *Service { return NewDisabledService() })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDynamicMiddlewareUsesLastKnownGoodService(t *testing.T) {
	hash, err := HashPassword("super-secret")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	svc := &Service{
		jwtSecret:         []byte("jwt-secret"),
		adminUsername:     "admin",
		adminPasswordHash: hash,
		tokenExpiration:   time.Hour,
	}
	token, err := svc.GenerateToken("admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	currentSvc := svc
	handler := DynamicMiddleware(func() *Service { return currentSvc })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first request to pass, got %d", rec.Code)
	}

	currentSvc = nil
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected fallback request to pass, got %d", rec2.Code)
	}
}

func TestDynamicMiddlewareNoLastGoodServiceReturnsServiceUnavailable(t *testing.T) {
	handler := DynamicMiddleware(func() *Service { return nil })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}
