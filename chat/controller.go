package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jeanhaley/task-breaker/ai"
)

// ConversationID represents a unique identifier for a conversation
type ConversationID string

// Conversation represents an active chat session with message history
type Conversation struct {
	ID        ConversationID    `json:"id"`
	Messages  []ai.Message      `json:"messages"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata"`
}

// ChatRequest represents a request to send a message in a conversation
type ChatRequest struct {
	ConversationID ConversationID `json:"conversation_id,omitempty"`
	Message        string         `json:"message"`
	SystemPrompt   string         `json:"system_prompt,omitempty"`
	Model          string         `json:"model,omitempty"`
	MaxTokens      *int           `json:"max_tokens,omitempty"`
	Temperature    *float64       `json:"temperature,omitempty"`
}

// ChatResponse represents the response from the chat controller
type ChatResponse struct {
	ConversationID ConversationID             `json:"conversation_id"`
	Message        ai.Message                 `json:"message"`
	Response       *ai.ChatCompletionResponse `json:"response"`
	Error          string                     `json:"error,omitempty"`
}

// Controller manages chat conversations and AI backend interactions
type Controller struct {
	backend       ai.Backend
	conversations map[ConversationID]*Conversation
	mutex         sync.RWMutex
	defaultModel  string
	maxTokens     int
	temperature   float64
}

// ControllerConfig holds configuration for the chat controller
type ControllerConfig struct {
	DefaultModel string  `json:"default_model"`
	MaxTokens    int     `json:"max_tokens"`
	Temperature  float64 `json:"temperature"`
}

// NewController creates a new chat controller with the specified backend
func NewController(backend ai.Backend, config *ControllerConfig) *Controller {
	if config == nil {
		config = &ControllerConfig{
			DefaultModel: "gpt-4",
			MaxTokens:    500,
			Temperature:  0.7,
		}
	}

	return &Controller{
		backend:       backend,
		conversations: make(map[ConversationID]*Conversation),
		defaultModel:  config.DefaultModel,
		maxTokens:     config.MaxTokens,
		temperature:   config.Temperature,
	}
}

// CreateConversation creates a new conversation with optional system prompt
func (c *Controller) CreateConversation(systemPrompt string) *Conversation {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	id := ConversationID(fmt.Sprintf("conv_%d_%d", time.Now().UnixNano(), len(c.conversations)))
	conversation := &Conversation{
		ID:        id,
		Messages:  make([]ai.Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}

	if systemPrompt != "" {
		conversation.Messages = append(conversation.Messages, ai.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	c.conversations[id] = conversation
	return conversation
}

// GetConversation retrieves a conversation by ID
func (c *Controller) GetConversation(id ConversationID) (*Conversation, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	conversation, exists := c.conversations[id]
	if !exists {
		return nil, fmt.Errorf("conversation %s not found", id)
	}

	return conversation, nil
}

// ListConversations returns all conversations
func (c *Controller) ListConversations() []*Conversation {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	conversations := make([]*Conversation, 0, len(c.conversations))
	for _, conv := range c.conversations {
		conversations = append(conversations, conv)
	}

	return conversations
}

// DeleteConversation removes a conversation
func (c *Controller) DeleteConversation(id ConversationID) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.conversations[id]; !exists {
		return fmt.Errorf("conversation %s not found", id)
	}

	delete(c.conversations, id)
	return nil
}

// SendMessage sends a message and gets a response from the AI backend
func (c *Controller) SendMessage(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	// Get or create conversation
	var conversation *Conversation
	var err error

	if request.ConversationID != "" {
		conversation, err = c.GetConversation(request.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversation: %w", err)
		}
	} else {
		conversation = c.CreateConversation(request.SystemPrompt)
	}

	// Add user message to conversation and prepare AI request
	userMessage := ai.Message{
		Role:    "user",
		Content: request.Message,
	}

	// Prepare model parameters
	model := request.Model
	if model == "" {
		model = c.defaultModel
	}

	maxTokens := request.MaxTokens
	if maxTokens == nil {
		maxTokens = &c.maxTokens
	}

	temperature := request.Temperature
	if temperature == nil {
		temperature = &c.temperature
	}

	// Update conversation and create AI request atomically
	c.mutex.Lock()
	conversation.Messages = append(conversation.Messages, userMessage)
	conversation.UpdatedAt = time.Now()

	// Create a copy of messages for the AI request to avoid holding the lock during API call
	messagesCopy := make([]ai.Message, len(conversation.Messages))
	copy(messagesCopy, conversation.Messages)
	c.mutex.Unlock()

	aiRequest := ai.ChatCompletionRequest{
		Model:       model,
		Messages:    messagesCopy,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Send request to AI backend
	response, err := c.backend.ChatCompletion(ctx, aiRequest)
	if err != nil {
		return &ChatResponse{
			ConversationID: conversation.ID,
			Message:        userMessage,
			Error:          err.Error(),
		}, err
	}

	// Extract assistant message from response
	if len(response.Choices) == 0 {
		return &ChatResponse{
			ConversationID: conversation.ID,
			Message:        userMessage,
			Error:          "no response choices returned",
		}, fmt.Errorf("no response choices returned")
	}

	assistantMessage := response.Choices[0].Message

	// Add assistant response to conversation
	c.mutex.Lock()
	conversation.Messages = append(conversation.Messages, assistantMessage)
	conversation.UpdatedAt = time.Now()
	c.mutex.Unlock()

	return &ChatResponse{
		ConversationID: conversation.ID,
		Message:        assistantMessage,
		Response:       response,
	}, nil
}

// ClearConversation removes all messages from a conversation except system messages
func (c *Controller) ClearConversation(id ConversationID) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conversation, exists := c.conversations[id]
	if !exists {
		return fmt.Errorf("conversation %s not found", id)
	}

	// Keep only system messages
	systemMessages := make([]ai.Message, 0)
	for _, msg := range conversation.Messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		}
	}

	conversation.Messages = systemMessages
	conversation.UpdatedAt = time.Now()

	return nil
}

