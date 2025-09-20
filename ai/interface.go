package ai

import (
	"context"
	"time"
)

// Message represents a single message in a conversation following OpenAI Chat Completions format.
// Both fields are REQUIRED for proper operation.
type Message struct {
	// Role specifies who sent the message. REQUIRED.
	// Valid values:
	//   - "system": Instructions/context for the AI (usually first message)
	//   - "user": Human user input
	//   - "assistant": AI model responses
	Role string `json:"role"`

	// Content contains the actual message text. REQUIRED.
	// Cannot be empty string for most AI providers.
	Content string `json:"content"`
}

// ChatCompletionRequest represents a request following OpenAI Chat Completions API standard.
// Model and Messages are REQUIRED. All other fields are OPTIONAL with sensible defaults.
//
// Example usage:
//
//	req := ChatCompletionRequest{
//	  Model: "gpt-4",                    // REQUIRED
//	  Messages: []Message{               // REQUIRED - must have at least 1
//	    {Role: "user", Content: "Hello"},
//	  },
//	  MaxTokens: &[]int{150}[0],        // OPTIONAL - nil means no limit
//	  Temperature: &[]float64{0.7}[0],  // OPTIONAL - nil uses provider default
//	}
type ChatCompletionRequest struct {
	// Model specifies which AI model to use. REQUIRED.
	// Examples: "gpt-4", "gpt-3.5-turbo", "claude-3-sonnet", "mock-model-v1"
	Model string `json:"model"`

	// Messages contains the conversation history. REQUIRED.
	// Must have at least one message. Order matters - messages are processed sequentially.
	// Typically starts with a "system" message, followed by alternating "user"/"assistant".
	Messages []Message `json:"messages"`

	// MaxTokens limits the response length. OPTIONAL.
	// If nil, no limit is applied (uses provider default).
	// If set, response will be truncated at this token count.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// Temperature controls response randomness. OPTIONAL.
	// Range: 0.0 (deterministic) to 2.0 (very random)
	// If nil, uses provider default (usually 0.7-1.0)
	// Lower values = more focused, higher values = more creative
	Temperature *float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling. OPTIONAL.
	// Range: 0.0 to 1.0. Alternative to temperature.
	// If nil, uses provider default (usually 1.0)
	// Lower values = more focused responses
	TopP *float64 `json:"top_p,omitempty"`

	// Stream enables real-time response streaming. OPTIONAL.
	// Default: false (returns complete response)
	// Note: Streaming support depends on backend implementation
	Stream bool `json:"stream,omitempty"`
}

// Usage represents token usage information from the AI provider.
// All fields are provided by the backend and are read-only.
type Usage struct {
	// PromptTokens is the number of tokens in the input messages
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the AI's response
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of prompt and completion tokens
	TotalTokens int `json:"total_tokens"`
}

// Choice represents a single completion choice from the AI.
// Most providers return exactly one choice, but the format supports multiple alternatives.
type Choice struct {
	// Index is the position of this choice in the choices array (usually 0)
	Index int `json:"index"`

	// Message contains the AI's response
	Message Message `json:"message"`

	// FinishReason indicates why the response ended. Common values:
	//   - "stop": Natural completion
	//   - "length": Hit max_tokens limit
	//   - "content_filter": Content was filtered
	//   - "function_call": AI wants to call a function (if supported)
	FinishReason string `json:"finish_reason"`
}

// ChatCompletionResponse represents a response following OpenAI Chat Completions API standard.
// All fields are populated by the backend implementation.
//
// Example response structure:
//
//	{
//	  "id": "chatcmpl-abc123",
//	  "object": "chat.completion",
//	  "created": 1677858242,
//	  "model": "gpt-3.5-turbo",
//	  "choices": [{
//	    "index": 0,
//	    "message": {"role": "assistant", "content": "Hello!"},
//	    "finish_reason": "stop"
//	  }],
//	  "usage": {"prompt_tokens": 13, "completion_tokens": 7, "total_tokens": 20}
//	}
type ChatCompletionResponse struct {
	// ID is a unique identifier for this completion
	ID string `json:"id"`

	// Object type identifier, always "chat.completion" for this endpoint
	Object string `json:"object"`

	// Created is the Unix timestamp when the completion was created
	Created int64 `json:"created"`

	// Model is the name of the model used to generate the response
	Model string `json:"model"`

	// Choices contains the AI's response(s). Usually contains exactly one choice.
	Choices []Choice `json:"choices"`

	// Usage provides token consumption information for billing/monitoring
	Usage Usage `json:"usage"`
}

