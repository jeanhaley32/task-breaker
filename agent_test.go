package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jeanhaley/task-breaker/ai"
	"github.com/jeanhaley/task-breaker/backends/mock"
)

func TestNewAgent(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	if agent == nil {
		t.Fatal("NewAgent() returned nil")
	}

	if agent.name != "TestAgent" {
		t.Errorf("Expected name 'TestAgent', got '%s'", agent.name)
	}

	if agent.aiBackend == nil {
		t.Error("Agent should have an AI backend")
	}

	if agent.aiBackend.Name() != backend.Name() {
		t.Error("Agent should use the provided backend")
	}

	if agent.context != "" {
		t.Error("New agent should have empty context initially")
	}
}

func TestAgent_LoadContext(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	// Create a temporary test file
	testContent := "This is test context data for the agent."
	testFile := createTempFile(t, testContent)
	defer os.Remove(testFile)

	// Test successful context loading
	err := agent.LoadContext(testFile)
	if err != nil {
		t.Fatalf("LoadContext failed: %v", err)
	}

	if agent.context != testContent {
		t.Errorf("Expected context '%s', got '%s'", testContent, agent.context)
	}

	// Test loading non-existent file
	err = agent.LoadContext("non-existent-file.txt")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}

	// Test loading directory instead of file
	tempDir := t.TempDir()
	err = agent.LoadContext(tempDir)
	if err == nil {
		t.Error("Expected error when loading directory")
	}
}

func TestAgent_SendMessage(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{
			name:    "simple message",
			message: "Hello",
			wantErr: false,
		},
		{
			name:    "empty message",
			message: "",
			wantErr: false, // Mock backend handles this
		},
		{
			name:    "long message",
			message: strings.Repeat("This is a long message. ", 50),
			wantErr: false,
		},
		{
			name:    "unicode message",
			message: "Hello 世界! 🤖",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := agent.SendMessage(tt.message)

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

			// Validate response
			if response == nil {
				t.Fatal("Response should not be nil")
			}

			if response.Content == "" {
				t.Error("Response content should not be empty")
			}

			if response.Model == "" {
				t.Error("Response model should not be empty")
			}

			if response.TokensUsed <= 0 {
				t.Error("Token usage should be positive")
			}

			if response.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}

			if response.Error != nil {
				t.Errorf("Response should not contain error: %v", response.Error)
			}

			// Verify legacy format
			if !strings.Contains(response.Content, "legacy format") {
				t.Error("Expected legacy format identifier in response")
			}
		})
	}
}

func TestAgent_SendChatCompletion(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	tests := []struct {
		name     string
		messages []ai.Message
		wantErr  bool
	}{
		{
			name: "single user message",
			messages: []ai.Message{
				{Role: "user", Content: "Hello"},
			},
			wantErr: false,
		},
		{
			name: "conversation flow",
			messages: []ai.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
				{Role: "user", Content: "How are you?"},
			},
			wantErr: false,
		},
		{
			name:     "empty messages",
			messages: []ai.Message{},
			wantErr:  false, // Mock backend handles this
		},
		{
			name: "system message included",
			messages: []ai.Message{
				{Role: "system", Content: "Be helpful"},
				{Role: "user", Content: "Hello"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := agent.SendChatCompletion(tt.messages)

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

			if len(response.Choices) != 1 {
				t.Errorf("Expected 1 choice, got %d", len(response.Choices))
			}

			choice := response.Choices[0]
			if choice.Message.Role != "assistant" {
				t.Errorf("Expected assistant role, got '%s'", choice.Message.Role)
			}

			if choice.Message.Content == "" {
				t.Error("Response content should not be empty")
			}

			if choice.FinishReason != "stop" {
				t.Errorf("Expected finish reason 'stop', got '%s'", choice.FinishReason)
			}

			// Verify OpenAI format
			if !strings.Contains(choice.Message.Content, "OpenAI format") {
				t.Error("Expected OpenAI format identifier in response")
			}
		})
	}
}

