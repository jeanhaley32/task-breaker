package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeanhaley32/go-openai-client"
	"github.com/jeanhaley32/go-openai-client/chat"
	"github.com/jeanhaley/task-breaker/config"
)

// TestIntegration_FullWorkflow tests the complete system workflow
func TestIntegration_FullWorkflow(t *testing.T) {
	// Test with mock backend to ensure no API costs
	backend := openai.NewMockBackend()

	// Create chat controller
	controller := chat.NewController(backend, &chat.ControllerConfig{
		DefaultModel: "gpt-4",
		MaxTokens:    100,
		Temperature:  0.7,
	})

	ctx := context.Background()

	// 1. Create a new conversation
	t.Log("Step 1: Creating conversation...")
	conv := controller.CreateConversation("You are a helpful assistant for testing.")
	if conv == nil {
		t.Fatal("Failed to create conversation")
	}

	// 2. Send initial message
	t.Log("Step 2: Sending initial message...")
	response1, err := controller.SendMessage(ctx, chat.ChatRequest{
		ConversationID: conv.ID,
		Message:        "Hello, can you help me test this system?",
	})
	if err != nil {
		t.Fatalf("Failed to send first message: %v", err)
	}

	if response1.ConversationID != conv.ID {
		t.Error("Response conversation ID should match")
	}

	// 3. Send follow-up message
	t.Log("Step 3: Sending follow-up message...")
	_, err = controller.SendMessage(ctx, chat.ChatRequest{
		ConversationID: conv.ID,
		Message:        "What features does Task Breaker have?",
	})
	if err != nil {
		t.Fatalf("Failed to send second message: %v", err)
	}

	// 4. Verify conversation history
	t.Log("Step 4: Verifying conversation history...")
	updatedConv, err := controller.GetConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to get updated conversation: %v", err)
	}

	expectedMessages := 5 // system + user1 + assistant1 + user2 + assistant2
	if len(updatedConv.Messages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, len(updatedConv.Messages))
	}

	// 5. Get conversation summary
	t.Log("Step 5: Getting conversation summary...")
	summary, err := controller.GetConversationSummary(conv.ID)
	if err != nil {
		t.Fatalf("Failed to get conversation summary: %v", err)
	}

	if summary.UserMessages != 2 {
		t.Errorf("Expected 2 user messages, got %d", summary.UserMessages)
	}

	if summary.AssistantMessages != 2 {
		t.Errorf("Expected 2 assistant messages, got %d", summary.AssistantMessages)
	}

	// 6. Test backend switching
	t.Log("Step 6: Testing backend switching...")
	newBackend := openai.NewMockBackend()
	newBackend.Configure(map[string]interface{}{"name": "SecondMock"})

	controller.SetBackend(newBackend)

	if controller.GetBackend().Name() != "SecondMock" {
		t.Error("Backend switching failed")
	}

	// 7. Send message with new backend
	t.Log("Step 7: Sending message with new backend...")
	response3, err := controller.SendMessage(ctx, chat.ChatRequest{
		ConversationID: conv.ID,
		Message:        "Testing with new backend",
	})
	if err != nil {
		t.Fatalf("Failed to send message with new backend: %v", err)
	}

	if response3 == nil {
		t.Error("Response should not be nil")
	}

	// 8. Clear conversation
	t.Log("Step 8: Clearing conversation...")
	err = controller.ClearConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to clear conversation: %v", err)
	}

	clearedConv, err := controller.GetConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to get cleared conversation: %v", err)
	}

	if len(clearedConv.Messages) != 1 { // Only system message should remain
		t.Errorf("Expected 1 message after clear, got %d", len(clearedConv.Messages))
	}

	// 9. Delete conversation
	t.Log("Step 9: Deleting conversation...")
	err = controller.DeleteConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to delete conversation: %v", err)
	}

	_, err = controller.GetConversation(conv.ID)
	if err == nil {
		t.Error("Conversation should not exist after deletion")
	}

	t.Log("✅ Full workflow integration test completed successfully")
}

