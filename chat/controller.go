package chat

import (
	openai "github.com/jeanhaley32/go-openai-client"
	"github.com/jeanhaley32/go-openai-client/chat"
)

// Type aliases for backward compatibility - all types now come from the library

// ConversationID represents a unique identifier for a conversation
type ConversationID = chat.ConversationID

// Conversation represents an active chat session with message history
type Conversation = chat.Conversation

// ChatRequest represents a request to send a message in a conversation
type ChatRequest = chat.ChatRequest

// ChatResponse represents the response from the chat controller
type ChatResponse = chat.ChatResponse

// Controller manages chat conversations and AI backend interactions
type Controller = chat.Controller

// ControllerConfig holds configuration for the chat controller
type ControllerConfig = chat.ControllerConfig

// ConversationSummary provides overview information about a conversation
type ConversationSummary = chat.ConversationSummary

// ControllerStats provides statistics about the controller
type ControllerStats = chat.ControllerStats

// NewController creates a new chat controller with the specified backend
func NewController(backend openai.Backend, config *ControllerConfig) *Controller {
	return chat.NewController(backend, config)
}