func TestAgent_SendChatCompletion_WithContext(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	// Load context
	contextContent := "You are a helpful coding assistant."
	testFile := createTempFile(t, contextContent)
	defer os.Remove(testFile)

	err := agent.LoadContext(testFile)
	if err != nil {
		t.Fatalf("Failed to load context: %v", err)
	}

	messages := []ai.Message{
		{Role: "user", Content: "Help me with Go"},
	}

	response, err := agent.SendChatCompletion(messages)
	if err != nil {
		t.Fatalf("SendChatCompletion failed: %v", err)
	}

	// The mock backend should include the system message from context
	// We can't directly verify this, but we can ensure the response is generated
	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Choices) == 0 {
		t.Fatal("Response should have at least one choice")
	}
}

func TestAgent_PrintContext(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	// Load some context
	contextContent := "Test context for printing"
	testFile := createTempFile(t, contextContent)
	defer os.Remove(testFile)

	err := agent.LoadContext(testFile)
	if err != nil {
		t.Fatalf("Failed to load context: %v", err)
	}

	// PrintContext doesn't return anything, so we just ensure it doesn't panic
	// In a real test environment, you might capture stdout to verify output
	agent.PrintContext()
}

func TestAgent_ContextIsolation(t *testing.T) {
	backend := mock.NewMockBackend()

	// Create two agents
	agent1 := NewAgent("Agent1", backend)
	agent2 := NewAgent("Agent2", backend)

	// Load different contexts
	context1 := "Context for agent 1"
	context2 := "Context for agent 2"

	file1 := createTempFile(t, context1)
	file2 := createTempFile(t, context2)
	defer os.Remove(file1)
	defer os.Remove(file2)

	err := agent1.LoadContext(file1)
	if err != nil {
		t.Fatalf("Failed to load context for agent1: %v", err)
	}

	err = agent2.LoadContext(file2)
	if err != nil {
		t.Fatalf("Failed to load context for agent2: %v", err)
	}

	// Verify contexts are isolated
	if agent1.context == agent2.context {
		t.Error("Agent contexts should be isolated")
	}

	if agent1.context != context1 {
		t.Errorf("Agent1 context mismatch: expected '%s', got '%s'", context1, agent1.context)
	}

	if agent2.context != context2 {
		t.Errorf("Agent2 context mismatch: expected '%s', got '%s'", context2, agent2.context)
	}
}

func TestAgent_MessageTimeout(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	// The mock backend has a 100ms delay, and our agent uses a 30-second timeout
	// So this should succeed (testing that timeout is reasonable)
	start := time.Now()

	response, err := agent.SendMessage("Hello")
	if err != nil {
		t.Fatalf("Message should not timeout: %v", err)
	}

	duration := time.Since(start)
	if duration > 5*time.Second {
		t.Errorf("Message took too long: %v", duration)
	}

	if response == nil {
		t.Error("Response should not be nil")
	}
}

func TestAgent_ConcurrentMessages(t *testing.T) {
	backend := mock.NewMockBackend()
	agent := NewAgent("TestAgent", backend)

	// Test that the agent can handle concurrent requests
	const numRequests = 5
	responses := make(chan *ai.Response, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			response, err := agent.SendMessage("Concurrent test")
			if err != nil {
				errors <- err
			} else {
				responses <- response
			}
		}(i)
	}

	// Collect results
	var successCount int
	var errorCount int

	for i := 0; i < numRequests; i++ {
		select {
		case <-responses:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful responses, got %d (errors: %d)", numRequests, successCount, errorCount)
	}
}

// Helper function to create temporary test files
func createTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-context-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	err = tmpFile.Close()
	if err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

// Benchmark tests
func BenchmarkAgent_SendMessage(b *testing.B) {
	backend := mock.NewMockBackend()
	agent := NewAgent("BenchAgent", backend)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.SendMessage("Benchmark test message")
		if err != nil {
			b.Fatalf("SendMessage failed: %v", err)
		}
	}
}

func BenchmarkAgent_SendChatCompletion(b *testing.B) {
	backend := mock.NewMockBackend()
	agent := NewAgent("BenchAgent", backend)

	messages := []ai.Message{
		{Role: "user", Content: "Benchmark test message"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.SendChatCompletion(messages)
		if err != nil {
			b.Fatalf("SendChatCompletion failed: %v", err)
		}
	}
}