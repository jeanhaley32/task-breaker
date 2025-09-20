package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/jeanhaley/task-breaker/ai"
	"github.com/jeanhaley/task-breaker/backends/mock"
)

type Agent struct {
	name      string
	context   string
	aiBackend ai.Backend
}

func NewAgent(name string, backend ai.Backend) *Agent {
	return &Agent{
		name:      name,
		aiBackend: backend,
	}
}

func (a *Agent) LoadContext(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open context file %s: %w", filename, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read context file %s: %w", filename, err)
	}

	a.context = string(content)
	return nil
}

func (a *Agent) PrintContext() {
	fmt.Printf("=== Agent: %s ===\n", a.name)
	fmt.Printf("Context:\n%s\n", a.context)
	fmt.Println("=================")
}

func (a *Agent) SendMessage(message string) (*ai.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the request
	req := ai.Request{
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: message,
			},
		},
		MaxTokens:   &[]int{150}[0],
		Temperature: &[]float64{0.7}[0],
	}

	return a.aiBackend.SendMessage(ctx, req)
}

func (a *Agent) SendChatCompletion(messages []ai.Message) (*ai.ChatCompletionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Add system message with context if available
	allMessages := messages
	if a.context != "" {
		systemMessage := ai.Message{
			Role:    "system",
			Content: a.context,
		}
		allMessages = append([]ai.Message{systemMessage}, messages...)
	}

	// Create OpenAI Chat Completions request
	req := ai.ChatCompletionRequest{
		Model:       "mock-model-v1",
		Messages:    allMessages,
		MaxTokens:   &[]int{150}[0],
		Temperature: &[]float64{0.7}[0],
	}

	return a.aiBackend.ChatCompletion(ctx, req)
}

func main() {
	// Initialize the mock backend
	backend := mock.NewMockBackend()

	// Check if backend is available
	ctx := context.Background()
	if !backend.IsAvailable(ctx) {
		log.Fatal("AI backend is not available")
	}

	fmt.Printf("Using AI backend: %s\n\n", backend.Name())

	// Create agent with AI backend
	agent := NewAgent("TaskBreakerAgent", backend)

	// Load context if provided
	if len(os.Args) >= 2 {
		contextFile := os.Args[1]
		if err := agent.LoadContext(contextFile); err != nil {
			log.Printf("Warning: Could not load context file: %v", err)
		} else {
			fmt.Println("Context loaded successfully")
			agent.PrintContext()
		}
	}

	// Test 1: Legacy SendMessage method
	fmt.Println("=== Test 1: Legacy Method ===")
	fmt.Println("Sending 'Hello World' using legacy method...")
	legacyResponse, err := agent.SendMessage("Hello World")
	if err != nil {
		log.Fatalf("Error sending legacy message: %v", err)
	}

	fmt.Printf("Content: %s\n", legacyResponse.Content)
	fmt.Printf("Model: %s\n", legacyResponse.Model)
	fmt.Printf("Tokens Used: %d\n", legacyResponse.TokensUsed)
	fmt.Printf("Timestamp: %s\n", legacyResponse.Timestamp.Format(time.RFC3339))

	// Test 2: OpenAI Chat Completions method
	fmt.Println("\n=== Test 2: OpenAI Chat Completions ===")
	fmt.Println("Sending conversation using OpenAI Chat Completions format...")

	messages := []ai.Message{
		{Role: "user", Content: "Hello World"},
		{Role: "assistant", Content: "Hello! How can I help you today?"},
		{Role: "user", Content: "Can you tell me about task breaking?"},
	}

	chatResponse, err := agent.SendChatCompletion(messages)
	if err != nil {
		log.Fatalf("Error sending chat completion: %v", err)
	}

	fmt.Printf("ID: %s\n", chatResponse.ID)
	fmt.Printf("Object: %s\n", chatResponse.Object)
	fmt.Printf("Model: %s\n", chatResponse.Model)
	fmt.Printf("Created: %d\n", chatResponse.Created)

	if len(chatResponse.Choices) > 0 {
		choice := chatResponse.Choices[0]
		fmt.Printf("Response: %s\n", choice.Message.Content)
		fmt.Printf("Finish Reason: %s\n", choice.FinishReason)
	}

	fmt.Printf("Usage - Prompt: %d, Completion: %d, Total: %d tokens\n",
		chatResponse.Usage.PromptTokens,
		chatResponse.Usage.CompletionTokens,
		chatResponse.Usage.TotalTokens)

	fmt.Println("==========================================")
}
