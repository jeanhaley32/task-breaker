package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jeanhaley/task-breaker/ai"
	"github.com/jeanhaley/task-breaker/backends/mock"
	"github.com/jeanhaley/task-breaker/backends/openai"
	"github.com/jeanhaley/task-breaker/chat"
	"github.com/jeanhaley/task-breaker/config"
)

func main() {
	// Load configuration
	configManager := config.NewManager("")
	if err := configManager.Load(); err != nil {
		// First run, initialize config
		if err := configManager.InitializeConfig(); err != nil {
			log.Fatalf("Failed to initialize configuration: %v", err)
		}
	}

	cfg := configManager.GetConfig()

	// Validate configuration
	if err := configManager.ValidateConfig(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize backend based on configuration
	var backend ai.Backend

	switch cfg.Default.Backend {
	case "openai":
		if cfg.OpenAI.APIKey == "" {
			log.Fatal("OpenAI API key not configured. Set OPENAI_API_KEY environment variable.")
		}
		backend = openai.NewOpenAIBackend(openai.Config{
			APIKey:     cfg.OpenAI.APIKey,
			BaseURL:    cfg.OpenAI.BaseURL,
			Model:      cfg.OpenAI.Model,
			Timeout:    cfg.OpenAI.Timeout,
			MaxRetries: cfg.OpenAI.MaxRetries,
		})
	case "mock":
		backend = mock.NewMockBackend()
	default:
		log.Fatalf("Unknown backend: %s", cfg.Default.Backend)
	}

	// Check backend availability
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !backend.IsAvailable(ctx) {
		log.Printf("Warning: Backend '%s' is not available", backend.Name())
		if cfg.Default.Backend != "mock" {
			log.Println("Falling back to mock backend")
			backend = mock.NewMockBackend()
		}
	}

	// Initialize chat controller
	controller := chat.NewController(backend, &chat.ControllerConfig{
		DefaultModel: cfg.ChatController.DefaultModel,
		MaxTokens:    cfg.ChatController.MaxTokens,
		Temperature:  cfg.ChatController.Temperature,
	})

	// Start interactive chat session
	fmt.Printf("🤖 Task Breaker Chat Interface\n")
	fmt.Printf("Backend: %s\n", backend.Name())
	fmt.Printf("Model: %s\n", cfg.Default.Model)
	fmt.Printf("\nType your message and press Enter. Type 'quit' to exit.\n")
	fmt.Printf("Commands: /new, /list, /clear, /stats, /help\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	var currentConversation *chat.Conversation

	// Create initial conversation
	systemPrompt := loadSystemPrompt()
	currentConversation = controller.CreateConversation(systemPrompt)
	fmt.Printf("Started new conversation: %s\n\n", currentConversation.ID)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			handleCommand(input, controller, &currentConversation, cfg)
			continue
		}

		// Handle quit
		if input == "quit" || input == "exit" {
			fmt.Println("Goodbye! 👋")
			break
		}

		// Send message
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		response, err := controller.SendMessage(ctx, chat.ChatRequest{
			ConversationID: currentConversation.ID,
			Message:        input,
			Model:          cfg.Default.Model,
		})
		cancel()

		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}

		// Display response
		fmt.Printf("🤖 %s: %s\n\n", backend.Name(), response.Message.Content)

		// Show token usage if available
		if response.Response != nil {
			usage := response.Response.Usage
			fmt.Printf("📊 Tokens: %d prompt + %d completion = %d total\n\n",
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
}

func handleCommand(command string, controller *chat.Controller, currentConv **chat.Conversation, cfg *config.Config) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/new":
		// Create new conversation
		systemPrompt := loadSystemPrompt()
		*currentConv = controller.CreateConversation(systemPrompt)
		fmt.Printf("✓ Started new conversation: %s\n\n", (*currentConv).ID)

	case "/list":
		// List all conversations
		conversations := controller.ListConversations()
		fmt.Printf("📋 Conversations (%d total):\n", len(conversations))
		for _, conv := range conversations {
			summary, err := controller.GetConversationSummary(conv.ID)
			if err != nil {
				fmt.Printf("  %s (error getting summary)\n", conv.ID)
				continue
			}

			status := ""
			if conv.ID == (*currentConv).ID {
				status = " [CURRENT]"
			}

			fmt.Printf("  %s%s - %d messages, updated %s\n",
				conv.ID, status, summary.MessageCount, summary.UpdatedAt.Format("15:04:05"))

			if summary.LastUserMessage != "" {
				preview := summary.LastUserMessage
				if len(preview) > 50 {
					preview = preview[:50] + "..."
				}
				fmt.Printf("    Last: %s\n", preview)
			}
		}
		fmt.Println()

	case "/clear":
		// Clear current conversation
		if err := controller.ClearConversation((*currentConv).ID); err != nil {
			fmt.Printf("❌ Error clearing conversation: %v\n\n", err)
		} else {
			fmt.Printf("✓ Cleared conversation %s\n\n", (*currentConv).ID)
		}

	case "/stats":
		// Show controller statistics
		stats := controller.GetStats()
		fmt.Printf("📊 Chat Statistics:\n")
		fmt.Printf("  Backend: %s\n", stats.BackendName)
		fmt.Printf("  Total Conversations: %d\n", stats.TotalConversations)
		fmt.Printf("  Total Messages: %d\n", stats.TotalMessages)
		if stats.TotalConversations > 0 {
			fmt.Printf("  Oldest: %s\n", stats.OldestConversation.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Newest: %s\n", stats.NewestConversation.Format("2006-01-02 15:04:05"))
		}

		// Backend availability
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		available := controller.IsBackendAvailable(ctx)
		cancel()

		if available {
			fmt.Printf("  Backend Status: ✅ Available\n")
		} else {
			fmt.Printf("  Backend Status: ❌ Unavailable\n")
		}
		fmt.Println()

	case "/switch":
		// Switch backend
		if len(parts) < 2 {
			fmt.Printf("Usage: /switch <backend>\nAvailable: openai, mock\n\n")
			return
		}

		var newBackend ai.Backend
		switch parts[1] {
		case "openai":
			if cfg.OpenAI.APIKey == "" {
				fmt.Printf("❌ OpenAI API key not configured\n\n")
				return
			}
			newBackend = openai.NewOpenAIBackend(openai.Config{
				APIKey:  cfg.OpenAI.APIKey,
				BaseURL: cfg.OpenAI.BaseURL,
				Model:   cfg.OpenAI.Model,
				Timeout: cfg.OpenAI.Timeout,
			})
		case "mock":
			newBackend = mock.NewMockBackend()
		default:
			fmt.Printf("❌ Unknown backend: %s\n\n", parts[1])
			return
		}

		// Test availability
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if !newBackend.IsAvailable(ctx) {
			cancel()
			fmt.Printf("❌ Backend '%s' is not available\n\n", parts[1])
			return
		}
		cancel()

		controller.SetBackend(newBackend)
		fmt.Printf("✓ Switched to %s backend\n\n", newBackend.Name())

	case "/help":
		fmt.Printf("🤖 Task Breaker Commands:\n")
		fmt.Printf("  /new          - Start a new conversation\n")
		fmt.Printf("  /list         - List all conversations\n")
		fmt.Printf("  /clear        - Clear current conversation\n")
		fmt.Printf("  /stats        - Show statistics\n")
		fmt.Printf("  /switch <be>  - Switch backend (openai, mock)\n")
		fmt.Printf("  /help         - Show this help\n")
		fmt.Printf("  quit/exit     - Exit the chat\n\n")

	default:
		fmt.Printf("❌ Unknown command: %s\nType /help for available commands\n\n", parts[0])
	}
}

func loadSystemPrompt() string {
	// Try to load system prompt from file
	if _, err := os.Stat("system-prompt.txt"); err == nil {
		data, err := os.ReadFile("system-prompt.txt")
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// Default system prompt
	return "You are a helpful AI assistant built with Task Breaker. You are knowledgeable, concise, and always try to provide accurate information."
}
