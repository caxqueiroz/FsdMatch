package fsd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strings"
	"time"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
)

// Feature mirrors the features table columns, with JSON-shaped fields
// kept as []string for ergonomic Go usage.
type Feature struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Acceptance    []string `json:"acceptance"`
	Actor         *string  `json:"actor,omitempty"`
	Inputs        []string `json:"inputs,omitempty"`
	Outputs       []string `json:"outputs,omitempty"`
	SideEffects   []string `json:"side_effects,omitempty"`
	NonFunctional []string `json:"non_functional,omitempty"`
	// FSDSection and FSDAnchor are filled by the atomizer from the
	// chunk context, not the LLM.
	FSDSection string `json:"-"`
	FSDAnchor  string `json:"-"`
}

// EmbeddingText is the canonical text fed to Titan for this feature.
// SPEC §7.1: title + description + acceptance criteria.
func (f Feature) EmbeddingText() string {
	var b strings.Builder
	b.WriteString(f.Title)
	b.WriteString("\n")
	b.WriteString(f.Description)
	for _, c := range f.Acceptance {
		b.WriteString("\n- ")
		b.WriteString(c)
	}
	return b.String()
}

// FeatureRowID is the deterministic int64 used as the rowid for the
// feature_vec virtual table. SPEC §6 invariants permit either a stable
// hash or a mapping table; we use FNV-64 of the FR id. Collision
// probability for thousands of features is negligible.
func FeatureRowID(id string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	r := int64(h.Sum64() & 0x7fffffffffffffff)
	if r == 0 {
		r = 1
	}
	return r
}

// Atomizer turns FSD chunks into Feature rows and embeddings, writing
// both atomically through the single-writer goroutine.
type Atomizer struct {
	bedrock  *embed.BedrockClient
	embedder embed.Embedder
	cache    *embed.Cache
	d        *db.DB
	model    string
	maxTok   int
	logger   *slog.Logger
}

// AtomizerOption configures an Atomizer.
type AtomizerOption func(*Atomizer)

// WithModel overrides the default Claude model.
func WithModel(m string) AtomizerOption { return func(a *Atomizer) { a.model = m } }

// WithMaxTokens overrides the per-call max output tokens (default 2048).
func WithMaxTokens(n int) AtomizerOption { return func(a *Atomizer) { a.maxTok = n } }

// WithLogger overrides the default logger.
func WithLogger(l *slog.Logger) AtomizerOption { return func(a *Atomizer) { a.logger = l } }

