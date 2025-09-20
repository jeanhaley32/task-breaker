package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jeanhaley/task-breaker/ai"
)

// OpenAIBackend implements the ai.Backend interface for OpenAI's API
type OpenAIBackend struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
}

// Config holds configuration for the OpenAI backend
type Config struct {
	APIKey     string        `json:"api_key"`
	BaseURL    string        `json:"base_url"`
	Model      string        `json:"model"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
}

// NewOpenAIBackend creates a new OpenAI backend instance
func NewOpenAIBackend(config Config) *OpenAIBackend {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.Model == "" {
		config.Model = "gpt-4"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &OpenAIBackend{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		model:   config.Model,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the name of this backend
func (o *OpenAIBackend) Name() string {
	return "OpenAI"
}

// ChatCompletion sends a chat completion request to OpenAI's API
func (o *OpenAIBackend) ChatCompletion(ctx context.Context, req ai.ChatCompletionRequest) (*ai.ChatCompletionResponse, error) {
	// Validate required fields
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}

	// Convert our request to OpenAI's format (they're the same, but we want to be explicit)
	openAIRequest := struct {
		Model       string       `json:"model"`
		Messages    []ai.Message `json:"messages"`
		MaxTokens   *int         `json:"max_tokens,omitempty"`
		Temperature *float64     `json:"temperature,omitempty"`
		TopP        *float64     `json:"top_p,omitempty"`
		Stream      bool         `json:"stream,omitempty"`
	}{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(openAIRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", o.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))

	// Send request
	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(responseBody, &errorResponse); err == nil {
			return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, errorResponse.Error.Message)
		}

		return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var openAIResponse ai.ChatCompletionResponse
	if err := json.Unmarshal(responseBody, &openAIResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &openAIResponse, nil
}

// SendMessage implements the legacy interface by converting to ChatCompletion
func (o *OpenAIBackend) SendMessage(ctx context.Context, req ai.Request) (*ai.Response, error) {
	// Convert legacy request to ChatCompletion format
	chatReq := ai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	// Use default model if none specified
	if chatReq.Model == "" {
		chatReq.Model = o.model
	}

	// Call ChatCompletion
	chatResp, err := o.ChatCompletion(ctx, chatReq)
	if err != nil {
		return &ai.Response{
			Error: err,
		}, err
	}

	// Convert to legacy response format
	if len(chatResp.Choices) == 0 {
		return &ai.Response{
			Error: fmt.Errorf("no response choices returned"),
		}, fmt.Errorf("no response choices returned")
	}

	return &ai.Response{
		Content:    chatResp.Choices[0].Message.Content,
		TokensUsed: chatResp.Usage.TotalTokens,
		Model:      chatResp.Model,
		Timestamp:  time.Unix(chatResp.Created, 0),
		Error:      nil,
	}, nil
}

// IsAvailable checks if the OpenAI API is reachable
func (o *OpenAIBackend) IsAvailable(ctx context.Context) bool {
	// Simple health check: try to list models
	url := fmt.Sprintf("%s/models", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Configure updates the backend configuration
func (o *OpenAIBackend) Configure(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		o.apiKey = apiKey
	}

	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		o.baseURL = baseURL
	}

	if model, ok := config["model"].(string); ok && model != "" {
		o.model = model
	}

	if timeout, ok := config["timeout"].(time.Duration); ok && timeout > 0 {
		o.httpClient.Timeout = timeout
	}

	// Validate that we have required configuration
	if o.apiKey == "" {
		return fmt.Errorf("api_key is required")
	}

	return nil
}

// GetModels retrieves available models from OpenAI
func (o *OpenAIBackend) GetModels(ctx context.Context) ([]Model, error) {
	url := fmt.Sprintf("%s/models", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var response struct {
		Data []Model `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data, nil
}

// Model represents an OpenAI model
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// GetDefaultModel returns the default model for this backend
func (o *OpenAIBackend) GetDefaultModel() string {
	return o.model
}

// SetDefaultModel sets the default model for this backend
func (o *OpenAIBackend) SetDefaultModel(model string) {
	o.model = model
}
