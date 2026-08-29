package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type providerAdapter interface {
	Complete(context.Context, selectedModelRoute, []chatMessage) (upstreamChatResult, error)
}

type mockProviderAdapter struct{}

func (mockProviderAdapter) Complete(_ context.Context, route selectedModelRoute, messages []chatMessage) (upstreamChatResult, error) {
	content := "Mock response from " + route.UpstreamModelID + " via " + route.ChannelName
	if len(messages) > 0 {
		content = "Mock response to: " + messages[len(messages)-1].Text()
	}
	return upstreamChatResult{Content: content, PromptTokens: len(messages), CompletionTokens: len(content), LatencyMS: route.ResponseTimeMS, ProviderRequestID: "mock-" + randomHex(8)}, nil
}

type geminiProviderAdapter struct{ app *App }

func (p geminiProviderAdapter) Complete(ctx context.Context, route selectedModelRoute, messages []chatMessage) (upstreamChatResult, error) {
	return p.app.callGeminiGenerateContent(ctx, route, messages)
}

type openAICompatibleProviderAdapter struct{ app *App }

func (p openAICompatibleProviderAdapter) Complete(ctx context.Context, route selectedModelRoute, messages []chatMessage) (upstreamChatResult, error) {
	credentialRef := strings.TrimSpace(route.CredentialRef)
	if credentialRef == "" {
		credentialRef = "OPENAI_API_KEY"
	}
	apiKey, _, err := p.app.providerCredential(credentialRef)
	if err != nil {
		return upstreamChatResult{}, err
	}
	type openAIMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	payload := struct {
		Model    string          `json:"model"`
		Messages []openAIMessage `json:"messages"`
	}{Model: route.UpstreamModelID}
	for _, message := range messages {
		text := strings.TrimSpace(message.Text())
		if text != "" {
			payload.Messages = append(payload.Messages, openAIMessage{Role: message.Role, Content: text})
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return upstreamChatResult{}, err
	}
	baseURL := strings.TrimRight(route.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return upstreamChatResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := p.app.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	started := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return upstreamChatResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return upstreamChatResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return upstreamChatResult{}, fmt.Errorf("provider_http_%d: %s", resp.StatusCode, truncateString(string(body), 500))
	}
	var decoded struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return upstreamChatResult{}, err
	}
	if len(decoded.Choices) == 0 {
		return upstreamChatResult{}, errors.New("provider response has no choices")
	}
	return upstreamChatResult{Content: decoded.Choices[0].Message.Content, PromptTokens: decoded.Usage.PromptTokens, CompletionTokens: decoded.Usage.CompletionTokens, LatencyMS: time.Since(started).Milliseconds(), ProviderRequestID: decoded.ID}, nil
}

func (a *App) providerAdapter(route selectedModelRoute) providerAdapter {
	if route.ChannelType == "google_gemini" {
		return geminiProviderAdapter{app: a}
	}
	if strings.HasPrefix(route.BaseURL, "mock://") || route.ProviderSlug == "mock-provider" {
		return mockProviderAdapter{}
	}
	return openAICompatibleProviderAdapter{app: a}
}
