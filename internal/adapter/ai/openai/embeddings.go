package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Client provides access to OpenAI APIs for embeddings and chat completions
type Client struct {
	apiKey     string
	httpClient *http.Client
	log        *zap.Logger
}

// NewClient creates a new OpenAI API client
func NewClient(apiKey string, log *zap.Logger) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		log:        log,
	}
}

// --- Embeddings ---

type embeddingsRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// GetEmbeddings generates embeddings for the given texts using text-embedding-3-small
func (c *Client) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("openai: API key not configured")
	}

	reqBody := embeddingsRequest{
		Input: texts,
		Model: "text-embedding-3-small",
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: API error status %d", resp.StatusCode)
	}

	var result embeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai: decode response: %w", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		embeddings[d.Index] = d.Embedding
	}

	c.log.Info("Generated embeddings",
		zap.Int("count", len(texts)),
		zap.Int("total_tokens", result.Usage.TotalTokens),
	)

	return embeddings, nil
}

// --- Chat Completion ---

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// ChatCompletion sends a chat completion request to OpenAI
func (c *Client) ChatCompletion(ctx context.Context, messages []Message) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("openai: API key not configured")
	}

	reqBody := chatRequest{
		Model:    "gpt-4o-mini",
		Messages: messages,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("openai: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai: API error status %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("openai: decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices returned")
	}

	return result.Choices[0].Message.Content, nil
}
