package websocket

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/seu-repo/sigec-ve/internal/service/voice"
	"go.uber.org/zap"
)

type VoiceStreamHandler struct {
	assistant *voice.VoiceAssistant
	logger    *zap.Logger
}

func NewVoiceStreamHandler(assistant *voice.VoiceAssistant, logger *zap.Logger) *VoiceStreamHandler {
	return &VoiceStreamHandler{
		assistant: assistant,
		logger:    logger,
	}
}

// HandleVoiceStream gerencia o streaming bidirecional de voz
func (h *VoiceStreamHandler) HandleVoiceStream(c *websocket.Conn) {
	userID := c.Locals("user_id").(string)

	ctx := context.Background()

	for {
		// Recebe áudio do cliente (navegador)
		messageType, audioData, err := c.ReadMessage()
		if err != nil {
			h.logger.Error("Erro ao ler mensagem WebSocket", zap.Error(err))
			break
		}

		if messageType == websocket.BinaryMessage {
			// Processa áudio com Gemini
			response, err := h.assistant.ProcessVoiceCommand(ctx, userID, audioData)
			if err != nil {
				h.logger.Error("Erro ao processar comando de voz", zap.Error(err))
				continue
			}

			// Envia resposta de volta para o cliente
			responseJSON, _ := json.Marshal(map[string]interface{}{
				"text":   response.Text,
				"audio":  response.Audio, // Base64
				"intent": response.Intent,
				"result": response.ActionResult,
			})

			if err := c.WriteMessage(websocket.TextMessage, responseJSON); err != nil {
				h.logger.Error("Erro ao enviar resposta", zap.Error(err))
				break
			}
		}
	}
}

// SetupVoiceRoutes configura rotas de WebSocket para voz
func SetupVoiceRoutes(app *fiber.App, handler *VoiceStreamHandler) {
	app.Use("/ws/voice", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/voice", websocket.New(handler.HandleVoiceStream))
}
