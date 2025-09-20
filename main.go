package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

type Agent struct {
	name    string
	context string
}

func NewAgent(name string) *Agent {
	return &Agent{
		name: name,
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

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <context-file>")
	}

	contextFile := os.Args[1]

	agent := NewAgent("HackathonAgent")

	if err := agent.LoadContext(contextFile); err != nil {
		log.Fatalf("Error loading context: %v", err)
	}

	agent.PrintContext()
}