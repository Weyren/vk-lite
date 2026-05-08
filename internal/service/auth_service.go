// internal/service/auth_service.go
package service

import (
	"context"
	"time"

	"github.com/Weyren/vk-lite/internal/repo"
	"github.com/Weyren/vk-lite/pkg/models"
	"github.com/Weyren/vk-lite/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo repo.UserRepo
	secret   []byte
	ttl      time.Duration
}

func NewAuthService(ur repo.UserRepo, cfg *utils.Config) *AuthService {
	return &AuthService{
		userRepo: ur,
		secret:   []byte(cfg.JWTSecret),
		ttl:      cfg.JWTTTL,
	}
}

// Register создаёт запись в БД
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login проверяет пароль и выдаёт JWT
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"sub": u.ID,
		"exp": time.Now().Add(s.ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}
