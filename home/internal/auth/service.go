package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// NewService creates a new auth service
// Returns nil (no error) if auth environment variables are not set, indicating auth is disabled
func NewService() (*Service, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPasswordHash := os.Getenv("ADMIN_PASSWORD")
	sha256Salt := os.Getenv("ADMIN_PASSWORD_SALT")

	// If none of the auth variables are set, return nil to indicate auth is disabled
	if jwtSecret == "" && adminUsername == "" && (adminPasswordHash == "" && sha256Salt == "") {
		return nil, nil
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
	}, nil
}

// ValidateCredentials checks if the provided credentials are valid
func (s *Service) ValidateCredentials(username, password string) error {
	if username != s.adminUsername {
		return ErrInvalidCredentials
	}

	hash := sha256.Sum256([]byte(password + s.sha256Salt))
	if hex.EncodeToString(hash[:]) != s.adminPasswordHash {
		return ErrInvalidCredentials
	}

	return nil
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

// HashPassword generates a bcrypt hash from a plain password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
