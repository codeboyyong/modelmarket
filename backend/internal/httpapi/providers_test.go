package httpapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAICompatibleProviderAdapter(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("TEST_OPENAI_KEY", "secret")
	app.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://provider.test/v1/chat/completions" {
			t.Fatalf("url = %s", req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing provider authorization")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"upstream-1","choices":[{"message":{"content":"hello"}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`)), Request: req}, nil
	})}
	route := selectedModelRoute{ChannelType: "openai_compatible", BaseURL: "https://provider.test/v1", CredentialRef: "TEST_OPENAI_KEY", UpstreamModelID: "model-1"}
	result, err := app.runChatUpstream(context.Background(), route, []chatMessage{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "hello" || result.PromptTokens != 3 || result.CompletionTokens != 2 || result.ProviderRequestID != "upstream-1" {
		t.Fatalf("result = %+v", result)
	}
}
