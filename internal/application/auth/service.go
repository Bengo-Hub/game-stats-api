package auth

import (
	"context"
	"errors"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/config"
	"github.com/bengobox/game-stats-api/internal/domain/user"
	"github.com/bengobox/game-stats-api/internal/pkg/auth"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	userRepo user.Repository
	cfg      *config.Config
}

func NewService(userRepo user.Repository, cfg *config.Config) *Service {
	return &Service{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	u, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !auth.CheckPasswordHash(req.Password, u.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := auth.GenerateToken(s.cfg.JWTSecret, auth.TokenPayload{
		UserID: u.ID,
		Role:   u.Role,
	}, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateToken(s.cfg.JWTSecret, auth.TokenPayload{
		UserID: u.ID,
		Role:   u.Role,
	}, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserDTO{
			ID:    u.ID,
			Email: u.Email,
			Name:  u.Name,
			Role:  u.Role,
		},
	}, nil
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (*TokenResponse, error) {
	claims, err := auth.ValidateToken(s.cfg.JWTSecret, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	accessToken, err := auth.GenerateToken(s.cfg.JWTSecret, auth.TokenPayload{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateToken(s.cfg.JWTSecret, auth.TokenPayload{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GetUserByID retrieves a user by their ID
func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*UserDTO, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &UserDTO{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
		Role:  u.Role,
	}, nil
}