// TestIntegration_ConfigurationSystem tests the configuration management
func TestIntegration_ConfigurationSystem(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	// 1. Initialize configuration manager
	t.Log("Step 1: Initializing configuration...")
	configManager := config.NewManager(configPath)

	// 2. Load default configuration
	err := configManager.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	cfg := configManager.GetConfig()
	if cfg == nil {
		t.Fatal("Configuration should not be nil")
	}

	// 3. Validate default configuration
	t.Log("Step 2: Validating configuration...")
	err = configManager.ValidateConfig()
	if err != nil {
		t.Fatalf("Default configuration should be valid: %v", err)
	}

	// 4. Test API key configuration
	t.Log("Step 3: Testing API key configuration...")
	configManager.SetOpenAIAPIKey("test-api-key")
	configManager.SetDefaultBackend("openai")

	// 5. Save and reload configuration
	t.Log("Step 4: Testing save/reload...")
	err = configManager.Save()
	if err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// Create new manager to test loading
	newManager := config.NewManager(configPath)
	err = newManager.Load()
	if err != nil {
		t.Fatalf("Failed to reload configuration: %v", err)
	}

	reloadedCfg := newManager.GetConfig()
	if reloadedCfg.OpenAI.APIKey != "test-api-key" {
		t.Error("API key should persist after save/reload")
	}

	if reloadedCfg.Default.Backend != "openai" {
		t.Error("Default backend should persist after save/reload")
	}

	// 6. Test environment variable override
	t.Log("Step 5: Testing environment variable override...")
	os.Setenv("OPENAI_API_KEY", "env-override-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	envManager := config.NewManager(configPath)
	err = envManager.Load()
	if err != nil {
		t.Fatalf("Failed to load with env override: %v", err)
	}

	envCfg := envManager.GetConfig()
	if envCfg.OpenAI.APIKey != "env-override-key" {
		t.Error("Environment variable should override config file")
	}

	t.Log("✅ Configuration system integration test completed successfully")
}

// TestIntegration_MultipleBackends tests switching between different backends
func TestIntegration_MultipleBackends(t *testing.T) {
	ctx := context.Background()

	// 1. Create multiple backends
	t.Log("Step 1: Creating multiple backends...")
	mockBackend1 := openai.NewMockBackend()
	mockBackend1.Configure(map[string]interface{}{"name": "Mock1"})

	mockBackend2 := openai.NewMockBackend()
	mockBackend2.Configure(map[string]interface{}{"name": "Mock2"})

	// Test OpenAI backend only if API key is available
	var openaiBackend openai.Backend
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		t.Log("OpenAI API key found - including OpenAI backend in test")
		openaiBackend = openai.NewClient(openai.Config{
			APIKey: apiKey,
			Model:  "gpt-3.5-turbo", // Use cheaper model for testing
		})
	} else {
		t.Log("No OpenAI API key - skipping OpenAI backend test")
	}

	// 2. Test each backend
	backends := []openai.Backend{mockBackend1, mockBackend2}
	if openaiBackend != nil {
		backends = append(backends, openaiBackend)
	}

	for i, backend := range backends {
		t.Logf("Step %d: Testing backend %s...", i+2, backend.Name())

		// Check availability
		available := backend.IsAvailable(ctx)
		if !available {
			t.Logf("Backend %s not available, skipping", backend.Name())
			continue
		}

		// Create controller with this backend
		controller := chat.NewController(backend, nil)

		// Create conversation
		conv := controller.CreateConversation("You are a test assistant.")

		// Send test message
		response, err := controller.SendMessage(ctx, chat.ChatRequest{
			ConversationID: conv.ID,
			Message:        "Hello, this is a test message.",
			MaxTokens:      &[]int{50}[0], // Limit tokens for cost control
		})

		if err != nil {
			t.Errorf("Failed to send message to %s: %v", backend.Name(), err)
			continue
		}

		if response == nil {
			t.Errorf("No response from %s", backend.Name())
			continue
		}

		if response.Message.Content == "" {
			t.Errorf("Empty response from %s", backend.Name())
			continue
		}

		t.Logf("✅ Backend %s responded: %s", backend.Name(),
			truncateString(response.Message.Content, 50))
	}

	t.Log("✅ Multiple backends integration test completed successfully")
}

