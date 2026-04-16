package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ModelProvider interface {
	Name() string
	Chat(endpoint string, apiKey string, payload map[string]interface{}) (map[string]interface{}, error)
}

type OpenAIProvider struct{}

func (OpenAIProvider) Name() string {
	return "openai"
}

func (OpenAIProvider) Chat(endpoint string, apiKey string, payload map[string]interface{}) (map[string]interface{}, error) {
	return postJSON(endpoint, apiKey, payload, "Authorization", "Bearer "+apiKey)
}

type AnthropicProvider struct{}

func (AnthropicProvider) Name() string {
	return "anthropic"
}

func (AnthropicProvider) Chat(endpoint string, apiKey string, payload map[string]interface{}) (map[string]interface{}, error) {
	return postJSON(endpoint, apiKey, payload, "x-api-key", apiKey)
}

func postJSON(endpoint string, apiKey string, payload map[string]interface{}, authHeader string, authValue string) (map[string]interface{}, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is empty")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set(authHeader, authValue)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return out, fmt.Errorf("provider request failed: %d", resp.StatusCode)
	}
	return out, nil
}
