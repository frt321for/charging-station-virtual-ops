package aiops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ModelScopeGenerator calls a ModelScope OpenAI-compatible chat endpoint.
type ModelScopeGenerator struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
}

// NewModelScopeGenerator creates a chat-completion generator when an API key is configured.
func NewModelScopeGenerator(baseURL string, model string, apiKey string) Generator {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api-inference.modelscope.cn/v1"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "deepseek-ai/DeepSeek-V3.2"
	}
	return &ModelScopeGenerator{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 35 * time.Second},
	}
}

// Generate sends a chat-completion request and returns the first message.
func (g *ModelScopeGenerator) Generate(ctx context.Context, prompt Prompt) (GeneratedContent, error) {
	body, err := json.Marshal(chatRequest{
		Model: g.model,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.System},
			{Role: "user", Content: prompt.User},
		},
		Temperature: 0.2,
		MaxTokens:   900,
	})
	if err != nil {
		return GeneratedContent{}, fmt.Errorf("marshal ai request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return GeneratedContent{}, fmt.Errorf("create ai request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+g.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := g.client.Do(request)
	if err != nil {
		return GeneratedContent{}, fmt.Errorf("call ai provider: %w", err)
	}
	defer response.Body.Close()

	var envelope chatResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return GeneratedContent{}, fmt.Errorf("decode ai response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return GeneratedContent{}, fmt.Errorf("ai provider status %d", response.StatusCode)
	}
	if len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" {
		return GeneratedContent{}, fmt.Errorf("ai provider returned empty content")
	}

	return GeneratedContent{
		Content:  strings.TrimSpace(envelope.Choices[0].Message.Content),
		Provider: "modelscope",
		Model:    g.model,
	}, nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
