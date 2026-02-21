package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
	"go.uber.org/zap"
)

// Claims represents the custom JWT claims used by the application.
type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role,omitempty"`
	Type string `json:"type"` // "access" or "refresh"
}

// JWTService handles generation, validation, and revocation of JWT tokens.
type JWTService struct {
	secret          string
	accessDuration  time.Duration
	refreshDuration time.Duration
	cache           ports.Cache
	log             *zap.Logger
}

// NewJWTService creates a new JWTService instance.
func NewJWTService(secret string, accessDuration, refreshDuration time.Duration, cache ports.Cache, log *zap.Logger) *JWTService {
	log.Info("JWT service initialized",
		zap.Duration("access_duration", accessDuration),
		zap.Duration("refresh_duration", refreshDuration),
	)

	return &JWTService{
		secret:          secret,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
		cache:           cache,
		log:             log,
	}
}

// GenerateAccessToken creates a signed JWT access token for the given user.
// The token includes sub (user ID), role, exp, type="access", and jti (unique ID).
func (s *JWTService) GenerateAccessToken(user *domain.User) (string, error) {
	jti := uuid.New().String()
	now := time.Now()

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
		Role: string(user.Role),
		Type: "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.secret))
	if err != nil {
		s.log.Error("failed to sign access token",
			zap.String("user_id", user.ID),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	s.log.Debug("access token generated",
		zap.String("user_id", user.ID),
		zap.String("jti", jti),
	)

	return signedToken, nil
}

// GenerateRefreshToken creates a signed JWT refresh token for the given user.
// The token includes sub (user ID), exp, type="refresh", and jti (unique ID).
func (s *JWTService) GenerateRefreshToken(user *domain.User) (string, error) {
	jti := uuid.New().String()
	now := time.Now()

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
		Type: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.secret))
	if err != nil {
		s.log.Error("failed to sign refresh token",
			zap.String("user_id", user.ID),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	s.log.Debug("refresh token generated",
		zap.String("user_id", user.ID),
		zap.String("jti", jti),
	)

	return signedToken, nil
}

// ValidateToken parses and validates a JWT token string, returning the claims
// if the token is valid and has not been revoked.
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		s.log.Debug("token validation failed", zap.Error(err))
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	s.log.Debug("token validated",
		zap.String("subject", claims.Subject),
		zap.String("type", claims.Type),
		zap.String("jti", claims.ID),
	)

	return claims, nil
}

// RevokeToken stores the token ID in the Redis cache with a TTL, effectively
// blacklisting it until it would have naturally expired.
func (s *JWTService) RevokeToken(ctx context.Context, tokenID string) error {
	key := fmt.Sprintf("revoked_token:%s", tokenID)

	// Store with a TTL equal to the longer of the two token durations
	// to ensure the revocation outlasts any valid token.
	ttl := s.refreshDuration
	if s.accessDuration > ttl {
		ttl = s.accessDuration
	}

	err := s.cache.Set(ctx, key, "revoked", ttl)
	if err != nil {
		s.log.Error("failed to revoke token",
			zap.String("token_id", tokenID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	s.log.Info("token revoked",
		zap.String("token_id", tokenID),
	)

	return nil
}

// IsTokenRevoked checks whether a token ID has been revoked by looking it up
// in the Redis cache.
func (s *JWTService) IsTokenRevoked(ctx context.Context, tokenID string) bool {
	key := fmt.Sprintf("revoked_token:%s", tokenID)

	val, err := s.cache.Get(ctx, key)
	if err != nil {
		// If the key does not exist or there is an error, treat as not revoked.
		return false
	}

	return val == "revoked"
}
