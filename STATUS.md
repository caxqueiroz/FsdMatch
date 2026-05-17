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

## Post-Phase 7 maintenance

- 2026-05-16 — Implemented `fsdtrace embed`. The command now repopulates
  `feature_vec` and/or `artifact_vec` for existing rows, honours
  `--what features|artifacts|all`, `--embedding-model`, and `--cassette`,
  and writes vec rows through `internal/db.Writer`. Added CLI regression
  coverage for vector population and invalid `--what` validation.
- 2026-05-16 — Fixed GitHub Actions CI after the manual
  `golangci-lint` install script failed checksum verification on
  `v2.12.2`. CI now uses the official `golangci/golangci-lint-action`
  pinned to the same linter version.
- 2026-05-16 — Tracked the synthetic `testdata/` fixtures required by
  the parser, atomizer, and Spring harvester tests. They were present
  locally but ignored, which made CI fail with missing fixture paths.
- 2026-05-16 — Added `README.md` with build/pipeline basics and a
  `scip-java` section explaining its role in semantic Java indexing and
  call graph relationship population.
- 2026-05-17 — Added Go Task workflows and a platform-specific zip
  distribution path. After unzip, `task run` renders the bundled demo DB
  into an HTML report; `task trace FSD=... REPO=...` runs a real project,
  with `SCIP=true` or `SCIP_INDEX=...` enabling the optional semantic call
  graph layer.
- 2026-05-17 — Added optional SCIP call graph report enrichment. The
  `report --include-call-graph` flag and Taskfile `INCLUDE_CALL_GRAPH=true`
  attach reachable support artifacts to directly implemented matches without
  changing default surface-only coverage classification.
- 2026-05-17 — Replaced Titan-specific embedding construction with a
  Bedrock embedding adapter factory. Titan remains the default, while Cohere
  Embed v3/v4 model IDs now use Cohere request/response shapes. Cohere stored
  rows are embedded with `search_document`, matcher/MCP lookup vectors use
  `search_query`, and Cohere v4 is forced to the existing 1024-dim vec0 schema.
- 2026-05-17 — Added the v1.0 GitHub tracing workflow. `fsdtrace trace github
  <url> --fsd <path>` downloads a public GitHub repository via the zipball API
  and runs the existing init/ingest/index/match/report pipeline. Added a
  Spring PetClinic sample FSD under `examples/petclinic/`, Taskfile
  `trace-github` shortcuts, and bumped the reported MCP/CLI version to 1.0.0.
- 2026-05-17 — Added provider-selectable model access. Bedrock remains the
  default path through `BEDROCK_BASE_URL`, while `--provider openai` or
  `FSDTRACE_PROVIDER=openai` uses OpenAI directly with `OPENAI_API_KEY`.
  OpenAI generation defaults to `gpt-5.5`; OpenAI embeddings default to
  `text-embedding-3-large` with `dimensions=1024` so the existing vec0 schema
  is unchanged. The CLI, GitHub trace wrapper, and MCP rematch/search paths all
  resolve provider-specific defaults without breaking Bedrock cassettes.
  Taskfile workflows accept `PROVIDER=openai` for the same path.
- 2026-05-17 — Broadened the default FSD anchor pattern from numeric-only
  `FR-\d+` to scoped FR identifiers such as `FR-HOME-1` and `FR-I18N-3`.
  The parser also stops a final FR chunk at the next same/higher-level
  markdown section so trailing Data Model/NFR sections are not folded into
  the last functional requirement.
- 2026-05-17 — Added deterministic matcher reranking. Retrieval now builds a
  larger internal pool from anchors plus vec0 KNN, reranks by anchor matches,
  identifier/class/method/source/file term overlap, and vector distance, then
  sends only the final `--top-k` candidates to the judgment model. This keeps
  OpenAI prompts smaller without relying solely on raw vector order.
- 2026-05-17 — Added batched judgment and incomplete-response retry. OpenAI
  judgment now sends candidates in batches of 8 with a 12k output budget; if a
  provider reports an incomplete response, the matcher recursively splits that
  batch and retries. The match prompt was bumped to `match-v2`: models now
  return only `implements`/`drifts` rows, and omitted candidates are treated as
  `unrelated` by the caller.
- 2026-05-17 — Added optional FR-level match parallelism. `fsdtrace match` and
  `fsdtrace trace github` now accept `--match-concurrency N`; the default stays
  serial (`1`), while higher values overlap per-FR retrieval/judgment and keep
  all SQLite writes serialized through `internal/db.Writer`.
- 2026-05-17 — Added inline SVG SCIP graphs to HTML reports. When
  `--include-call-graph` is enabled, each implemented match with support
  artifacts now shows a small dependency graph rooted at the matched artifact,
  alongside the existing precise file:line support rows.
- 2026-05-18 — Added resilience for long live-provider runs. Bedrock and
  OpenAI HTTP clients now retry `429` and all `5xx` responses up to five times,
  honoring `Retry-After`. `ingest fsd`, `index code`, `match`, and
  `trace github` now accept `--resume` with a stable `--run-id`; the GitHub
  wrapper also requires `--checkout-dir` so artifact paths remain stable.
  Interactive runs render stderr progress bars while stdout stays scriptable.

## Next steps (post-Phase 7)

- 2026-05-16 — Model IDs are configurable through flags, env, or `fsdtrace.yaml`.
  Resolution order is `flag > env > config file > default`; supported env vars:
  `FSDTRACE_PROVIDER`, `FSDTRACE_EMBEDDING_MODEL`,
  `FSDTRACE_ATOMIZER_MODEL`, `FSDTRACE_JUDGMENT_MODEL`, and
  `FSDTRACE_REJUDGE_MODEL`.
- Wire a real Bedrock cassette for the Phase 1 `make smoke` end-to-end (currently it just runs `init`). Best done when the smoke needs to exercise live judgment too.
- Phase 7 leaves Grafana/ClickHouse for a future request. The data already lives in SQLite and the JSON report is lossless, so a downstream consumer can pick it up at any time.
- Once a real Spring repo + `scip-java` are available, exercise the live SCIP merge path to confirm `relationships` populate as expected.