// Legacy types for backward compatibility and internal use.
// These are provided for existing code that hasn't migrated to ChatCompletion* types.
// New code should use ChatCompletionRequest and ChatCompletionResponse instead.

// Request is an alias for ChatCompletionRequest for backward compatibility
type Request = ChatCompletionRequest

// Response represents a simplified response format used by legacy SendMessage method.
// Unlike ChatCompletionResponse, this flattens the response into simple fields.
type Response struct {
	// Content contains the AI's response text
	Content string `json:"content"`

	// TokensUsed is the total number of tokens consumed (prompt + completion)
	TokensUsed int `json:"tokens_used"`

	// Model is the name of the model that generated the response
	Model string `json:"model"`

	// Timestamp indicates when the response was generated
	Timestamp time.Time `json:"timestamp"`

	// Error contains any error that occurred during processing
	Error error `json:"error"`
}

// Backend defines the interface that all AI backends must implement.
// This interface supports both modern OpenAI Chat Completions format and legacy methods.
//
// Implementation requirements:
//   - All methods must be safe for concurrent use
//   - Context cancellation must be respected
//   - Errors should be wrapped with descriptive messages
//
// Example implementation pattern:
//
//	type MyBackend struct {
//	  apiKey string
//	  baseURL string
//	}
//
//	func (b *MyBackend) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
//	  // Validate required fields
//	  if req.Model == "" { return nil, errors.New("model is required") }
//	  if len(req.Messages) == 0 { return nil, errors.New("messages is required") }
//	  // ... implement API call
//	}
type Backend interface {
	// Name returns a human-readable identifier for this backend.
	// Used for logging and debugging. Should be unique within your application.
	// Examples: "OpenAI", "Claude", "LocalLLaMA", "MockAI"
	Name() string

	// ChatCompletion sends a chat completion request following OpenAI Chat Completions standard.
	// This is the PREFERRED method for new implementations.
	//
	// Requirements:
	//   - req.Model and req.Messages are required
	//   - Must respect context cancellation
	//   - Should handle optional parameters gracefully (nil pointers)
	//   - Must return proper Usage information for token tracking
	//
	// Returns:
	//   - ChatCompletionResponse with at least one Choice
	//   - Error if request fails or is invalid
	ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)

	// SendMessage sends a request using the legacy simplified format.
	// This method exists for backward compatibility. New code should use ChatCompletion.
	//
	// Implementation can typically convert Request to ChatCompletionRequest internally,
	// call ChatCompletion, then flatten the response to the legacy Response format.
	SendMessage(ctx context.Context, req Request) (*Response, error)

	// IsAvailable performs a health check to determine if the backend is ready to serve requests.
	// Should be fast (< 1 second) and not consume API quotas if possible.
	//
	// Returns:
	//   - true: Backend is healthy and ready
	//   - false: Backend is unavailable (network issues, API down, etc.)
	//
	// Used by load balancers and circuit breakers for failover decisions.
	IsAvailable(ctx context.Context) bool

	// Configure sets backend-specific configuration options.
	// Called once during backend initialization with provider-specific settings.
	//
	// Common configuration keys:
	//   - "api_key": Authentication token
	//   - "base_url": Custom API endpoint
	//   - "timeout": Request timeout duration
	//   - "max_retries": Number of retry attempts
	//
	// Returns error if configuration is invalid or cannot be applied.
	Configure(config map[string]interface{}) error
}
