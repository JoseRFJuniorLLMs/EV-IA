package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

type LiveClient struct {
	apiKey  string
	modelID string
	logger  *zap.Logger
	conn    *websocket.Conn
}

type VoiceConfig struct {
	Voice            string `json:"voice"`             // "Puck", "Charon", "Kore", "Fenrir", "Aoede"
	Language         string `json:"language"`          // "pt-BR"
	SpeechModel      string `json:"speech_model"`      // "gemini-2.0-flash-exp"
	ResponseModality string `json:"response_modality"` // "AUDIO"
}

func NewLiveClient(apiKey string, logger *zap.Logger) *LiveClient {
	return &LiveClient{
		apiKey:  apiKey,
		modelID: "gemini-2.0-flash-exp",
		logger:  logger,
	}
}

// ConnectVoiceStream estabelece conexão bidirecional com Gemini Live API
func (c *LiveClient) ConnectVoiceStream(ctx context.Context) error {
	url := "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContent"

	headers := http.Header{
		"Content-Type": []string{"application/json"},
	}

	conn, _, err := websocket.Dial(ctx, url+"?key="+c.apiKey, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return err
	}

	c.conn = conn

	// Enviar setup inicial
	setup := map[string]interface{}{
		"setup": map[string]interface{}{
			"model": "models/" + c.modelID,
			"generation_config": map[string]interface{}{
				"response_modalities": []string{"AUDIO"},
				"speech_config": map[string]interface{}{
					"voice_config": map[string]interface{}{
						"prebuilt_voice_config": map[string]string{
							"voice_name": "Aoede",
						},
					},
				},
			},
			"system_instruction": map[string]interface{}{
				"parts": []map[string]string{
					{
						"text": `Você é um assistente virtual para estações de carregamento de veículos elétricos.
                        Seu nome é EVA (Electric Vehicle Assistant).
                        Você ajuda usuários a:
                        - Verificar status de carregadores
                        - Iniciar/parar sessões de carregamento
                        - Consultar histórico e custos
                        - Reportar problemas
                        - Agendar carregamentos
                        
                        Seja profissional, clara e objetiva. Fale em português brasileiro.`,
					},
				},
			},
		},
	}

	return c.send(setup)
}

// SendAudioChunk envia áudio PCM16 para o Gemini
func (c *LiveClient) SendAudioChunk(audioData []byte) error {
	msg := map[string]interface{}{
		"realtime_input": map[string]interface{}{
			"media_chunks": []map[string]string{
				{
					"mime_type": "audio/pcm",
					"data":      base64.StdEncoding.EncodeToString(audioData),
				},
			},
		},
	}

	return c.send(msg)
}

// ReceiveResponse recebe resposta de voz do Gemini
func (c *LiveClient) ReceiveResponse(ctx context.Context) (*VoiceResponse, error) {
	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return nil, err
	}

	var response VoiceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *LiveClient) send(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return c.conn.Write(context.Background(), websocket.MessageText, data)
}

type VoiceResponse struct {
	ServerContent struct {
		ModelTurn struct {
			Parts []struct {
				Text       string `json:"text,omitempty"`
				InlineData struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"` // Base64 audio
				} `json:"inlineData,omitempty"`
			} `json:"parts"`
		} `json:"modelTurn"`
		TurnComplete bool `json:"turnComplete"`
	} `json:"serverContent"`
}