// GetConversationSummary returns a summary of the conversation
func (c *Controller) GetConversationSummary(id ConversationID) (*ConversationSummary, error) {
	conversation, err := c.GetConversation(id)
	if err != nil {
		return nil, err
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var userMessages, assistantMessages, systemMessages int
	var totalTokens int

	for _, msg := range conversation.Messages {
		switch msg.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		case "system":
			systemMessages++
		}
		// Rough token estimation
		totalTokens += len(msg.Content) / 4
	}

	return &ConversationSummary{
		ID:                   conversation.ID,
		MessageCount:         len(conversation.Messages),
		UserMessages:         userMessages,
		AssistantMessages:    assistantMessages,
		SystemMessages:       systemMessages,
		EstimatedTokens:      totalTokens,
		CreatedAt:            conversation.CreatedAt,
		UpdatedAt:            conversation.UpdatedAt,
		LastUserMessage:      getLastMessageByRole(conversation.Messages, "user"),
		LastAssistantMessage: getLastMessageByRole(conversation.Messages, "assistant"),
	}, nil
}

// ConversationSummary provides overview information about a conversation
type ConversationSummary struct {
	ID                   ConversationID `json:"id"`
	MessageCount         int            `json:"message_count"`
	UserMessages         int            `json:"user_messages"`
	AssistantMessages    int            `json:"assistant_messages"`
	SystemMessages       int            `json:"system_messages"`
	EstimatedTokens      int            `json:"estimated_tokens"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	LastUserMessage      string         `json:"last_user_message"`
	LastAssistantMessage string         `json:"last_assistant_message"`
}

// SetBackend allows changing the AI backend at runtime
func (c *Controller) SetBackend(backend ai.Backend) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.backend = backend
}

// GetBackend returns the current AI backend
func (c *Controller) GetBackend() ai.Backend {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.backend
}

// IsBackendAvailable checks if the current backend is available
func (c *Controller) IsBackendAvailable(ctx context.Context) bool {
	c.mutex.RLock()
	backend := c.backend
	c.mutex.RUnlock()

	return backend.IsAvailable(ctx)
}

// GetStats returns controller statistics
func (c *Controller) GetStats() ControllerStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var totalMessages, totalConversations int
	oldestConversation := time.Now()
	newestConversation := time.Time{}

	totalConversations = len(c.conversations)

	for _, conv := range c.conversations {
		totalMessages += len(conv.Messages)
		if conv.CreatedAt.Before(oldestConversation) {
			oldestConversation = conv.CreatedAt
		}
		if conv.UpdatedAt.After(newestConversation) {
			newestConversation = conv.UpdatedAt
		}
	}

	return ControllerStats{
		TotalConversations: totalConversations,
		TotalMessages:      totalMessages,
		BackendName:        c.backend.Name(),
		OldestConversation: oldestConversation,
		NewestConversation: newestConversation,
	}
}

// ControllerStats provides statistics about the controller
type ControllerStats struct {
	TotalConversations int       `json:"total_conversations"`
	TotalMessages      int       `json:"total_messages"`
	BackendName        string    `json:"backend_name"`
	OldestConversation time.Time `json:"oldest_conversation"`
	NewestConversation time.Time `json:"newest_conversation"`
}

// Helper function to get the last message of a specific role
func getLastMessageByRole(messages []ai.Message, role string) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == role {
			return messages[i].Content
		}
	}
	return ""
}
