package mock

import (
	openai "github.com/jeanhaley32/go-openai-client"
)

// MockBackend is an alias for the library's MockBackend to maintain API compatibility
type MockBackend = openai.MockBackend

// NewMockBackend creates a new mock backend instance using the library
func NewMockBackend() *MockBackend {
	return openai.NewMockBackend()
}