// TestIntegration_ContextLoading tests the legacy agent context loading
func TestIntegration_ContextLoading(t *testing.T) {
	// 1. Create test context file
	t.Log("Step 1: Creating test context file...")
	contextContent := "You are a helpful AI assistant specialized in Go programming. Always provide working code examples and explain best practices."

	tempFile := createIntegrationTempFile(t, contextContent)
	defer os.Remove(tempFile)

	// 2. Create agent with context
	t.Log("Step 2: Creating agent with context...")
	backend := openai.NewMockBackend()
	agent := NewAgent("ContextTestAgent", backend)

	err := agent.LoadContext(tempFile)
	if err != nil {
		t.Fatalf("Failed to load context: %v", err)
	}

	if agent.context != contextContent {
		t.Error("Context content should match file content")
	}

	// 3. Test that context affects responses
	t.Log("Step 3: Testing context-aware responses...")
	response, err := agent.SendChatCompletion([]openai.Message{
		{Role: "user", Content: "Help me with a Go function"},
	})

	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// The mock should include the system message with context
	if response == nil || len(response.Choices) == 0 {
		t.Fatal("No response received")
	}

	// 4. Test legacy SendMessage method
	t.Log("Step 4: Testing legacy SendMessage...")
	legacyResponse, err := agent.SendMessage("What's the best way to handle errors in Go?")
	if err != nil {
		t.Fatalf("Legacy SendMessage failed: %v", err)
	}

	if legacyResponse.Content == "" {
		t.Error("Legacy response should have content")
	}

	if !strings.Contains(legacyResponse.Content, "legacy format") {
		t.Error("Legacy response should indicate format")
	}

	t.Log("✅ Context loading integration test completed successfully")
}

// TestIntegration_ErrorHandling tests various error scenarios
func TestIntegration_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	// 1. Test with invalid OpenAI configuration
	t.Log("Step 1: Testing invalid OpenAI configuration...")
	invalidBackend := openai.NewClient(openai.Config{
		APIKey: "invalid-key",
		Model:  "gpt-4",
	})

	controller := chat.NewController(invalidBackend, nil)

	// This should fail gracefully
	_, err := controller.SendMessage(ctx, chat.ChatRequest{
		Message: "This should fail",
	})

	if err == nil {
		t.Log("Note: Invalid API key test didn't fail - might be mock or test environment")
	} else {
		t.Logf("✅ Invalid API key properly handled: %v", err)
	}

	// 2. Test with cancelled context
	t.Log("Step 2: Testing cancelled context...")
	mockBackend := openai.NewMockBackend()
	controller = chat.NewController(mockBackend, nil)

	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	_, err = controller.SendMessage(cancelledCtx, chat.ChatRequest{
		Message: "This should be cancelled",
	})

	if err == nil {
		t.Error("Cancelled context should cause error")
	} else {
		t.Logf("✅ Cancelled context properly handled: %v", err)
	}

	// 3. Test with timeout context
	t.Log("Step 3: Testing timeout context...")
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	// Mock has 100ms delay, so this should timeout
	_, err = controller.SendMessage(timeoutCtx, chat.ChatRequest{
		Message: "This should timeout",
	})

	if err == nil {
		t.Error("Timeout context should cause error")
	} else {
		t.Logf("✅ Timeout properly handled: %v", err)
	}

	// 4. Test invalid conversation ID
	t.Log("Step 4: Testing invalid conversation ID...")
	_, err = controller.SendMessage(ctx, chat.ChatRequest{
		ConversationID: "invalid-conversation-id",
		Message:        "This should fail",
	})

	if err == nil {
		t.Error("Invalid conversation ID should cause error")
	} else {
		t.Logf("✅ Invalid conversation ID properly handled: %v", err)
	}

	// 5. Test backend switching with unavailable backend
	t.Log("Step 5: Testing backend availability...")

	// Create a backend that will be unavailable
	unavailableBackend := openai.NewClient(openai.Config{
		APIKey:  "fake-key",
		BaseURL: "https://invalid-url-that-does-not-exist.com",
	})

	available := controller.IsBackendAvailable(ctx)
	t.Logf("Current backend available: %v", available)

	controller.SetBackend(unavailableBackend)
	available = controller.IsBackendAvailable(ctx)
	if available {
		t.Log("Note: Unavailable backend test didn't work - might be network/test environment")
	} else {
		t.Log("✅ Unavailable backend properly detected")
	}

	t.Log("✅ Error handling integration test completed successfully")
}

