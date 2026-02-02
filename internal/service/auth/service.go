package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

type Service struct {
	userRepo  ports.UserRepository
	cache     ports.Cache
	jwtSecret []byte
	log       *zap.Logger
}

func NewService(userRepo ports.UserRepository, cache ports.Cache, jwtSecret string, log *zap.Logger) ports.AuthService {
	return &Service{
		userRepo:  userRepo,
		cache:     cache,
		jwtSecret: []byte(jwtSecret),
		log:       log,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", "", errors.New("invalid credentials")
	}
	if user == nil {
		return "", "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", errors.New("invalid credentials")
	}

	return s.generateTokens(user)
}

func (s *Service) Register(ctx context.Context, user *domain.User) error {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPwd)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	// Default role/status
	if user.Role == "" {
		user.Role = domain.UserRoleUser
	}
	user.Status = "Active"

	return s.userRepo.Save(ctx, user)
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Parse and validate refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid user id in token")
	}

	// Verify user exists and status
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return "", errors.New("user not found")
	}

	// Generate new access token
	accessToken, _, err := s.generateTokens(user) // Only assume we return access token here for simplicity or re-gen both
	// But generateTokens returns both. Let's just return access token as requested.
	// Actually typical flow might rotate refresh token too.
	// For this implementations, I'll allow re-using refresh token logic inside generateTokens but discard refresh?
	// Or better:

	accessTokenStr, err := s.generateAccessToken(user)
	return accessTokenStr, err
}

func (s *Service) ValidateToken(ctx context.Context, tokenStr string) (*domain.User, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid sub")
	}

	// Could cache user lookup here
	return s.userRepo.FindByID(ctx, userID)
}

func (s *Service) generateTokens(user *domain.User) (string, string, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": "refresh",
	})
	refreshTokenStr, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenStr, refreshTokenStr, nil
}

func (s *Service) generateAccessToken(user *domain.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"type": "access",
	})
	return token.SignedString(s.jwtSecret)
}
