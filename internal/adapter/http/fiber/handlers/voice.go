package handlers

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/service/voice"
)

type VoiceHandler struct {
	assistant *voice.VoiceAssistant
	log       *zap.Logger
}

func NewVoiceHandler(assistant *voice.VoiceAssistant, log *zap.Logger) *VoiceHandler {
	return &VoiceHandler{
		assistant: assistant,
		log:       log,
	}
}

type AudioCommandRequest struct {
	Audio     string `json:"audio"` // Base64
	SessionID string `json:"session_id"`
}

func (h *VoiceHandler) ProcessCommand(c *fiber.Ctx) error {
	var req AudioCommandRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}

	userID := c.Locals("user_id").(string)

	audioBytes, err := base64.StdEncoding.DecodeString(req.Audio)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid base64 audio"})
	}

	resp, err := h.assistant.ProcessVoiceCommand(c.Context(), userID, audioBytes)
	if err != nil {
		h.log.Error("Failed to process voice command", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process voice command"})
	}

	return c.JSON(resp)
}

func (h *VoiceHandler) GetHistory(c *fiber.Ctx) error {
	// Not implemented in this snippet
	return c.JSON([]string{})
}
