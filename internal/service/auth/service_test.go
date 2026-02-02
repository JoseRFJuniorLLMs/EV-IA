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
	// Arrange
	ctx := context.Background()
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	mockUser := &domain.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Password: string(hashedPassword),
		Role:     domain.UserRoleUser,
		Status:   "Active",
	}

	mockRepo := &mocks.MockUserRepository{
		FindByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			if email == "test@example.com" {
				return mockUser, nil
			}
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	// Act
	accessToken, refreshToken, err := service.Login(ctx, "test@example.com", password)

	// Assert
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

func TestLogin_InvalidEmail(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{
		FindByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil // User not found
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	// Act
	_, _, err := service.Login(ctx, "notfound@example.com", "password")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("expected 'invalid credentials', got '%s'", err.Error())
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	// Arrange
	ctx := context.Background()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	mockUser := &domain.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Password: string(hashedPassword),
	}

	mockRepo := &mocks.MockUserRepository{
		FindByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return mockUser, nil
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	// Act
	_, _, err := service.Login(ctx, "test@example.com", "wrongpassword")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("expected 'invalid credentials', got '%s'", err.Error())
	}
}

func TestLogin_RepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{
		FindByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, errors.New("database error")
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	// Act
	_, _, err := service.Login(ctx, "test@example.com", "password")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegister_Success(t *testing.T) {
	// Arrange
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
		Password: "password123",
	}

	// Act
	err := service.Register(ctx, newUser)

	// Assert
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
	// Arrange
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

	// Act
	err := service.Register(ctx, newUser)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateToken_Success(t *testing.T) {
	// Arrange
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

	// Create a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-123",
		"role": "user",
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"type": "access",
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	// Act
	user, err := service.ValidateToken(ctx, tokenStr)

	// Assert
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
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{}
	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	// Act
	_, err := service.ValidateToken(ctx, "invalid-token")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Arrange
	ctx := context.Background()
	jwtSecret := "test-secret-key"

	mockRepo := &mocks.MockUserRepository{}
	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, jwtSecret, newTestLogger())

	// Create an expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-123",
		"role": "user",
		"exp":  time.Now().Add(-1 * time.Hour).Unix(), // Expired
		"type": "access",
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	// Act
	_, err := service.ValidateToken(ctx, tokenStr)

	// Assert
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestRefreshToken_Success(t *testing.T) {
	// Arrange
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

	// Create a valid refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-123",
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": "refresh",
	})
	refreshTokenStr, _ := refreshToken.SignedString([]byte(jwtSecret))

	// Act
	newAccessToken, err := service.RefreshToken(ctx, refreshTokenStr)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if newAccessToken == "" {
		t.Error("expected new access token, got empty string")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockUserRepository{}
	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, "test-secret-key", newTestLogger())

	// Act
	_, err := service.RefreshToken(ctx, "invalid-refresh-token")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	jwtSecret := "test-secret-key"

	mockRepo := &mocks.MockUserRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return nil, nil // User not found
		},
	}

	mockCache := mocks.NewMockCache()
	service := NewService(mockRepo, mockCache, jwtSecret, newTestLogger())

	// Create a valid refresh token for non-existent user
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "nonexistent-user",
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": "refresh",
	})
	refreshTokenStr, _ := refreshToken.SignedString([]byte(jwtSecret))

	// Act
	_, err := service.RefreshToken(ctx, refreshTokenStr)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
