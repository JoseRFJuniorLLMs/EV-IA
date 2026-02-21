package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/mocks"
)

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestLogin_Success(t *testing.T) {
	ctx := context.Background()
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	mockUser := &domain.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Document: "12345678901",
		Password: string(hashedPassword),
		Role:     domain.UserRoleUser,
		Status:   "Active",
	}

	mockRepo := &mocks.MockUserRepository{
		FindByDocumentFunc: func(ctx context.Context, document string) (*domain.User, error) {
			if document == "12345678901" {
				return mockUser, nil
			}
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	accessToken, refreshToken, err := service.Login(ctx, "12345678901", password)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if accessToken == "" {
		t.Error("expected access token, got empty string")
	}
	if refreshToken == "" {
		t.Error("expected refresh token, got empty string")
	}
}

func TestLogin_InvalidCPF(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{
		FindByDocumentFunc: func(ctx context.Context, document string) (*domain.User, error) {
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	_, _, err := service.Login(ctx, "00000000000", "password")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("expected 'invalid credentials', got '%s'", err.Error())
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	mockUser := &domain.User{
		ID:       "user-123",
		Document: "12345678901",
		Password: string(hashedPassword),
	}

	mockRepo := &mocks.MockUserRepository{
		FindByDocumentFunc: func(ctx context.Context, document string) (*domain.User, error) {
			return mockUser, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	_, _, err := service.Login(ctx, "12345678901", "wrongpassword")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("expected 'invalid credentials', got '%s'", err.Error())
	}
}

func TestLogin_RepositoryError(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{
		FindByDocumentFunc: func(ctx context.Context, document string) (*domain.User, error) {
			return nil, errors.New("database error")
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	_, _, err := service.Login(ctx, "12345678901", "password")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegister_Success(t *testing.T) {
	ctx := context.Background()
	var savedUser *domain.User

	mockRepo := &mocks.MockUserRepository{
		SaveFunc: func(ctx context.Context, user *domain.User) error {
			savedUser = user
			return nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	newUser := &domain.User{
		ID:       "new-user-123",
		Name:     "Test User",
		Email:    "new@example.com",
		Document: "12345678901",
		Password: "password123",
	}

	err := service.Register(ctx, newUser)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if savedUser == nil {
		t.Fatal("expected user to be saved")
	}
	if savedUser.Password == "password123" {
		t.Error("password should be hashed, not plain text")
	}
	if savedUser.Role != domain.UserRoleUser {
		t.Errorf("expected default role 'user', got '%s'", savedUser.Role)
	}
	if savedUser.Status != "Active" {
		t.Errorf("expected status 'Active', got '%s'", savedUser.Status)
	}
}

func TestRegister_RepositoryError(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{
		SaveFunc: func(ctx context.Context, user *domain.User) error {
			return errors.New("database error")
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	newUser := &domain.User{
		ID:       "new-user-123",
		Email:    "new@example.com",
		Password: "password123",
	}

	err := service.Register(ctx, newUser)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateToken_Success(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "test-secret-key"

	mockUser := &domain.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  domain.UserRoleUser,
	}

	mockRepo := &mocks.MockUserRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			if id == "user-123" {
				return mockUser, nil
			}
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, jwtSecret, newTestLogger())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-123",
		"role": "user",
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"type": "access",
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	user, err := service.ValidateToken(ctx, tokenStr)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.ID != "user-123" {
		t.Errorf("expected user ID 'user-123', got '%s'", user.ID)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{}
	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	_, err := service.ValidateToken(ctx, "invalid-token")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "test-secret-key"

	mockRepo := &mocks.MockUserRepository{}
	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, jwtSecret, newTestLogger())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-123",
		"role": "user",
		"exp":  time.Now().Add(-1 * time.Hour).Unix(),
		"type": "access",
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	_, err := service.ValidateToken(ctx, tokenStr)

	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestRefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "test-secret-key"

	mockUser := &domain.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  domain.UserRoleUser,
	}

	mockRepo := &mocks.MockUserRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			if id == "user-123" {
				return mockUser, nil
			}
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, jwtSecret, newTestLogger())

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-123",
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": "refresh",
	})
	refreshTokenStr, _ := refreshToken.SignedString([]byte(jwtSecret))

	newAccessToken, err := service.RefreshToken(ctx, refreshTokenStr)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if newAccessToken == "" {
		t.Error("expected new access token, got empty string")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{}
	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	_, err := service.RefreshToken(ctx, "invalid-refresh-token")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "test-secret-key"

	mockRepo := &mocks.MockUserRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, jwtSecret, newTestLogger())

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "nonexistent-user",
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": "refresh",
	})
	refreshTokenStr, _ := refreshToken.SignedString([]byte(jwtSecret))

	_, err := service.RefreshToken(ctx, refreshTokenStr)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
