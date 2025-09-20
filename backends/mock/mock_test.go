package mock

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeanhaley/task-breaker/ai"
)

func TestNewMockBackend(t *testing.T) {
	backend := NewMockBackend()

	if backend == nil {
		t.Fatal("NewMockBackend() returned nil")
	}

	if backend.Name() != "MockAI" {
		t.Errorf("Expected name 'MockAI', got '%s'", backend.Name())
	}

	if backend.config == nil {
		t.Error("Config map should be initialized")
	}
}

func TestMockBackend_Name(t *testing.T) {
	backend := NewMockBackend()

	name := backend.Name()
	if name == "" {
		t.Error("Name() should not return empty string")
	}

	// Test custom name configuration
	err := backend.Configure(map[string]interface{}{
		"name": "CustomMock",
	})
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	if backend.Name() != "CustomMock" {
		t.Errorf("Expected configured name 'CustomMock', got '%s'", backend.Name())
	}
}

func TestMockBackend_IsAvailable(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()

	available := backend.IsAvailable(ctx)
	if !available {
		t.Error("Mock backend should always be available")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	available = backend.IsAvailable(cancelledCtx)
	if !available {
		t.Error("Mock backend availability should not depend on context state")
	}
}

func TestMockBackend_Configure(t *testing.T) {
	backend := NewMockBackend()

	tests := []struct {
		name   string
		config map[string]interface{}
		valid  bool
	}{
		{
			name:   "empty config",
			config: map[string]interface{}{},
			valid:  true,
		},
		{
			name: "name configuration",
			config: map[string]interface{}{
				"name": "TestMock",
			},
			valid: true,
		},
		{
			name: "multiple settings",
			config: map[string]interface{}{
				"name":    "TestMock",
				"setting": "value",
				"number":  42,
			},
			valid: true,
		},
		{
			name:   "nil config",
			config: nil,
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.Configure(tt.config)
			if tt.valid && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestMockBackend_ChatCompletion(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()

	tests := []struct {
		name    string
		request ai.ChatCompletionRequest
		wantErr bool
	}{
		{
			name: "simple user message",
			request: ai.ChatCompletionRequest{
				Model: "mock-model",
				Messages: []ai.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "conversation with system message",
			request: ai.ChatCompletionRequest{
				Model: "mock-model",
				Messages: []ai.Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there!"},
					{Role: "user", Content: "How are you?"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty messages",
			request: ai.ChatCompletionRequest{
				Model:    "mock-model",
				Messages: []ai.Message{},
			},
			wantErr: false, // Mock backend handles this gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := backend.ChatCompletion(ctx, tt.request)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}

			if err != nil {
				return // Expected error case
			}

			// Validate response structure
			if response == nil {
				t.Fatal("Response should not be nil")
			}

			if response.ID == "" {
				t.Error("Response ID should not be empty")
			}

			if response.Object != "chat.completion" {
				t.Errorf("Expected object 'chat.completion', got '%s'", response.Object)
			}

			if response.Model != tt.request.Model {
				t.Errorf("Expected model '%s', got '%s'", tt.request.Model, response.Model)
			}

			if len(response.Choices) != 1 {
				t.Errorf("Expected 1 choice, got %d", len(response.Choices))
			}

			choice := response.Choices[0]
			if choice.Index != 0 {
				t.Errorf("Expected choice index 0, got %d", choice.Index)
			}

			if choice.Message.Role != "assistant" {
				t.Errorf("Expected assistant role, got '%s'", choice.Message.Role)
			}

			if choice.Message.Content == "" {
				t.Error("Response content should not be empty")
			}

			if choice.FinishReason != "stop" {
				t.Errorf("Expected finish reason 'stop', got '%s'", choice.FinishReason)
			}

			// Validate usage information
			if response.Usage.TotalTokens != response.Usage.PromptTokens+response.Usage.CompletionTokens {
				t.Error("Total tokens should equal prompt + completion tokens")
			}

			if response.Usage.PromptTokens < 0 || response.Usage.CompletionTokens < 0 {
				t.Error("Token counts should not be negative")
			}
		})
	}
}

func TestMockBackend_ChatCompletion_ContextHandling(t *testing.T) {
	backend := NewMockBackend()

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	request := ai.ChatCompletionRequest{
		Model: "mock-model",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	// Mock backend simulates 100ms delay, so this should timeout
	_, err := backend.ChatCompletion(ctx, request)
	if err == nil {
		t.Error("Expected timeout error")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got: %v", err)
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = backend.ChatCompletion(cancelledCtx, request)
	if err == nil {
		t.Error("Expected cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("Expected context canceled, got: %v", err)
	}
}

func TestMockBackend_ChatCompletion_ResponseContent(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()

	// Test response content varies with input
	tests := []struct {
		name          string
		userMessage   string
		expectContains string
	}{
		{
			name:          "hello message",
			userMessage:   "Hello",
			expectContains: "Hello",
		},
		{
			name:          "question message",
			userMessage:   "What is AI?",
			expectContains: "What is AI?",
		},
		{
			name:          "complex message",
			userMessage:   "Can you help me with Go programming?",
			expectContains: "Can you help me with Go programming?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := ai.ChatCompletionRequest{
				Model: "mock-model",
				Messages: []ai.Message{
					{Role: "user", Content: tt.userMessage},
				},
			}

			response, err := backend.ChatCompletion(ctx, request)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			content := response.Choices[0].Message.Content
			if !strings.Contains(content, tt.expectContains) {
				t.Errorf("Expected response to contain '%s', got: %s", tt.expectContains, content)
			}

			// Verify it mentions OpenAI format
			if !strings.Contains(content, "OpenAI format") {
				t.Error("Expected response to mention OpenAI format")
			}
		})
	}
}

func TestMockBackend_SendMessage_Legacy(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()

	request := ai.Request{
		Model: "mock-model",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   &[]int{150}[0],
		Temperature: &[]float64{0.7}[0],
	}

	response, err := backend.SendMessage(ctx, request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Content == "" {
		t.Error("Response content should not be empty")
	}

	if response.Model != "mock-model-v1" {
		t.Errorf("Expected model 'mock-model-v1', got '%s'", response.Model)
	}

	if response.TokensUsed <= 0 {
		t.Error("Token usage should be positive")
	}

	if response.Error != nil {
		t.Errorf("Expected no error in response, got: %v", response.Error)
	}

	if response.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	// Verify legacy format identifier
	if !strings.Contains(response.Content, "legacy format") {
		t.Error("Expected response to mention legacy format")
	}
}

func TestMockBackend_SendMessage_ContextHandling(t *testing.T) {
	backend := NewMockBackend()

	// Test timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	request := ai.Request{
		Model: "mock-model",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := backend.SendMessage(ctx, request)
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Test cancellation handling
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = backend.SendMessage(cancelledCtx, request)
	if err == nil {
		t.Error("Expected cancellation error")
	}
}

func TestMockBackend_TokenEstimation(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()

	// Test that longer messages result in higher token counts
	shortRequest := ai.ChatCompletionRequest{
		Model: "mock-model",
		Messages: []ai.Message{
			{Role: "user", Content: "Hi"},
		},
	}

	longRequest := ai.ChatCompletionRequest{
		Model: "mock-model",
		Messages: []ai.Message{
			{Role: "user", Content: "This is a much longer message that should result in a higher token count when processed by the mock backend implementation."},
		},
	}

	shortResponse, err := backend.ChatCompletion(ctx, shortRequest)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	longResponse, err := backend.ChatCompletion(ctx, longRequest)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if longResponse.Usage.PromptTokens <= shortResponse.Usage.PromptTokens {
		t.Error("Longer message should have more prompt tokens")
	}

	if longResponse.Usage.TotalTokens <= shortResponse.Usage.TotalTokens {
		t.Error("Longer message should have more total tokens")
	}
}