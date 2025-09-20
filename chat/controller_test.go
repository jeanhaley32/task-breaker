package chat

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jeanhaley/task-breaker/backends/mock"
)

func TestNewController(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)

	if controller == nil {
		t.Fatal("NewController() returned nil")
	}

	if controller.backend != backend {
		t.Error("Controller should use provided backend")
	}

	if controller.defaultModel != "gpt-4" {
		t.Errorf("Expected default model 'gpt-4', got '%s'", controller.defaultModel)
	}
}

func TestController_CreateConversation(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)

	// Test without system prompt
	conv1 := controller.CreateConversation("")
	if conv1 == nil {
		t.Fatal("CreateConversation() returned nil")
	}

	if len(conv1.Messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(conv1.Messages))
	}

	// Test with system prompt
	systemPrompt := "You are a helpful assistant"
	conv2 := controller.CreateConversation(systemPrompt)
	if conv2 == nil {
		t.Fatal("CreateConversation() returned nil")
	}

	if len(conv2.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(conv2.Messages))
	}

	if conv2.Messages[0].Role != "system" {
		t.Errorf("Expected system role, got '%s'", conv2.Messages[0].Role)
	}

	if conv2.Messages[0].Content != systemPrompt {
		t.Errorf("Expected system prompt '%s', got '%s'", systemPrompt, conv2.Messages[0].Content)
	}

	// Test that IDs are unique
	if conv1.ID == conv2.ID {
		t.Error("Conversation IDs should be unique")
	}
}

func TestController_SendMessage(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)
	ctx := context.Background()

	// Test sending message without existing conversation
	request := ChatRequest{
		Message: "Hello, world!",
	}

	response, err := controller.SendMessage(ctx, request)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.ConversationID == "" {
		t.Error("Response should include conversation ID")
	}

	if response.Message.Role != "assistant" {
		t.Errorf("Expected assistant role, got '%s'", response.Message.Role)
	}

	if response.Message.Content == "" {
		t.Error("Response content should not be empty")
	}

	// Verify conversation was created and has messages
	conv, err := controller.GetConversation(response.ConversationID)
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	if len(conv.Messages) != 2 {
		t.Errorf("Expected 2 messages in conversation, got %d", len(conv.Messages))
	}

	// Check user message
	if conv.Messages[0].Role != "user" {
		t.Errorf("Expected first message to be user, got '%s'", conv.Messages[0].Role)
	}

	if conv.Messages[0].Content != "Hello, world!" {
		t.Errorf("Expected user message 'Hello, world!', got '%s'", conv.Messages[0].Content)
	}

	// Check assistant message
	if conv.Messages[1].Role != "assistant" {
		t.Errorf("Expected second message to be assistant, got '%s'", conv.Messages[1].Role)
	}
}

func TestController_SendMessage_ExistingConversation(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)
	ctx := context.Background()

	// Create conversation
	conv := controller.CreateConversation("You are helpful")

	// Send first message
	request1 := ChatRequest{
		ConversationID: conv.ID,
		Message:        "What is 2+2?",
	}

	response1, err := controller.SendMessage(ctx, request1)
	if err != nil {
		t.Fatalf("First message failed: %v", err)
	}

	// Send second message
	request2 := ChatRequest{
		ConversationID: conv.ID,
		Message:        "What about 3+3?",
	}

	response2, err := controller.SendMessage(ctx, request2)
	if err != nil {
		t.Fatalf("Second message failed: %v", err)
	}

	// Verify both responses are from same conversation
	if response1.ConversationID != response2.ConversationID {
		t.Error("Both messages should be in same conversation")
	}

	// Verify conversation has all messages
	updatedConv, err := controller.GetConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to get updated conversation: %v", err)
	}

	expectedMessages := 5 // system + user1 + assistant1 + user2 + assistant2
	if len(updatedConv.Messages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, len(updatedConv.Messages))
	}
}