// NewAtomizer constructs an Atomizer.
func NewAtomizer(d *db.DB, bedrock *embed.BedrockClient, embedder embed.Embedder, opts ...AtomizerOption) *Atomizer {
	a := &Atomizer{
		bedrock:  bedrock,
		embedder: embedder,
		cache:    embed.NewCache(d),
		d:        d,
		model:    DefaultAtomizerModel,
		maxTok:   2048,
		logger:   slog.Default(),
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// IngestResult is the per-run summary returned by Ingest.
type IngestResult struct {
	RunID    string
	Features []Feature
}

// Ingest atomizes all chunks and persists features + embeddings. The
// returned slice is in input order. Re-runs are idempotent: an UPSERT
// on features.id and INSERT-OR-REPLACE on feature_vec.rowid keep counts
// stable.
func (a *Atomizer) Ingest(ctx context.Context, chunks []Chunk, runID string) (*IngestResult, error) {
	if runID == "" {
		runID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	out := &IngestResult{RunID: runID, Features: make([]Feature, 0, len(chunks))}

	for _, ch := range chunks {
		f, err := a.atomizeChunk(ctx, ch)
		if err != nil {
			return nil, fmt.Errorf("atomize %s: %w", ch.Anchor, err)
		}
		f.FSDAnchor = ch.Anchor
		f.FSDSection = ch.Section

		v, err := embed.Cached(ctx, a.embedder, a.cache, f.EmbeddingText())
		if err != nil {
			return nil, fmt.Errorf("embed %s: %w", f.ID, err)
		}

		if err := a.writeFeature(ctx, f, v, runID); err != nil {
			return nil, fmt.Errorf("write %s: %w", f.ID, err)
		}
		out.Features = append(out.Features, f)
		a.logger.InfoContext(ctx, "atomized FR",
			"id", f.ID, "title", f.Title, "section", f.FSDSection,
			"run_id", runID, "prompt_version", AtomizerPromptVersion)
	}
	return out, nil
}

// bedrockMessage is the Anthropic-on-Bedrock messages payload.
type bedrockMessage struct {
	AnthropicVersion string                `json:"anthropic_version"`
	MaxTokens        int                   `json:"max_tokens"`
	System           string                `json:"system,omitempty"`
	Messages         []bedrockMessageEntry `json:"messages"`
}

type bedrockMessageEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bedrockMessageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

func (a *Atomizer) atomizeChunk(ctx context.Context, ch Chunk) (Feature, error) {
	req := bedrockMessage{
		AnthropicVersion: BedrockAnthropicVersion,
		MaxTokens:        a.maxTok,
		System:           AtomizerSystem,
		Messages: []bedrockMessageEntry{
			{Role: "user", Content: BuildAtomizerUserMessage(ch.Anchor, ch.Section, ch.Text)},
		},
	}
	var resp bedrockMessageResponse
	if err := a.bedrock.Invoke(ctx, a.model, req, &resp); err != nil {
		return Feature{}, err
	}
	text := joinResponseText(resp)
	if text == "" {
		return Feature{}, errors.New("atomizer: empty response")
	}
	var f Feature
	if err := json.Unmarshal([]byte(stripFence(text)), &f); err != nil {
		return Feature{}, fmt.Errorf("decoding atomizer response: %w; body=%q",
			err, truncate(text, 256))
	}
	if f.ID == "" {
		// Fall back to the chunk anchor when the model omits the id.
		f.ID = ch.Anchor
	}
	return f, nil
}

func joinResponseText(r bedrockMessageResponse) string {
	var b strings.Builder
	for _, c := range r.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

// stripFence removes a leading ```json ... ``` fence if present. The
// prompt asks for raw JSON but defensive handling is cheap.
func stripFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

func (a *Atomizer) writeFeature(ctx context.Context, f Feature, v []float32, runID string) error {
	acceptanceJSON, err := json.Marshal(orEmpty(f.Acceptance))
	if err != nil {
		return err
	}
	inputsJSON, err := json.Marshal(orEmpty(f.Inputs))
	if err != nil {
		return err
	}
	outputsJSON, err := json.Marshal(orEmpty(f.Outputs))
	if err != nil {
		return err
	}
	sideJSON, err := json.Marshal(orEmpty(f.SideEffects))
	if err != nil {
		return err
	}
	nonFuncJSON, err := json.Marshal(orEmpty(f.NonFunctional))
	if err != nil {
		return err
	}
	var actor *string
	if f.Actor != nil && *f.Actor != "" {
		actor = f.Actor
	}
	rowID := FeatureRowID(f.ID)
	blob := db.PackFloat32(v)

	return a.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO features
			  (id, title, description, acceptance, actor, inputs, outputs,
			   side_effects, non_functional, fsd_section, fsd_anchor, run_id, created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(id) DO UPDATE SET
			  title          = excluded.title,
			  description    = excluded.description,
			  acceptance     = excluded.acceptance,
			  actor          = excluded.actor,
			  inputs         = excluded.inputs,
			  outputs        = excluded.outputs,
			  side_effects   = excluded.side_effects,
			  non_functional = excluded.non_functional,
			  fsd_section    = excluded.fsd_section,
			  fsd_anchor     = excluded.fsd_anchor,
			  run_id         = excluded.run_id,
			  created_at     = excluded.created_at
		`,
			f.ID, f.Title, f.Description, acceptanceJSON, actor,
			inputsJSON, outputsJSON, sideJSON, nonFuncJSON,
			f.FSDSection, f.FSDAnchor, runID, time.Now().Unix(),
		); err != nil {
			return err
		}
		// vec0 has no UPSERT; delete-then-insert is the documented pattern.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM feature_vec WHERE rowid = ?`, rowID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO feature_vec(rowid, embedding) VALUES (?, ?)`,
			rowID, blob); err != nil {
			return err
		}
		return nil
	})
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
