package handlers

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

type AuthHandler struct {
	service ports.AuthService
	log     *zap.Logger
}

func NewAuthHandler(service ports.AuthService, log *zap.Logger) *AuthHandler {
	return &AuthHandler{
		service: service,
		log:     log,
	}
}

type LoginRequest struct {
	CPF      string `json:"cpf"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	CPF      string `json:"cpf"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.CPF == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "CPF and password are required"})
	}

	token, refreshToken, err := h.service.Login(c.Context(), req.CPF, req.Password)
	if err != nil {
		h.log.Warn("Login failed", zap.String("cpf", req.CPF), zap.Error(err))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	user, _ := h.service.ValidateToken(c.Context(), token)

	return c.JSON(fiber.Map{
		"tokens": fiber.Map{
			"accessToken":  token,
			"refreshToken": refreshToken,
		},
		"user": user,
	})
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.CPF == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "CPF is required"})
	}

	user := domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Document: req.CPF,
	}
	plainPassword := req.Password

	if err := h.service.Register(c.Context(), &user); err != nil {
		if err.Error() == "email already registered" || err.Error() == "cpf already registered" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Auto-login after registration using CPF
	token, refreshToken, err := h.service.Login(c.Context(), req.CPF, plainPassword)
	if err != nil {
		user.Password = ""
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user})
	}

	user.Password = ""
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": user,
		"tokens": fiber.Map{
			"accessToken":  token,
			"refreshToken": refreshToken,
		},
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	token, err := h.service.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"accessToken":  token,
		"refreshToken": req.RefreshToken,
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	user := c.Locals("user")
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Not authenticated"})
	}
	return c.JSON(user)
}
