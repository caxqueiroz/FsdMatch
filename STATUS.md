# fsdtrace — Status

## Current phase
**Phase 7 — Polish: complete, awaiting review.**

All seven phases from `SPEC.md` §11 have shipped under the CGO-enabled
build contract (modulo optional Grafana/ClickHouse export and
cross-platform release automation, both deferred per SPEC).

## Phase 7 acceptance criteria
- [x] SSE transport for MCP (`fsdtrace mcp serve --transport sse --addr ...`)
- [x] `--rejudge-drifts` second pass with `--rejudge-model` (default `anthropic.claude-opus-4-v2:0`)
- [x] Native CGO build tested in CI (`.github/workflows/ci.yml`: lint + race-tests + build)
- [x] SPEC reconciliation:
  - §2 stack table: `CGO_ENABLED=1` is required (tree-sitter requires it); SCIP module path corrected to `github.com/scip-code/scip/bindings/go/scip`; the false `nocgo` cross-compile path was removed
  - §6 schema: `matches.notes` now documents the `tested=… test_count=…` decoration suffix
  - §7.4: explicit pointer to where the test cross-check decoration lives, and how reporters strip it
  - §11 Phase 7: SSE/rejudge marked done; Grafana noted as deferred
- [x] Optional Grafana dashboard: deferred (SPEC §7's "optional" tag) with rationale recorded in SPEC §11

## What landed in Phase 7

- `internal/mcp/server.go` — added `ServeSSE(ctx, srv, addr)` wrapping `server.NewSSEServer(srv).Start(addr)`. Honours ctx cancellation via a 5s graceful Shutdown.
- `internal/cli/mcp.go` — `--transport stdio|sse` plus `--addr 127.0.0.1:8765` default. The previous "Phase 7 stub" error path is gone.
- `internal/match/pipeline.go`
  - `RejudgeDrifts(ctx, runID, opusModel) (*RejudgeSummary, error)`. Loads every `verdict='drifts'` row for the run, hydrates the artifact context, replays the prompt against a stronger judge, and updates the row in place via the existing UPSERT path.
  - `RejudgeSummary { RunID, Model, Total, PromotedToImplements, StillDrifts, DowngradedToUnrelated }` — surfaced to the CLI's stdout summary line.
- `internal/cli/match.go` — `--rejudge-drifts` (bool) and `--rejudge-model` (string, default Opus). When set, the matcher runs the second pass after the first and prints both summary lines.
- `internal/match/pipeline_test.go::TestRejudgeDrifts_PromotesAndPersists` — seeds a drifts row, points a fake Opus at httptest, asserts: row updated in place, model column flipped to Opus, confidence raised, notes carry the rejudge marker, RejudgeSummary counts correct.
- `.github/workflows/ci.yml` — single native-CGO CI job: golangci-lint v2.12.2, `go test -race`, and `CGO_ENABLED=1 go build`.
- `scripts/cross-compile.sh` — now fails fast with a clear explanation that cross-compilation requires explicit target C toolchains; the previous `CGO_ENABLED=0 -tags nocgo` path was removed because the repository has no no-indexer build split.
- `SPEC.md` updates as listed above.

## Decisions log (full project history)

- 2026-05-15 (Phase 1) — `.golangci.yml` v2 schema; `make test` keeps default CGO so `-race` works.
- 2026-05-15 (Phase 1) — `modernc.org/sqlite/vec` auto-registers vec0 via `init`; pinned at v1.50.1.
- 2026-05-16 (Phase 2) — FNV-64 → `feature_vec.rowid`; cassette keyed by SHA-256 of request body, recorded through real code paths via `RecordingTransport`. Renamed `embed.EmbedCached` → `embed.Cached`.
- 2026-05-16 (Phase 3) — User authorised CGO for tree-sitter; `make build`/`install` use `CGO_ENABLED=1`.
- 2026-05-16 (Phase 3) — SCIP module renamed to `github.com/scip-code/scip` at v0.7.x.
- 2026-05-16 (Phase 4) — `tested`/`test_count` persisted as a suffix on `matches.notes`.
- 2026-05-16 (Phase 4) — Anchor-matched artifacts seed the candidate list ahead of vec0 KNN.
- 2026-05-16 (Phase 5) — Orphans scoped to public-surface kinds; latest-run selection orders by `MAX(rowid)`.
- 2026-05-16 (Phase 6) — Lazy DB + Bedrock open via `sync.Once`; `NewServer` does no I/O.
- 2026-05-16 (Phase 6) — `install-claude-code` writes config at 0o600 and merges as a JSON-object union.
- **2026-05-16 (Phase 7)** — `RejudgeDrifts` updates rows in place (same `run_id`) so `report` and the MCP tools see the corrected verdict immediately. The rejudge model column flip records who said what; the prompt version stays `match-v1` because the prompt is identical (only the candidate set narrows).
- **2026-05-16 (Phase 7)** — User confirmed `CGO_ENABLED=1` is acceptable and preferred over a partial `CGO_ENABLED=0` build. Cross-platform release automation is deferred until target C toolchains are explicitly configured.
- **2026-05-16 (Phase 7)** — Grafana/ClickHouse coverage dashboard deferred. None of the upstream consumers have ClickHouse plumbing yet; revisit when one does.

## Verification log (2026-05-16)
- `make lint` → 0 issues across the tree.
- `make test -race` → all seven internal packages pass (db, embed, fsd, code, match, report, mcp).
- Manual SSE handshake against the built binary: spun up `mcp serve --transport sse --addr 127.0.0.1:<random>`, opened `/sse`, received the `endpoint` event with the per-session message URL, posted JSON-RPC `initialize`/`tools/list`/`tools/call(fsd_coverage_summary)`, observed:
  - server identity `fsdtrace 0.1.0`
  - 9 tools listed
  - structured coverage payload returned over the SSE stream
- `--rejudge-drifts` CLI plumbing wired (visible in `fsdtrace match --help`), and the `RejudgeDrifts` pipeline method is exercised end-to-end by `TestRejudgeDrifts_PromotesAndPersists` against a fake Opus server: pre-existing drifts row → Opus call → row UPSERT-updated to `implements`, confidence 0.95, model column rewritten, notes contain the rejudge marker.

## Open questions

- None blocking under the CGO-enabled build contract. Cross-platform release automation remains deferred until target C toolchains are configured.

## Next steps (post-Phase 7)

- Wire a real Bedrock cassette for the Phase 1 `make smoke` end-to-end (currently it just runs `init`). Best done when the smoke needs to exercise live judgment too.
- Phase 7 leaves Grafana/ClickHouse for a future request. The data already lives in SQLite and the JSON report is lossless, so a downstream consumer can pick it up at any time.
- Once a real Spring repo + `scip-java` are available, exercise the live SCIP merge path to confirm `relationships` populate as expected.
