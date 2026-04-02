package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token has expired")
	ErrMissingEnvVars      = errors.New("missing required environment variables")
	ErrInvalidPasswordHash = errors.New("ADMIN_PASSWORD must be a valid bcrypt hash. Generate one using: cd server/scripts && go run hash-password.go yourPassword")
)

type Service struct {
	jwtSecret         []byte
	adminUsername     string
	adminPasswordHash string
	sha256Salt        string
	tokenExpiration   time.Duration
	disabled          bool
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// NewService creates a new auth service
// Returns a disabled service (no error) if auth environment variables are not set.
func NewService() (*Service, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPasswordHash := os.Getenv("ADMIN_PASSWORD")
	sha256Salt := os.Getenv("ADMIN_PASSWORD_SALT")

	// If none of the auth variables are set, return nil to indicate auth is disabled
	if jwtSecret == "" && adminUsername == "" && (adminPasswordHash == "" && sha256Salt == "") {
		return NewDisabledService(), nil
	}

	// If some but not all are set, return an error
	if jwtSecret == "" || adminUsername == "" || (adminPasswordHash == "" && sha256Salt == "") {
		return nil, ErrMissingEnvVars
	}

	return &Service{
		jwtSecret:         []byte(jwtSecret),
		adminUsername:     adminUsername,
		adminPasswordHash: adminPasswordHash,
		sha256Salt:        sha256Salt,
		tokenExpiration:   7 * 24 * time.Hour, // 7 days
		disabled:          false,
	}, nil
}

func NewDisabledService() *Service {
	return &Service{disabled: true}
}

func (s *Service) IsDisabled() bool {
	return s != nil && s.disabled
}

func (s *Service) IsUsable() bool {
	return s != nil && !s.disabled && len(s.jwtSecret) > 0 && s.adminUsername != "" && s.adminPasswordHash != ""
}

// ValidateCredentials checks if the provided credentials are valid
func (s *Service) ValidateCredentials(username, password string) error {
	if !s.IsUsable() {
		return ErrInvalidCredentials
	}

	if username != s.adminUsername {
		return ErrInvalidCredentials
	}

	if isBcryptHash(s.adminPasswordHash) {
		if err := bcrypt.CompareHashAndPassword([]byte(s.adminPasswordHash), []byte(password)); err != nil {
			return ErrInvalidCredentials
		}
		return nil
	}

	if s.sha256Salt == "" {
		return ErrInvalidCredentials
	}

	hash := sha256.Sum256([]byte(password + s.sha256Salt))
	if hex.EncodeToString(hash[:]) == s.adminPasswordHash {
		return nil
	}

	return ErrInvalidCredentials
}

// GenerateToken creates a new JWT token for the user
func (s *Service) GenerateToken(username string) (string, error) {
	now := time.Now()
	expirationTime := now.Add(s.tokenExpiration)

	claims := &Claims{
		Username: username,
		Role:     "admin", // For now, all users are admins
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "vps-monitor",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// VerifyToken validates a JWT token and returns the claims
func (s *Service) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserFromClaims extracts user information from claims
func GetUserFromClaims(claims *Claims) models.User {
	return models.User{
		Username: claims.Username,
		Role:     claims.Role,
	}
}

// NewServiceFromFileConfig creates an auth service from file-based config.
// Returns nil if the config is nil, disabled, or incomplete.
func NewServiceFromFileConfig(cfg *config.FileAuthConfig) *Service {
	if cfg == nil || !cfg.Enabled {
		return NewDisabledService()
	}
	if cfg.JWTSecret == "" || cfg.AdminUsername == "" || cfg.AdminPasswordHash == "" {
		return nil
	}
	return &Service{
		jwtSecret:         []byte(cfg.JWTSecret),
		adminUsername:     cfg.AdminUsername,
		adminPasswordHash: cfg.AdminPasswordHash,
		sha256Salt:        cfg.AdminPasswordSalt,
		tokenExpiration:   7 * 24 * time.Hour,
		disabled:          false,
	}
}

// HashPasswordSHA256 computes SHA256(password + salt) and returns the hex-encoded hash.
func HashPasswordSHA256(password, salt string) string {
	h := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(h[:])
}

// GenerateRandomHex generates a cryptographically random hex string of the specified byte length.
func GenerateRandomHex(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashPassword generates a bcrypt hash from a plain password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func isBcryptHash(hash string) bool {
	return len(hash) >= 4 && hash[0] == '$' && hash[1] == '2'
}
