package embed

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBedrockClientInvokeHappyPath(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model/foo:0/invoke" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("bad json: %v", err)
		}
		_, _ = io.WriteString(w, `{"echo":"ok"}`)
	}))
	t.Cleanup(ts.Close)

	c, err := NewClient(ts.URL, withClock(time.Now, func(time.Duration) {}))
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Echo string `json:"echo"`
	}
	if err := c.Invoke(context.Background(), "foo:0", map[string]any{"inputText": "hi"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Echo != "ok" {
		t.Errorf("got %q", out.Echo)
	}
}

func TestBedrockClientRetriesOn503(t *testing.T) {
	var calls atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":"slow down"}`)
			return
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(ts.Close)

	var slept int
	c, err := NewClient(ts.URL, withClock(time.Now, func(time.Duration) { slept++ }))
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.Invoke(context.Background(), "foo:0", map[string]any{}, &out); err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if !out.OK {
		t.Error("expected ok=true")
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
	if slept != 2 {
		t.Errorf("expected 2 backoff sleeps, got %d", slept)
	}
}

func TestBedrockClient4xxIsNotRetried(t *testing.T) {
	var calls atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"bad"}`)
	}))
	t.Cleanup(ts.Close)

	c, err := NewClient(ts.URL, withClock(time.Now, func(time.Duration) {}))
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	err = c.Invoke(context.Background(), "foo:0", map[string]any{}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("unexpected error: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected exactly 1 call, got %d", calls.Load())
	}
}

func TestNewClientRejectsEmptyURL(t *testing.T) {
	if _, err := NewClient("  "); err == nil {
		t.Fatal("expected error")
	}
}
