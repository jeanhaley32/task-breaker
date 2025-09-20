package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents the application configuration
type Config struct {
	OpenAI         OpenAIConfig     `json:"openai"`
	Claude         ClaudeConfig     `json:"claude"`
	Default        DefaultConfig    `json:"default"`
	ChatController ControllerConfig `json:"chat_controller"`
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey     string        `json:"api_key"`
	BaseURL    string        `json:"base_url"`
	Model      string        `json:"model"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
}

// ClaudeConfig holds Claude-specific configuration
type ClaudeConfig struct {
	APIKey     string        `json:"api_key"`
	BaseURL    string        `json:"base_url"`
	Model      string        `json:"model"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
}

// DefaultConfig holds default settings
type DefaultConfig struct {
	Backend     string  `json:"backend"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// ControllerConfig holds chat controller configuration
type ControllerConfig struct {
	DefaultModel string  `json:"default_model"`
	MaxTokens    int     `json:"max_tokens"`
	Temperature  float64 `json:"temperature"`
}

// Manager handles configuration loading and saving
type Manager struct {
	configPath string
	config     *Config
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	if configPath == "" {
		// Default to user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			configPath = ".task-breaker-config.json"
		} else {
			configPath = filepath.Join(homeDir, ".task-breaker-config.json")
		}
	}

	return &Manager{
		configPath: configPath,
		config:     getDefaultConfig(),
	}
}

// Load reads the configuration from file
func (m *Manager) Load() error {
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults and create it
		return m.Save()
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Load from environment variables if not set in config
	m.loadFromEnv()

	return nil
}

// Save writes the configuration to file
func (m *Manager) Save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}

// SetOpenAIAPIKey sets the OpenAI API key
func (m *Manager) SetOpenAIAPIKey(apiKey string) {
	m.config.OpenAI.APIKey = apiKey
}

// SetClaudeAPIKey sets the Claude API key
func (m *Manager) SetClaudeAPIKey(apiKey string) {
	m.config.Claude.APIKey = apiKey
}

// SetDefaultBackend sets the default backend
func (m *Manager) SetDefaultBackend(backend string) {
	m.config.Default.Backend = backend
}

// loadFromEnv loads configuration from environment variables
func (m *Manager) loadFromEnv() {
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		m.config.OpenAI.APIKey = apiKey
	}

	if apiKey := os.Getenv("CLAUDE_API_KEY"); apiKey != "" {
		m.config.Claude.APIKey = apiKey
	}

	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		m.config.OpenAI.BaseURL = baseURL
	}

	if baseURL := os.Getenv("CLAUDE_BASE_URL"); baseURL != "" {
		m.config.Claude.BaseURL = baseURL
	}

	if backend := os.Getenv("DEFAULT_BACKEND"); backend != "" {
		m.config.Default.Backend = backend
	}

	if model := os.Getenv("DEFAULT_MODEL"); model != "" {
		m.config.Default.Model = model
	}
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		OpenAI: OpenAIConfig{
			BaseURL:    "https://api.openai.com/v1",
			Model:      "gpt-4",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
		},
		Claude: ClaudeConfig{
			BaseURL:    "https://api.anthropic.com/v1",
			Model:      "claude-3-sonnet-20240229",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
		},
		Default: DefaultConfig{
			Backend:     "mock",
			Model:       "gpt-4",
			MaxTokens:   500,
			Temperature: 0.7,
		},
		ChatController: ControllerConfig{
			DefaultModel: "gpt-4",
			MaxTokens:    500,
			Temperature:  0.7,
		},
	}
}

// ValidateConfig checks if the configuration is valid
func (m *Manager) ValidateConfig() error {
	config := m.config

	// Check if at least one backend is configured
	hasValidBackend := false

	if config.OpenAI.APIKey != "" {
		hasValidBackend = true
	}

	if config.Claude.APIKey != "" {
		hasValidBackend = true
	}

	// Mock backend is always available
	if config.Default.Backend == "mock" {
		hasValidBackend = true
	}

	if !hasValidBackend {
		return fmt.Errorf("no valid backend configured - set OPENAI_API_KEY or CLAUDE_API_KEY environment variable")
	}

	// Validate temperature range
	if config.Default.Temperature < 0.0 || config.Default.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0")
	}

	// Validate max tokens
	if config.Default.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than 0")
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// InitializeConfig creates a configuration file with prompts for API keys
func (m *Manager) InitializeConfig() error {
	fmt.Println("Initializing Task Breaker configuration...")
	fmt.Println()

	// Check for environment variables first
	openAIKey := os.Getenv("OPENAI_API_KEY")
	claudeKey := os.Getenv("CLAUDE_API_KEY")

	if openAIKey != "" {
		fmt.Println("✓ Found OPENAI_API_KEY in environment")
		m.config.OpenAI.APIKey = openAIKey
	}

	if claudeKey != "" {
		fmt.Println("✓ Found CLAUDE_API_KEY in environment")
		m.config.Claude.APIKey = claudeKey
	}

	if openAIKey == "" && claudeKey == "" {
		fmt.Println("No API keys found in environment variables.")
		fmt.Println("You can set them later using environment variables:")
		fmt.Println("  export OPENAI_API_KEY=your_key_here")
		fmt.Println("  export CLAUDE_API_KEY=your_key_here")
		fmt.Println()
		fmt.Println("Or you can edit the config file at:", m.configPath)
	}

	// Set reasonable defaults
	if openAIKey != "" {
		m.config.Default.Backend = "openai"
	} else if claudeKey != "" {
		m.config.Default.Backend = "claude"
	} else {
		m.config.Default.Backend = "mock"
		fmt.Println("Using mock backend for testing (no API costs)")
	}

	if err := m.Save(); err != nil {
		return fmt.Errorf("failed to save initial configuration: %w", err)
	}

	fmt.Printf("✓ Configuration saved to: %s\n", m.configPath)
	return nil
}
