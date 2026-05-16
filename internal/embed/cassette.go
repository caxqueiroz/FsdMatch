package embed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

// Cassette plays back recorded Bedrock responses keyed by the SHA-256
// of the request body. Used by the offline smoke test so re-runs are
// hermetic and free.
//
// File format (JSON):
//
//	{
//	  "entries": [
//	    {
//	      "model_id": "amazon.titan-embed-text-v2:0",
//	      "request_sha256": "abc...",
//	      "status": 200,
//	      "response": { ... raw Bedrock response ... }
//	    }
//	  ]
//	}
type Cassette struct {
	mu      sync.Mutex
	entries map[string]cassetteEntry
}

type cassetteFile struct {
	Entries []cassetteEntry `json:"entries"`
}

type cassetteEntry struct {
	ModelID       string          `json:"model_id"`
	RequestSHA256 string          `json:"request_sha256"`
	Status        int             `json:"status"`
	Response      json.RawMessage `json:"response"`
}

// LoadCassette reads a cassette file from disk.
func LoadCassette(path string) (*Cassette, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- cassette path supplied by smoke harness
	if err != nil {
		return nil, fmt.Errorf("load cassette %s: %w", path, err)
	}
	var cf cassetteFile
	if err := json.Unmarshal(raw, &cf); err != nil {
		return nil, fmt.Errorf("parse cassette %s: %w", path, err)
	}
	c := &Cassette{entries: make(map[string]cassetteEntry, len(cf.Entries))}
	for _, e := range cf.Entries {
		c.entries[e.RequestSHA256] = e
	}
	return c, nil
}

// NewEmptyCassette returns a cassette with no entries. Recordings can be
// appended via Record.
func NewEmptyCassette() *Cassette {
	return &Cassette{entries: map[string]cassetteEntry{}}
}

// Record adds a (request, response) pair under the SHA-256 of body.
func (c *Cassette) Record(modelID string, requestBody, responseBody []byte, status int) {
	sum := sha256.Sum256(requestBody)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[hex.EncodeToString(sum[:])] = cassetteEntry{
		ModelID:       modelID,
		RequestSHA256: hex.EncodeToString(sum[:]),
		Status:        status,
		Response:      append(json.RawMessage(nil), responseBody...),
	}
}

// Save writes the cassette to path.
func (c *Cassette) Save(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]cassetteEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, c.entries[k])
	}
	raw, err := json.MarshalIndent(cassetteFile{Entries: entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cassette: %w", err)
	}
	return os.WriteFile(path, raw, 0o644) // #nosec G306 -- cassettes are not secrets
}

// RoundTrip implements http.RoundTripper. The model ID is extracted from
// the URL path /model/{id}/invoke; the request body must JSON-match a
// recorded entry. Missing entries return HTTP 599 with a diagnostic
// body so tests fail loudly rather than silently.
func (c *Cassette) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("cassette: read body: %w", err)
		}
		_ = req.Body.Close()
		body = raw
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	sum := sha256.Sum256(body)
	key := hex.EncodeToString(sum[:])

	c.mu.Lock()
	entry, ok := c.entries[key]
	c.mu.Unlock()

	if !ok {
		return makeResp(599, []byte(fmt.Sprintf(
			`{"error":"cassette miss","request_sha256":%q,"request_body":%s}`,
			key, body))), nil
	}
	return makeResp(entry.Status, entry.Response), nil
}

func makeResp(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d cassette", status),
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

// HTTPClient returns an *http.Client whose Transport is this cassette.
func (c *Cassette) HTTPClient() *http.Client {
	return &http.Client{Transport: c}
}

// RecordingTransport wraps another http.RoundTripper, forwarding every
// request and persisting (request, response) pairs into the embedded
// Cassette. Use it to generate cassettes from a live or fake backend.
type RecordingTransport struct {
	Inner    http.RoundTripper
	Cassette *Cassette
}

// RoundTrip implements http.RoundTripper.
func (r *RecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var reqBody []byte
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("record: read req body: %w", err)
		}
		_ = req.Body.Close()
		reqBody = raw
		req.Body = io.NopCloser(bytes.NewReader(raw))
	}

	inner := r.Inner
	if inner == nil {
		inner = http.DefaultTransport
	}
	resp, err := inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("record: read resp body: %w", err)
	}
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	modelID := extractModelID(req.URL.Path)
	r.Cassette.Record(modelID, reqBody, respBody, resp.StatusCode)
	return resp, nil
}

// extractModelID parses "/model/{id}/invoke". Returns "" when the path
// does not match; the recorder still stores the entry by request hash
// so cassette playback is unaffected.
func extractModelID(p string) string {
	const prefix = "/model/"
	if !strings.HasPrefix(p, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(p, prefix)
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i]
	}
	return rest
}
