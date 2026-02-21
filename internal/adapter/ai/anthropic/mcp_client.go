package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// MCPClient provides access to Anthropic's API via the Model Context Protocol
type MCPClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	log        *zap.Logger
}

// NewMCPClient creates a new Anthropic MCP client
func NewMCPClient(apiKey string, log *zap.Logger) *MCPClient {
	return &MCPClient{
		apiKey:     apiKey,
		model:      "claude-sonnet-4-20250514",
		httpClient: &http.Client{},
		log:        log,
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// SendMessage sends a message to the Anthropic API and returns the response
func (c *MCPClient) SendMessage(ctx context.Context, message string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("anthropic: API key not configured")
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 1024,
		Messages: []anthropicMessage{
			{Role: "user", Content: message},
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("anthropic: create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic: API error status %d: %s", resp.StatusCode, string(body))
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("anthropic: decode response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("anthropic: no content returned")
	}

	c.log.Info("Anthropic message completed",
		zap.Int("input_tokens", result.Usage.InputTokens),
		zap.Int("output_tokens", result.Usage.OutputTokens),
	)

	return result.Content[0].Text, nil
}

// StreamMessage sends a message and streams the response via callback
func (c *MCPClient) StreamMessage(ctx context.Context, message string, callback func(chunk string)) error {
	if c.apiKey == "" {
		return fmt.Errorf("anthropic: API key not configured")
	}

	reqBody := map[string]interface{}{
		"model":      c.model,
		"max_tokens": 1024,
		"stream":     true,
		"messages": []anthropicMessage{
			{Role: "user", Content: message},
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("anthropic: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("anthropic: create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic: API error status %d: %s", resp.StatusCode, string(body))
	}

	// Read SSE stream
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			callback(event.Delta.Text)
		}
	}

	return nil
}
