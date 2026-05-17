package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIClientGenerateUsesResponsesAPI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["model"]; got != DefaultOpenAIModel {
			t.Fatalf("model = %v", got)
		}
		if got := body["instructions"]; got != "system prompt" {
			t.Fatalf("instructions = %v", got)
		}
		if got := body["input"]; got != "user prompt" {
			t.Fatalf("input = %v", got)
		}
		if got := body["max_output_tokens"]; got != float64(777) {
			t.Fatalf("max_output_tokens = %v", got)
		}
		_, _ = w.Write([]byte(`{"status":"completed","output_text":"{\"ok\":true}"}`))
	}))
	t.Cleanup(ts.Close)

	c, err := NewOpenAIClient("test-key",
		WithOpenAIBaseURL(ts.URL),
		WithOpenAIHTTPClient(ts.Client()))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Generate(context.Background(), GenerateRequest{
		Model:     DefaultOpenAIModel,
		System:    "system prompt",
		User:      "user prompt",
		MaxTokens: 777,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"ok":true}` {
		t.Fatalf("Generate() = %q", got)
	}
}

func TestOpenAIClientGenerateReadsNestedOutputText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"status":"completed",
			"output":[{"type":"message","content":[{"type":"output_text","text":"nested text"}]}]
		}`))
	}))
	t.Cleanup(ts.Close)

	c, err := NewOpenAIClient("test-key",
		WithOpenAIBaseURL(ts.URL),
		WithOpenAIHTTPClient(ts.Client()))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Generate(context.Background(), GenerateRequest{Model: "gpt-test", User: "prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "nested text" {
		t.Fatalf("Generate() = %q", got)
	}
}

func TestOpenAIClientGenerateRejectsIncompletePartialText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"status":"incomplete",
			"incomplete_details":{"reason":"max_output_tokens"},
			"output_text":"[{\"artifact_id\":1"
		}`))
	}))
	t.Cleanup(ts.Close)

	c, err := NewOpenAIClient("test-key",
		WithOpenAIBaseURL(ts.URL),
		WithOpenAIHTTPClient(ts.Client()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Generate(context.Background(), GenerateRequest{Model: "gpt-test", User: "prompt"})
	if err == nil {
		t.Fatal("expected incomplete response error")
	}
	if !strings.Contains(err.Error(), "max_output_tokens") {
		t.Fatalf("error = %v", err)
	}
	if !IsIncomplete(err) {
		t.Fatalf("expected incomplete error, got %T", err)
	}
}
