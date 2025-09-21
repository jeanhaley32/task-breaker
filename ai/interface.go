package ai

import (
	openai "github.com/jeanhaley32/go-openai-client"
)

// Type aliases for backward compatibility - all types now come from the library

// Message represents a single message in a conversation following OpenAI Chat Completions format.
type Message = openai.Message

// ChatCompletionRequest represents a request following OpenAI Chat Completions API standard.
type ChatCompletionRequest = openai.ChatCompletionRequest

// Usage represents token usage information from the AI provider.
type Usage = openai.Usage

// Choice represents a single completion choice from the AI.
type Choice = openai.Choice

// ChatCompletionResponse represents a response following OpenAI Chat Completions API standard.
type ChatCompletionResponse = openai.ChatCompletionResponse

// Legacy types for backward compatibility
type Request = openai.Request
type Response = openai.Response

// Backend defines the interface that all AI backends must implement.
type Backend = openai.Backend