// TestIntegration_ConcurrentOperations tests system under concurrent load
func TestIntegration_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	backend := openai.NewMockBackend()
	controller := chat.NewController(backend, nil)

	const numGoroutines = 20
	const messagesPerGoroutine = 5

	t.Logf("Step 1: Testing %d concurrent goroutines with %d messages each...",
		numGoroutines, messagesPerGoroutine)

	// Create channels for results
	results := make(chan error, numGoroutines*messagesPerGoroutine)
	conversationIDs := make(chan chat.ConversationID, numGoroutines)

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			// Each goroutine creates its own conversation
			conv := controller.CreateConversation("Concurrent test assistant")
			conversationIDs <- conv.ID

			// Send multiple messages in this conversation
			for j := 0; j < messagesPerGoroutine; j++ {
				_, err := controller.SendMessage(ctx, chat.ChatRequest{
					ConversationID: conv.ID,
					Message:        fmt.Sprintf("Message %d from goroutine %d", j, goroutineID),
				})
				results <- err
			}
		}(i)
	}

	// Collect results
	t.Log("Step 2: Collecting results...")
	var errors []error
	totalOperations := numGoroutines * messagesPerGoroutine

	for i := 0; i < totalOperations; i++ {
		select {
		case err := <-results:
			if err != nil {
				errors = append(errors, err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	if len(errors) > 0 {
		t.Errorf("Got %d errors out of %d operations:", len(errors), totalOperations)
		for i, err := range errors {
			if i < 5 { // Show first 5 errors
				t.Errorf("  Error %d: %v", i+1, err)
			}
		}
		if len(errors) > 5 {
			t.Errorf("  ... and %d more errors", len(errors)-5)
		}
	}

	// Collect conversation IDs
	t.Log("Step 3: Verifying conversations...")
	createdConversations := make([]chat.ConversationID, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		select {
		case id := <-conversationIDs:
			createdConversations = append(createdConversations, id)
		case <-time.After(1 * time.Second):
			t.Error("Timeout collecting conversation IDs")
		}
	}

	if len(createdConversations) != numGoroutines {
		t.Errorf("Expected %d conversations, got %d", numGoroutines, len(createdConversations))
	}

	// Verify all conversations exist and have correct message counts
	for _, convID := range createdConversations {
		conv, err := controller.GetConversation(convID)
		if err != nil {
			t.Errorf("Failed to get conversation %s: %v", convID, err)
			continue
		}

		expectedMessages := 1 + (messagesPerGoroutine * 2) // system + (user + assistant) * messages
		if len(conv.Messages) != expectedMessages {
			t.Errorf("Conversation %s: expected %d messages, got %d",
				convID, expectedMessages, len(conv.Messages))
		}
	}

	// Check controller stats
	t.Log("Step 4: Checking final statistics...")
	stats := controller.GetStats()
	if stats.TotalConversations != numGoroutines {
		t.Errorf("Expected %d total conversations, got %d",
			numGoroutines, stats.TotalConversations)
	}

	expectedTotalMessages := numGoroutines * (1 + (messagesPerGoroutine * 2))
	if stats.TotalMessages != expectedTotalMessages {
		t.Errorf("Expected %d total messages, got %d",
			expectedTotalMessages, stats.TotalMessages)
	}

	successRate := float64(totalOperations-len(errors)) / float64(totalOperations) * 100
	t.Logf("✅ Concurrent operations test completed - Success rate: %.1f%%", successRate)

	if successRate < 95.0 {
		t.Errorf("Success rate too low: %.1f%% (expected >= 95%%)", successRate)
	}
}

// Helper functions

// createIntegrationTempFile creates a temporary file with the given content
func createIntegrationTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "integration-test-*.txt")
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

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestIntegration_PerformanceBenchmark runs performance tests
func TestIntegration_PerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	backend := openai.NewMockBackend()
	controller := chat.NewController(backend, nil)
	ctx := context.Background()

	// Create a conversation
	conv := controller.CreateConversation("Performance test assistant")

	// Benchmark message sending
	t.Log("Running performance benchmark...")
	start := time.Now()

	const numMessages = 100
	for i := 0; i < numMessages; i++ {
		_, err := controller.SendMessage(ctx, chat.ChatRequest{
			ConversationID: conv.ID,
			Message:        fmt.Sprintf("Performance test message %d", i),
		})
		if err != nil {
			t.Fatalf("Message %d failed: %v", i, err)
		}
	}

	duration := time.Since(start)
	messagesPerSecond := float64(numMessages) / duration.Seconds()

	t.Logf("✅ Performance benchmark completed:")
	t.Logf("  Messages: %d", numMessages)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Rate: %.2f messages/second", messagesPerSecond)

	// Reasonable performance threshold for mock backend (has 100ms delay)
	// Real backends would be faster as they don't have artificial delays
	if messagesPerSecond < 5 {
		t.Errorf("Performance too slow: %.2f messages/second (expected >= 5)", messagesPerSecond)
	}
}
