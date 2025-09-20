# OpenAI Chat Completions Standard Protocol

This document explains the OpenAI Chat Completions API standard that our AI interface implements. This protocol has become the universal standard for communicating with AI language models across different providers.

## Overview

The OpenAI Chat Completions API is a standardized protocol for sending conversational requests to AI models and receiving structured responses. Originally developed by OpenAI for their GPT models, it has been adopted by most major AI providers including:

- **OpenAI** (GPT-4, GPT-3.5-turbo)
- **Anthropic Claude** (via compatibility layer)
- **Local Models** (Ollama, LM Studio, etc.)
- **Azure OpenAI**
- **Many open-source projects**

## Why This Standard Matters

### Universal Compatibility
By implementing this standard, your application can seamlessly switch between different AI providers without changing your core logic. This enables:

- **Provider Independence**: Switch from OpenAI to Claude to local models
- **Fallback Strategies**: Use multiple providers for redundancy
- **Cost Optimization**: Route requests to the most cost-effective provider
- **Feature Testing**: Compare responses across different models

### Industry Adoption
The Chat Completions format has become the "HTTP of AI APIs" - a common protocol that most tools and libraries expect. This means:

- Existing tools and libraries work out of the box
- Documentation and examples are widely available
- Developer knowledge transfers between projects
- Reduced integration complexity

## Protocol Structure

### Basic Request Format

```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant"},
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "max_tokens": 150,
  "temperature": 0.7
}
```

### Basic Response Format

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677858242,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! I'm doing well, thank you for asking. How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 13,
    "completion_tokens": 17,
    "total_tokens": 30
  }
}
```

## Message Roles and Flow

### The Three Sacred Roles

The protocol defines three message roles that establish the conversation structure:

#### 1. System Role (`"system"`)
- **Purpose**: Provides instructions and context to the AI
- **Position**: Usually the first message in the conversation
- **Content**: High-level instructions, persona definition, task context
- **Example**: `"You are a helpful coding assistant. Always provide working code examples."`

#### 2. User Role (`"user"`)
- **Purpose**: Represents human input or questions
- **Position**: Alternates with assistant messages
- **Content**: Questions, requests, follow-up clarifications
- **Example**: `"How do I implement a binary search in Go?"`

#### 3. Assistant Role (`"assistant"`)
- **Purpose**: Represents AI model responses
- **Position**: Follows user messages (except for conversation history)
- **Content**: AI-generated responses, answers, code examples
- **Example**: `"Here's a binary search implementation in Go..."`

### Conversation Flow Patterns

#### Simple Request-Response
```json
{
  "messages": [
    {"role": "user", "content": "What is 2+2?"}
  ]
}
```

#### With System Instructions
```json
{
  "messages": [
    {"role": "system", "content": "You are a math tutor. Explain your reasoning."},
    {"role": "user", "content": "What is 2+2?"}
  ]
}
```

#### Multi-turn Conversation
```json
{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hello! How can I help you?"},
    {"role": "user", "content": "Tell me about cats"},
    {"role": "assistant", "content": "Cats are fascinating animals..."},
    {"role": "user", "content": "What about dogs?"}
  ]
}
```

## Request Parameters

### Required Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `model` | string | Model identifier (e.g., "gpt-4", "claude-3-sonnet") |
| `messages` | array | Conversation history with at least one message |

### Optional Parameters

| Parameter | Type | Range | Default | Description |
|-----------|------|-------|---------|-------------|
| `max_tokens` | int | 1-∞ | Provider default | Maximum tokens in response |
| `temperature` | float | 0.0-2.0 | ~0.7-1.0 | Response randomness |
| `top_p` | float | 0.0-1.0 | 1.0 | Nucleus sampling |
| `stream` | bool | - | false | Enable response streaming |

### Parameter Usage Guidelines

#### Temperature Settings
- **0.0-0.3**: Deterministic, factual responses (code, math, structured data)
- **0.4-0.7**: Balanced creativity and consistency (most use cases)
- **0.8-1.2**: Creative writing, brainstorming, varied responses
- **1.3-2.0**: Highly creative, experimental (use with caution)

#### Token Limits
- **Conservative (50-150)**: Short answers, confirmations
- **Moderate (150-500)**: Explanations, code examples
- **Liberal (500-2000)**: Detailed analysis, long-form content
- **Maximum (2000+)**: Documents, comprehensive guides

## Response Structure

### Response Components

#### Metadata Fields
- `id`: Unique identifier for tracking and debugging
- `object`: Always "chat.completion" for this endpoint
- `created`: Unix timestamp for request timing
- `model`: Actual model used (may differ from requested)

#### Choices Array
- `index`: Position in choices array (usually 0)
- `message`: The AI's response with role "assistant"
- `finish_reason`: Why the response ended

#### Usage Information
- `prompt_tokens`: Input message token count
- `completion_tokens`: Response token count
- `total_tokens`: Sum for billing calculations

### Finish Reasons

| Reason | Meaning | Action Needed |
|--------|---------|---------------|
| `"stop"` | Natural completion | None - response is complete |
| `"length"` | Hit max_tokens limit | Consider increasing max_tokens |
| `"content_filter"` | Content was filtered | Revise input content |
| `"function_call"` | Model wants to call function | Handle function calling (if supported) |

## Implementation Best Practices

### Error Handling
```go
response, err := backend.ChatCompletion(ctx, request)
if err != nil {
    // Handle network/API errors
    return fmt.Errorf("chat completion failed: %w", err)
}

