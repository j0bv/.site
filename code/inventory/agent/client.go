package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// ChatRequest is the Ollama /api/chat request body.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Message is a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse is the Ollama /api/chat response.
type ChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// Chat sends a single user message to the SLM and returns the assistant content.
// Uses SLM_URL (default http://localhost:11434), SLM_MODEL, and a 60s timeout.
func Chat(ctx context.Context, userContent string) (string, error) {
	baseURL := os.Getenv("SLM_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := os.Getenv("SLM_MODEL")
	if model == "" {
		model = "qwen2.5:3b"
	}

	timeoutSec := 60
	if v := os.Getenv("SLM_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	}

	reqBody := ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "user", Content: userContent},
		},
		Stream: false,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request SLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SLM returned status %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	return chatResp.Message.Content, nil
}
