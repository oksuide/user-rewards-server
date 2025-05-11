package auth

import (
	"Test/internal/model"
	"Test/internal/repository"
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type JWTService struct {
	secretKey string
	duration  time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTService(secretKey string, duration time.Duration) *JWTService {
	return &JWTService{secretKey: secretKey, duration: duration}
}

func (s *JWTService) GenerateToken(userID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.duration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

type AuthService struct {
	repo       repository.UserRepository
	jwtService *JWTService
}

func NewAuthService(repo repository.UserRepository, jwt *JWTService) *AuthService {
	return &AuthService{repo: repo, jwtService: jwt}
}

func (s *AuthService) Register(ctx context.Context, username, password, email string) error {
	if err := validatePassword(password); err != nil {
		return err
	}

	exists, err := s.repo.EmailExists(ctx, email)
	if err != nil {
		return fmt.Errorf("check email existence: %w", err)
	}
	if exists {
		return fmt.Errorf("email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:        uuid.NewString(),
		Name:      username,
		Email:     email,
		Password:  string(hash),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("get user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return token, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}