func TestController_GetConversation(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)

	// Test getting non-existent conversation
	_, err := controller.GetConversation("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent conversation")
	}

	// Test getting existing conversation
	conv := controller.CreateConversation("Test system prompt")
	retrieved, err := controller.GetConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to get existing conversation: %v", err)
	}

	if retrieved.ID != conv.ID {
		t.Error("Retrieved conversation ID should match")
	}

	if len(retrieved.Messages) != len(conv.Messages) {
		t.Error("Retrieved conversation should have same message count")
	}
}

func TestController_ListConversations(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)

	// Initially should be empty
	conversations := controller.ListConversations()
	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(conversations))
	}

	// Create some conversations
	conv1 := controller.CreateConversation("System 1")
	time.Sleep(1 * time.Millisecond) // Ensure unique timestamps
	conv2 := controller.CreateConversation("System 2")
	time.Sleep(1 * time.Millisecond)
	conv3 := controller.CreateConversation("")

	// List should now contain all conversations
	conversations = controller.ListConversations()
	if len(conversations) != 3 {
		t.Errorf("Expected 3 conversations, got %d", len(conversations))
	}

	// Verify all IDs are present
	ids := make(map[ConversationID]bool)
	for _, conv := range conversations {
		ids[conv.ID] = true
	}

	if !ids[conv1.ID] || !ids[conv2.ID] || !ids[conv3.ID] {
		t.Error("All created conversations should be in list")
	}
}

func TestController_DeleteConversation(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)

	// Test deleting non-existent conversation
	err := controller.DeleteConversation("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent conversation")
	}

	// Test deleting existing conversation
	conv := controller.CreateConversation("Test")
	err = controller.DeleteConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to delete conversation: %v", err)
	}

	// Verify conversation is gone
	_, err = controller.GetConversation(conv.ID)
	if err == nil {
		t.Error("Conversation should no longer exist after deletion")
	}

	// Verify it's not in list
	conversations := controller.ListConversations()
	for _, c := range conversations {
		if c.ID == conv.ID {
			t.Error("Deleted conversation should not appear in list")
		}
	}
}

func TestController_ClearConversation(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)
	ctx := context.Background()

	// Create conversation with system prompt
	conv := controller.CreateConversation("You are helpful")

	// Send some messages
	controller.SendMessage(ctx, ChatRequest{
		ConversationID: conv.ID,
		Message:        "First message",
	})

	controller.SendMessage(ctx, ChatRequest{
		ConversationID: conv.ID,
		Message:        "Second message",
	})

	// Verify conversation has messages
	updatedConv, _ := controller.GetConversation(conv.ID)
	if len(updatedConv.Messages) != 5 { // system + user + assistant + user + assistant
		t.Errorf("Expected 5 messages before clear, got %d", len(updatedConv.Messages))
	}

	// Clear conversation
	err := controller.ClearConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to clear conversation: %v", err)
	}

	// Verify only system message remains
	clearedConv, _ := controller.GetConversation(conv.ID)
	if len(clearedConv.Messages) != 1 {
		t.Errorf("Expected 1 message after clear, got %d", len(clearedConv.Messages))
	}

	if clearedConv.Messages[0].Role != "system" {
		t.Error("Remaining message should be system message")
	}
}

func TestController_GetConversationSummary(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)
	ctx := context.Background()

	// Create conversation and send messages
	conv := controller.CreateConversation("You are helpful")

	controller.SendMessage(ctx, ChatRequest{
		ConversationID: conv.ID,
		Message:        "Hello",
	})

	controller.SendMessage(ctx, ChatRequest{
		ConversationID: conv.ID,
		Message:        "How are you?",
	})

	// Get summary
	summary, err := controller.GetConversationSummary(conv.ID)
	if err != nil {
		t.Fatalf("Failed to get conversation summary: %v", err)
	}

	if summary.ID != conv.ID {
		t.Error("Summary ID should match conversation ID")
	}

	if summary.MessageCount != 5 { // system + 2*(user + assistant)
		t.Errorf("Expected 5 total messages, got %d", summary.MessageCount)
	}

	if summary.SystemMessages != 1 {
		t.Errorf("Expected 1 system message, got %d", summary.SystemMessages)
	}

	if summary.UserMessages != 2 {
		t.Errorf("Expected 2 user messages, got %d", summary.UserMessages)
	}

	if summary.AssistantMessages != 2 {
		t.Errorf("Expected 2 assistant messages, got %d", summary.AssistantMessages)
	}

	if summary.LastUserMessage != "How are you?" {
		t.Errorf("Expected last user message 'How are you?', got '%s'", summary.LastUserMessage)
	}

	if !strings.Contains(summary.LastAssistantMessage, "Mock AI") {
		t.Error("Last assistant message should contain mock response")
	}

	if summary.EstimatedTokens <= 0 {
		t.Error("Estimated tokens should be positive")
	}
}