if len(response.Choices) == 0 {
    // Handle empty response
    return errors.New("no response choices returned")
}

choice := response.Choices[0]
if choice.FinishReason != "stop" {
    // Handle incomplete responses
    log.Warnf("Response incomplete: %s", choice.FinishReason)
}
```

### Context Management
```go
// Build conversation with system context
messages := []ai.Message{
    {Role: "system", Content: systemInstructions},
}

// Add conversation history
for _, msg := range conversationHistory {
    messages = append(messages, msg)
}

// Add current user input
messages = append(messages, ai.Message{
    Role: "user",
    Content: userInput,
})
```

### Token Management
```go
// Monitor token usage
if response.Usage.TotalTokens > warningThreshold {
    log.Warnf("High token usage: %d tokens", response.Usage.TotalTokens)
}

// Implement conversation trimming
if estimatedTokens > maxContextLength {
    messages = trimOldestMessages(messages, targetLength)
}
```

## Common Use Cases

### 1. Simple Q&A System
```go
request := ai.ChatCompletionRequest{
    Model: "gpt-4",
    Messages: []ai.Message{
        {Role: "user", Content: "What is the capital of France?"},
    },
    MaxTokens: &[]int{50}[0],
    Temperature: &[]float64{0.1}[0], // Low temperature for factual answers
}
```

### 2. Code Assistant
```go
request := ai.ChatCompletionRequest{
    Model: "gpt-4",
    Messages: []ai.Message{
        {Role: "system", Content: "You are a Go programming expert. Provide working code examples."},
        {Role: "user", Content: "How do I read a JSON file in Go?"},
    },
    MaxTokens: &[]int{500}[0],
    Temperature: &[]float64{0.3}[0], // Low-medium for consistent code
}
```

### 3. Creative Writing Assistant
```go
request := ai.ChatCompletionRequest{
    Model: "gpt-4",
    Messages: []ai.Message{
        {Role: "system", Content: "You are a creative writing assistant. Help with storytelling and narrative."},
        {Role: "user", Content: "Write a short story about a robot learning to paint."},
    },
    MaxTokens: &[]int{1000}[0],
    Temperature: &[]float64{0.9}[0], // High temperature for creativity
}
```

### 4. Multi-turn Conversation
```go
// Maintain conversation state
type Conversation struct {
    messages []ai.Message
}

func (c *Conversation) AddUserMessage(content string) {
    c.messages = append(c.messages, ai.Message{
        Role: "user",
        Content: content,
    })
}

func (c *Conversation) AddAssistantMessage(content string) {
    c.messages = append(c.messages, ai.Message{
        Role: "assistant",
        Content: content,
    })
}

func (c *Conversation) SendRequest(backend ai.Backend) (*ai.ChatCompletionResponse, error) {
    request := ai.ChatCompletionRequest{
        Model: "gpt-4",
        Messages: c.messages,
        MaxTokens: &[]int{300}[0],
        Temperature: &[]float64{0.7}[0],
    }

    return backend.ChatCompletion(context.Background(), request)
}
```

## Provider-Specific Considerations

### OpenAI
- Supports all standard parameters
- Function calling available on newer models
- Strict content filtering
- Pay-per-token pricing

### Anthropic Claude
- Compatibility layer may not support all parameters
- Different token limits per model
- Alternative safety approach
- Different pricing structure

### Local Models
- Parameter support varies by model
- No usage tracking/billing
- Hardware-dependent performance
- Full privacy control

## Migration Guide

### From Legacy SendMessage to ChatCompletion

**Old Code:**
```go
response, err := backend.SendMessage(ctx, ai.Request{
    Messages: []ai.Message{{Role: "user", Content: "Hello"}},
    MaxTokens: &[]int{150}[0],
    Temperature: &[]float64{0.7}[0],
})

content := response.Content
```

**New Code:**
```go
response, err := backend.ChatCompletion(ctx, ai.ChatCompletionRequest{
    Model: "gpt-4",
    Messages: []ai.Message{{Role: "user", Content: "Hello"}},
    MaxTokens: &[]int{150}[0],
    Temperature: &[]float64{0.7}[0],
})

content := response.Choices[0].Message.Content
```

### Benefits of Migration
- **Standardization**: Code works with any OpenAI-compatible provider
- **Rich Metadata**: Access to token usage, timing, and completion details
- **Future-Proof**: Supports advanced features like function calling
- **Better Debugging**: Unique IDs and detailed response information

## Conclusion

The OpenAI Chat Completions standard provides a robust, flexible foundation for AI applications. By understanding and implementing this protocol correctly, you can build applications that work seamlessly across different AI providers while following industry best practices.

This standardization enables the hot-swappable backend architecture that makes `task-breaker` provider-agnostic and future-ready.