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
	name     string
	context  string
	aiBackend ai.Backend
}

func NewAgent(name string, backend ai.Backend) *Agent {
	return &Agent{
		name:     name,
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
				Role:      "user",
				Content:   message,
				Timestamp: time.Now(),
			},
		},
		MaxTokens:   150,
		Temperature: 0.7,
		Context:     a.context,
	}

	return a.aiBackend.SendMessage(ctx, req)
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

	// Send Hello World message
	fmt.Println("Sending 'Hello World' to AI backend...")
	response, err := agent.SendMessage("Hello World")
	if err != nil {
		log.Fatalf("Error sending message: %v", err)
	}

	// Display the response
	fmt.Printf("\n=== AI Response ===\n")
	fmt.Printf("Content: %s\n", response.Content)
	fmt.Printf("Model: %s\n", response.Model)
	fmt.Printf("Tokens Used: %d\n", response.TokensUsed)
	fmt.Printf("Timestamp: %s\n", response.Timestamp.Format(time.RFC3339))
	fmt.Println("==================")
}