package openai

import (
	openai "github.com/jeanhaley32/go-openai-client"
)

// OpenAIBackend is an alias for the library's Client to maintain API compatibility
type OpenAIBackend = openai.Client

// Config is an alias for the library's Config to maintain API compatibility
type Config = openai.Config

// NewOpenAIBackend creates a new OpenAI backend instance using the library
func NewOpenAIBackend(config Config) *OpenAIBackend {
	return openai.NewClient(config)
}

// Model is an alias for the library's Model to maintain API compatibility
type Model = openai.Model