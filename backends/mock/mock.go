package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/jeanhaley/task-breaker/ai"
)

// MockBackend is a simple mock implementation of the AI backend interface
type MockBackend struct {
	name   string
	config map[string]interface{}
}

// NewMockBackend creates a new mock backend instance
func NewMockBackend() *MockBackend {
	return &MockBackend{
		name:   "MockAI",
		config: make(map[string]interface{}),
	}
}

// Name returns the name of this backend
func (m *MockBackend) Name() string {
	return m.name
}

// ChatCompletion implements OpenAI Chat Completions API standard
func (m *MockBackend) ChatCompletion(ctx context.Context, req ai.ChatCompletionRequest) (*ai.ChatCompletionResponse, error) {
	// Simulate some processing time
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create a mock response based on the last message
	var responseContent string
	if len(req.Messages) > 0 {
		lastMessage := req.Messages[len(req.Messages)-1]
		responseContent = fmt.Sprintf("Mock AI (OpenAI format) received: '%s'. This is a simulated response using Chat Completions API!", lastMessage.Content)
	} else {
		responseContent = "Mock AI: Hello! I'm responding via the OpenAI Chat Completions format."
	}

	// Calculate token usage
	promptTokens := 0
	for _, msg := range req.Messages {
		promptTokens += len(msg.Content) / 4 // Rough estimate
	}
	completionTokens := len(responseContent) / 4
	totalTokens := promptTokens + completionTokens

	return &ai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-mock-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ai.Choice{
			{
				Index: 0,
				Message: ai.Message{
					Role:    "assistant",
					Content: responseContent,
				},
				FinishReason: "stop",
			},
		},
		Usage: ai.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
	}, nil
}

// SendMessage simulates sending a message to an AI and returns a mock response (legacy method)
func (m *MockBackend) SendMessage(ctx context.Context, req ai.Request) (*ai.Response, error) {
	// Simulate some processing time
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create a simple mock response based on the last message
	var responseContent string
	if len(req.Messages) > 0 {
		lastMessage := req.Messages[len(req.Messages)-1]
		responseContent = fmt.Sprintf("Mock AI (legacy format) received: '%s'. This is a simulated response!", lastMessage.Content)
	} else {
		responseContent = "Mock AI: Hello! I'm a mock backend for testing."
	}

	return &ai.Response{
		Content:    responseContent,
		TokensUsed: len(responseContent) / 4, // Rough token estimate
		Model:      "mock-model-v1",
		Timestamp:  time.Now(),
		Error:      nil,
	}, nil
}

// IsAvailable always returns true for the mock backend
func (m *MockBackend) IsAvailable(ctx context.Context) bool {
	return true
}

// Configure sets configuration for the mock backend
func (m *MockBackend) Configure(config map[string]interface{}) error {
	for key, value := range config {
		m.config[key] = value
	}

	// Check if name is being configured
	if name, ok := config["name"].(string); ok {
		m.name = name
	}

	return nil
}
