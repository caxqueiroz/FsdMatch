// Package embed talks to Amazon Bedrock through the user's KrakenD route.
// KrakenD handles SigV4, model routing and SSE pass-through; this package
// only speaks JSON over HTTPS.
//
// Hard constraint (CLAUDE.md): the AWS endpoint is never hardcoded.
// Construct a client with the base URL from configuration (env
// BEDROCK_BASE_URL) and let the gateway route it.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cax/fsdtrace/internal/retry"
)

// MaxRetries is the per-request cap on retry attempts for 429/5xx responses.
const MaxRetries = retry.MaxRetries

// HTTPDoer is the subset of *http.Client we actually use, so tests can
// substitute an httptest.Server-backed client transparently.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// BedrockClient invokes models behind a KrakenD route. It is safe for
// concurrent use.
type BedrockClient struct {
	baseURL string
	http    HTTPDoer
	logger  *slog.Logger
	// nowFn and sleepFn exist for deterministic testing of backoff.
	nowFn   func() time.Time
	sleepFn func(time.Duration)
}

// ClientOption configures a BedrockClient.
type ClientOption func(*BedrockClient)

// WithHTTPClient overrides the default *http.Client.
func WithHTTPClient(h HTTPDoer) ClientOption { return func(c *BedrockClient) { c.http = h } }

// WithLogger overrides the default slog handler.
func WithLogger(l *slog.Logger) ClientOption { return func(c *BedrockClient) { c.logger = l } }

// withClock is used by tests to make backoff deterministic.
func withClock(now func() time.Time, sleep func(time.Duration)) ClientOption {
	return func(c *BedrockClient) {
		c.nowFn = now
		c.sleepFn = sleep
	}
}

// NewClient constructs a BedrockClient pointed at baseURL (e.g. the
// KrakenD route). baseURL must be a fully qualified URL.
func NewClient(baseURL string, opts ...ClientOption) (*BedrockClient, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("bedrock: base URL is empty")
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("bedrock: parsing base URL %q: %w", baseURL, err)
	}
	c := &BedrockClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 60 * time.Second},
		logger:  slog.Default(),
		nowFn:   time.Now,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Invoke POSTs body to /model/{modelID}/invoke and unmarshals the JSON
// response into out. Retries on 429/5xx with capped exponential backoff
// + jitter, honoring Retry-After when present.
func (c *BedrockClient) Invoke(ctx context.Context, modelID string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("bedrock: marshal: %w", err)
	}
	endpoint := fmt.Sprintf("%s/model/%s/invoke", c.baseURL, url.PathEscape(modelID))

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		respBody, status, header, err := c.doOnce(ctx, endpoint, payload)
		switch {
		case err != nil:
			if ctx.Err() != nil {
				return fmt.Errorf("bedrock: %s: %w", modelID, ctx.Err())
			}
			if attempt == MaxRetries {
				return fmt.Errorf("bedrock: %s: %w", modelID, err)
			}
			if err := retry.Sleep(ctx, c.sleepFn, retry.Backoff(attempt)); err != nil {
				return fmt.Errorf("bedrock: %s: %w", modelID, err)
			}
			continue
		case retry.RetryableStatus(status):
			if attempt == MaxRetries {
				return fmt.Errorf("bedrock: %s: HTTP %d after %d retries", modelID, status, attempt)
			}
			c.logger.WarnContext(ctx, "bedrock retryable status",
				"model", modelID, "status", status, "attempt", attempt)
			delay := retry.Delay(attempt, header.Get("Retry-After"), c.nowFn)
			if err := retry.Sleep(ctx, c.sleepFn, delay); err != nil {
				return fmt.Errorf("bedrock: %s: %w", modelID, err)
			}
			continue
		case status >= 400:
			return fmt.Errorf("bedrock: %s: HTTP %d: %s", modelID, status, truncate(respBody, 512))
		}
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("bedrock: %s: decode: %w; body=%s",
				modelID, err, truncate(respBody, 256))
		}
		return nil
	}
	return fmt.Errorf("bedrock: %s: retries exhausted", modelID)
}

func (c *BedrockClient) doOnce(ctx context.Context, endpoint string, payload []byte) ([]byte, int, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("reading body: %w", err)
	}
	return body, resp.StatusCode, resp.Header, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "...(truncated)"
}