func TestController_SetBackend(t *testing.T) {
	backend1 := mock.NewMockBackend()
	backend2 := mock.NewMockBackend()

	controller := NewController(backend1, nil)

	if controller.GetBackend() != backend1 {
		t.Error("Initial backend should be backend1")
	}

	controller.SetBackend(backend2)

	if controller.GetBackend() != backend2 {
		t.Error("Backend should be updated to backend2")
	}
}

func TestController_IsBackendAvailable(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)
	ctx := context.Background()

	available := controller.IsBackendAvailable(ctx)
	if !available {
		t.Error("Mock backend should always be available")
	}
}

func TestController_GetStats(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)
	ctx := context.Background()

	// Initial stats
	stats := controller.GetStats()
	if stats.TotalConversations != 0 {
		t.Errorf("Expected 0 conversations initially, got %d", stats.TotalConversations)
	}

	if stats.TotalMessages != 0 {
		t.Errorf("Expected 0 messages initially, got %d", stats.TotalMessages)
	}

	if stats.BackendName != "MockAI" {
		t.Errorf("Expected backend name 'MockAI', got '%s'", stats.BackendName)
	}

	// Create conversations and messages
	conv1 := controller.CreateConversation("System 1")
	conv2 := controller.CreateConversation("")

	controller.SendMessage(ctx, ChatRequest{
		ConversationID: conv1.ID,
		Message:        "Test message 1",
	})

	controller.SendMessage(ctx, ChatRequest{
		ConversationID: conv2.ID,
		Message:        "Test message 2",
	})

	// Updated stats
	stats = controller.GetStats()
	if stats.TotalConversations != 2 {
		t.Errorf("Expected 2 conversations, got %d", stats.TotalConversations)
	}

	expectedMessages := 5 // system + user + assistant + user + assistant
	if stats.TotalMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, stats.TotalMessages)
	}
}

func TestController_ConcurrentAccess(t *testing.T) {
	backend := mock.NewMockBackend()
	controller := NewController(backend, nil)
	ctx := context.Background()

	// Test concurrent conversation creation
	const numGoroutines = 10
	conversations := make(chan *Conversation, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			conv := controller.CreateConversation("Concurrent test")
			conversations <- conv
		}()
	}

	// Collect results
	created := make([]*Conversation, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		conv := <-conversations
		created = append(created, conv)
	}

	// Verify all conversations were created with unique IDs
	ids := make(map[ConversationID]bool)
	for _, conv := range created {
		if ids[conv.ID] {
			t.Error("Duplicate conversation ID found")
		}
		ids[conv.ID] = true
	}

	// Test concurrent message sending
	conv := controller.CreateConversation("Concurrent messages")
	responses := make(chan *ChatResponse, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(msgNum int) {
			response, err := controller.SendMessage(ctx, ChatRequest{
				ConversationID: conv.ID,
				Message:        fmt.Sprintf("Message %d", msgNum),
			})
			if err != nil {
				errors <- err
			} else {
				responses <- response
			}
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numGoroutines; i++ {
		select {
		case <-responses:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	if successCount != numGoroutines {
		t.Errorf("Expected %d successful messages, got %d (errors: %d)", numGoroutines, successCount, errorCount)
	}
}
