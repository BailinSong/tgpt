package main

import (
	"encoding/json"
	"os"
)

// Message represents a single message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Messages represents a conversation consisting of multiple messages.
type Messages struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float32   `json:"temperature"`
}

// NewMessages creates a new Messages object.
func NewMessages() *Messages {
	return &Messages{
		Messages:    make([]Message, 0),
		Model:       "gpt-3.5-turbo", // Default model value
		Stream:      true,            // Default stream value
		Temperature: .7,              // Default temperature value
	}
}

// AddMessage adds a new message to the conversation.
func (m *Messages) AddMessage(role, content string) {
	message := Message{
		Role:    role,
		Content: content,
	}
	m.Messages = append(m.Messages, message)
}

func (m *Messages) AddSystemMessage(content string) {
	m.AddMessage("system", content)
}

func (m *Messages) AddUserMessage(content string) {
	m.AddMessage("user", content)
}

func (m *Messages) AddAssistantMessage(content string) {
	m.AddMessage("assistant", content)
}

// Serialize serializes the Messages object to a JSON string.
func (m *Messages) Serialize() (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (m *Messages) save(path string) error {
	// Serialize the Messages object to a JSON string
	data, err := m.Serialize()
	if err != nil {
		return err
	}
	// Write the JSON data to the specified file
	err = os.WriteFile(path, []byte(data), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (m *Messages) load(path string) error {
	// Read the JSON data from the specified file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Deserialize the JSON data into the Messages object
	err = m.Deserialize(string(data))
	if err != nil {
		return err
	}
	return nil
}

// Deserialize deserializes a JSON string into a Messages object.
func (m *Messages) Deserialize(data string) error {
	err := json.Unmarshal([]byte(data), m)
	if err != nil {
		return err
	}
	return nil
}
