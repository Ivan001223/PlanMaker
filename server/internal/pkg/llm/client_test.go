package llm

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-key", "https://api.example.com")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got '%s'", client.apiKey)
	}
	if client.baseURL != "https://api.example.com" {
		t.Errorf("expected baseURL 'https://api.example.com', got '%s'", client.baseURL)
	}
}

func TestMockResponse(t *testing.T) {
	client := NewClient("mock", "https://api.example.com")
	req := ChatRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := client.mockResponse(req)
	if err != nil {
		t.Fatalf("mockResponse returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("mockResponse returned nil")
	}
	if len(resp.Choices) == 0 {
		t.Fatal("mockResponse returned empty choices")
	}
	if resp.Choices[0].Message.Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", resp.Choices[0].Message.Role)
	}
	if resp.Choices[0].Message.Content == "" {
		t.Error("mockResponse returned empty content")
	}
}

func TestChatCompletionWithMock(t *testing.T) {
	client := NewClient("mock", "https://api.example.com")
	req := ChatRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Test message"},
		},
	}

	resp, err := client.ChatCompletion(req)
	if err != nil {
		t.Fatalf("ChatCompletion returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("ChatCompletion returned nil")
	}
	if resp.Model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", resp.Model)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("ChatCompletion returned empty choices")
	}
}
