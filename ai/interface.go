package ai

import (
	"context"
	"time"
)

// Message represents a single message in a conversation following OpenAI Chat Completions format
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"` // The message content
}

// ChatCompletionRequest represents a request following OpenAI Chat Completions API standard
type ChatCompletionRequest struct {
	Model       string    `json:"model"`                 // Model identifier
	Messages    []Message `json:"messages"`              // Conversation history
	MaxTokens   *int      `json:"max_tokens,omitempty"`  // Maximum tokens in response
	Temperature *float64  `json:"temperature,omitempty"` // Response randomness (0.0-2.0)
	TopP        *float64  `json:"top_p,omitempty"`       // Nucleus sampling parameter
	Stream      bool      `json:"stream,omitempty"`      // Whether to stream responses
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Choice represents a single completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // "stop", "length", "content_filter", etc.
}

// ChatCompletionResponse represents a response following OpenAI Chat Completions API standard
type ChatCompletionResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`  // "chat.completion"
	Created int64     `json:"created"` // Unix timestamp
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
}

// Legacy types for backward compatibility and internal use
type Request = ChatCompletionRequest
type Response struct {
	Content     string    `json:"content"`      // The AI's response
	TokensUsed  int       `json:"tokens_used"`  // Number of tokens consumed
	Model       string    `json:"model"`        // Model that generated the response
	Timestamp   time.Time `json:"timestamp"`    // When the response was generated
	Error       error     `json:"error"`        // Any error that occurred
}

// Backend defines the interface that all AI backends must implement
type Backend interface {
	// Name returns the name/identifier of this backend
	Name() string

	// ChatCompletion sends a chat completion request following OpenAI standard
	ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)

	// SendMessage sends a request to the AI backend and returns the response (legacy method)
	SendMessage(ctx context.Context, req Request) (*Response, error)

	// IsAvailable checks if the backend is currently available
	IsAvailable(ctx context.Context) bool

	// Configure allows setting backend-specific configuration
	Configure(config map[string]interface{}) error
}