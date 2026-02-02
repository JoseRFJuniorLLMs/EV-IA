package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/seu-repo/sigec-ve/internal/adapter/ai/gemini"
	"github.com/seu-repo/sigec-ve/internal/core/domain"
	"github.com/seu-repo/sigec-ve/internal/core/ports"
	"go.uber.org/zap"
)

type VoiceAssistant struct {
	gemini        *gemini.LiveClient
	deviceService ports.DeviceService
	txService     ports.TransactionService
	logger        *zap.Logger
}

func NewVoiceAssistant(
	gemini *gemini.LiveClient,
	deviceSvc ports.DeviceService,
	txSvc ports.TransactionService,
	logger *zap.Logger,
) *VoiceAssistant {
	return &VoiceAssistant{
		gemini:        gemini,
		deviceService: deviceSvc,
		txService:     txSvc,
		logger:        logger,
	}
}

// ProcessVoiceCommand processa comando de voz do usuário
func (va *VoiceAssistant) ProcessVoiceCommand(
	ctx context.Context,
	userID string,
	audioChunk []byte,
) (*domain.VoiceResponse, error) {

	// 1. Envia áudio para Gemini
	if err := va.gemini.SendAudioChunk(audioChunk); err != nil {
		return nil, err
	}

	// 2. Recebe resposta do Gemini
	geminiResp, err := va.gemini.ReceiveResponse(ctx)
	if err != nil {
		return nil, err
	}

	// 3. Extrai texto e áudio da resposta
	var responseText string
	var responseAudio []byte

	for _, part := range geminiResp.ServerContent.ModelTurn.Parts {
		if part.Text != "" {
			responseText = part.Text
		}
		if part.InlineData.MimeType == "audio/pcm" {
			responseAudio, _ = base64.StdEncoding.DecodeString(part.InlineData.Data)
		}
	}

	// 4. Parse de intenção (NLU simplificado)
	intent := va.parseIntent(responseText)

	// 5. Executa ação baseada na intenção
	actionResult := va.executeAction(ctx, userID, intent)

	return &domain.VoiceResponse{
		Text:         responseText,
		Audio:        responseAudio,
		Intent:       intent.Name,
		ActionResult: actionResult,
		Confidence:   intent.Confidence,
	}, nil
}

// parseIntent identifica a intenção do usuário
func (va *VoiceAssistant) parseIntent(text string) *domain.Intent {
	// Implementação de NLU básica
	// Em produção, usar modelo fine-tuned ou serviço como Dialogflow

	intents := map[string][]string{
		"check_status": {"status", "situação", "carregador", "disponível"},
		"start_charge": {"iniciar", "começar", "carregamento", "carregar"},
		"stop_charge":  {"parar", "interromper", "cancelar"},
		"check_cost":   {"custo", "preço", "valor", "quanto"},
		"report_issue": {"problema", "defeito", "não funciona", "erro"},
	}

	// Análise simples por palavras-chave
	for intentName, keywords := range intents {
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(text), keyword) {
				return &domain.Intent{
					Name:       intentName,
					Confidence: 0.85,
					Entities:   va.extractEntities(text),
				}
			}
		}
	}

	return &domain.Intent{
		Name:       "unknown",
		Confidence: 0.0,
	}
}

// executeAction executa a ação identificada
func (va *VoiceAssistant) executeAction(
	ctx context.Context,
	userID string,
	intent *domain.Intent,
) string {

	switch intent.Name {
	case "check_status":
		devices, _ := va.deviceService.ListAvailableDevices(ctx)
		return fmt.Sprintf("Existem %d carregadores disponíveis no momento", len(devices))

	case "start_charge":
		stationID := intent.Entities["station_id"]
		tx, err := va.txService.StartCharging(ctx, userID, stationID)
		if err != nil {
			return "Não foi possível iniciar o carregamento. Verifique se há um carregador disponível."
		}
		return fmt.Sprintf("Carregamento iniciado com sucesso! ID da sessão: %s", tx.ID)

	case "stop_charge":
		err := va.txService.StopActiveCharging(ctx, userID)
		if err != nil {
			return "Não foi possível parar o carregamento."
		}
		return "Carregamento finalizado com sucesso!"

	case "check_cost":
		cost, _ := va.txService.GetCurrentSessionCost(ctx, userID)
		return fmt.Sprintf("O custo atual da sua sessão é R$ %.2f", cost)

	default:
		return "Desculpe, não entendi o que você precisa. Pode repetir?"
	}
}

func (va *VoiceAssistant) extractEntities(text string) map[string]string {
	// Placeholder for entity extraction logic
	return make(map[string]string)
}
