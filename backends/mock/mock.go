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

// SendMessage simulates sending a message to an AI and returns a mock response
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
		responseContent = fmt.Sprintf("Mock AI received: '%s'. This is a simulated response!", lastMessage.Content)
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