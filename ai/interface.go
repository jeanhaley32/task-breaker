package ai

import (
	"context"
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	Role      string    `json:"role"`      // "user", "assistant", "system"
	Content   string    `json:"content"`   // The message content
	Timestamp time.Time `json:"timestamp"` // When the message was created
}

// Request represents a request to an AI backend
type Request struct {
	Messages    []Message `json:"messages"`    // Conversation history
	MaxTokens   int       `json:"max_tokens"`  // Maximum tokens in response
	Temperature float64   `json:"temperature"` // Response randomness (0.0-1.0)
	Context     string    `json:"context"`     // Additional context for the AI
}

// Response represents a response from an AI backend
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

	// SendMessage sends a request to the AI backend and returns the response
	SendMessage(ctx context.Context, req Request) (*Response, error)

	// IsAvailable checks if the backend is currently available
	IsAvailable(ctx context.Context) bool

	// Configure allows setting backend-specific configuration
	Configure(config map[string]interface{}) error
}