package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/seu-repo/sigec-ve/internal/adapter/ai/gemini"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
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
		devices, err := va.deviceService.ListAvailableDevices(ctx)
		if err != nil {
			va.logger.Error("Failed to list available devices", zap.Error(err))
			return "Desculpe, não consegui verificar os carregadores disponíveis no momento."
		}
		if len(devices) == 0 {
			return "No momento não há carregadores disponíveis. Por favor, tente novamente em alguns minutos."
		}
		return fmt.Sprintf("Existem %d carregadores disponíveis no momento.", len(devices))

	case "start_charge":
		stationID := ""
		if intent.Entities != nil {
			stationID = intent.Entities["station_id"]
		}
		tx, err := va.txService.StartCharging(ctx, userID, stationID)
		if err != nil {
			va.logger.Error("Failed to start charging", zap.Error(err), zap.String("user_id", userID))
			return fmt.Sprintf("Não foi possível iniciar o carregamento: %s", err.Error())
		}
		return fmt.Sprintf("Carregamento iniciado com sucesso! ID da sessão: %s. Você será notificado quando terminar.", tx.ID)

	case "stop_charge":
		err := va.txService.StopActiveCharging(ctx, userID)
		if err != nil {
			va.logger.Error("Failed to stop charging", zap.Error(err), zap.String("user_id", userID))
			return fmt.Sprintf("Não foi possível parar o carregamento: %s", err.Error())
		}
		return "Carregamento finalizado com sucesso! O valor será cobrado automaticamente."

	case "check_cost":
		cost, err := va.txService.GetCurrentSessionCost(ctx, userID)
		if err != nil {
			va.logger.Warn("Failed to get current session cost", zap.Error(err))
			return "Você não possui uma sessão de carregamento ativa no momento."
		}
		return fmt.Sprintf("O custo estimado da sua sessão atual é R$ %.2f.", cost)

	case "report_issue":
		// Log the issue for later processing
		va.logger.Info("User reported issue via voice",
			zap.String("user_id", userID),
			zap.String("issue_text", intent.Entities["issue_description"]),
		)
		return "Seu problema foi registrado. Nossa equipe de suporte entrará em contato em breve."

	default:
		return "Desculpe, não entendi o que você precisa. Você pode perguntar sobre carregadores disponíveis, iniciar ou parar um carregamento, ou verificar o custo da sessão atual."
	}
}

func (va *VoiceAssistant) extractEntities(text string) map[string]string {
	// Placeholder for entity extraction logic
	return make(map[string]string)
}
