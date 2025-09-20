package ai

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessage_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "user message",
			message: Message{
				Role:    "user",
				Content: "Hello, world!",
			},
			expected: `{"role":"user","content":"Hello, world!"}`,
		},
		{
			name: "system message",
			message: Message{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			expected: `{"role":"system","content":"You are a helpful assistant."}`,
		},
		{
			name: "assistant message",
			message: Message{
				Role:    "assistant",
				Content: "How can I help you today?",
			},
			expected: `{"role":"assistant","content":"How can I help you today?"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.message)
			if err != nil {
				t.Fatalf("Failed to marshal message: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}

			// Test deserialization
			var decoded Message
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal message: %v", err)
			}

			if decoded.Role != tt.message.Role {
				t.Errorf("Expected role %s, got %s", tt.message.Role, decoded.Role)
			}
			if decoded.Content != tt.message.Content {
				t.Errorf("Expected content %s, got %s", tt.message.Content, decoded.Content)
			}
		})
	}
}

func TestChatCompletionRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request ChatCompletionRequest
		valid   bool
	}{
		{
			name: "valid minimal request",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			valid: true,
		},
		{
			name: "missing model",
			request: ChatCompletionRequest{
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			valid: false,
		},
		{
			name: "empty messages",
			request: ChatCompletionRequest{
				Model:    "gpt-4",
				Messages: []Message{},
			},
			valid: false,
		},
		{
			name: "valid with optional parameters",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
				MaxTokens:   &[]int{150}[0],
				Temperature: &[]float64{0.7}[0],
				TopP:        &[]float64{0.9}[0],
				Stream:      false,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateChatCompletionRequest(tt.request)
			if valid != tt.valid {
				t.Errorf("Expected validation result %v, got %v", tt.valid, valid)
			}
		})
	}
}

func TestChatCompletionResponse_Structure(t *testing.T) {
	response := ChatCompletionResponse{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello! How can I help you?",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}

	// Test JSON serialization
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Test JSON deserialization
	var decoded ChatCompletionResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Validate structure
	if decoded.ID != response.ID {
		t.Errorf("Expected ID %s, got %s", response.ID, decoded.ID)
	}
	if decoded.Object != response.Object {
		t.Errorf("Expected object %s, got %s", response.Object, decoded.Object)
	}
	if len(decoded.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(decoded.Choices))
	}
	if decoded.Choices[0].Message.Content != "Hello! How can I help you?" {
		t.Errorf("Unexpected message content: %s", decoded.Choices[0].Message.Content)
	}
	if decoded.Usage.TotalTokens != 18 {
		t.Errorf("Expected total tokens 18, got %d", decoded.Usage.TotalTokens)
	}
}

func TestUsage_TokenCalculations(t *testing.T) {
	usage := Usage{
		PromptTokens:     25,
		CompletionTokens: 15,
		TotalTokens:      40,
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		t.Errorf("Total tokens should equal prompt + completion: %d != %d + %d",
			usage.TotalTokens, usage.PromptTokens, usage.CompletionTokens)
	}
}

func TestChoice_FinishReasons(t *testing.T) {
	validFinishReasons := []string{"stop", "length", "content_filter", "function_call"}

	for _, reason := range validFinishReasons {
		choice := Choice{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: "Test response",
			},
			FinishReason: reason,
		}

		if choice.FinishReason != reason {
			t.Errorf("Expected finish reason %s, got %s", reason, choice.FinishReason)
		}
	}
}

func TestLegacyResponse_BackwardCompatibility(t *testing.T) {
	response := Response{
		Content:    "Test response",
		TokensUsed: 25,
		Model:      "test-model",
		Timestamp:  time.Now(),
		Error:      nil,
	}

	// Test serialization
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal legacy response: %v", err)
	}

	// Test deserialization
	var decoded Response
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal legacy response: %v", err)
	}

	if decoded.Content != response.Content {
		t.Errorf("Expected content %s, got %s", response.Content, decoded.Content)
	}
	if decoded.TokensUsed != response.TokensUsed {
		t.Errorf("Expected tokens %d, got %d", response.TokensUsed, decoded.TokensUsed)
	}
	if decoded.Model != response.Model {
		t.Errorf("Expected model %s, got %s", response.Model, decoded.Model)
	}
}

// Helper function for request validation (would be in production code)
func validateChatCompletionRequest(req ChatCompletionRequest) bool {
	if req.Model == "" {
		return false
	}
	if len(req.Messages) == 0 {
		return false
	}
	for _, msg := range req.Messages {
		if msg.Role == "" || msg.Content == "" {
			return false
		}
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			return false
		}
	}
	return true
}

func TestValidateChatCompletionRequest_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		request ChatCompletionRequest
		valid   bool
	}{
		{
			name: "message with empty content",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "user", Content: ""},
				},
			},
			valid: false,
		},
		{
			name: "message with invalid role",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "invalid", Content: "Hello"},
				},
			},
			valid: false,
		},
		{
			name: "message with empty role",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "", Content: "Hello"},
				},
			},
			valid: false,
		},
		{
			name: "conversation flow",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there!"},
					{Role: "user", Content: "How are you?"},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateChatCompletionRequest(tt.request)
			if valid != tt.valid {
				t.Errorf("Expected validation result %v, got %v", tt.valid, valid)
			}
		})
	}
}
