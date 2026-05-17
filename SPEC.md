# fsdtrace — FSD Validation for Spring Boot

A single Go binary that validates an FSD (Functional Specification Document) against a Java/Spring Boot codebase, producing a traceability matrix with file:line evidence. Exposes a CLI and an MCP server so Claude Code can query coverage, drift, and orphan endpoints directly.

This document is the authoritative design. Implementation must follow it; deviations require an explicit decision recorded in `STATUS.md`.

---

## 1. Goals & Non-Goals

**Goals**
- Atomize the FSD into structured FR (Functional Requirement) objects.
- Index the Spring Boot codebase with compiler-accurate semantics (SCIP) plus Spring annotation harvesting.
- Match FRs to code via anchor + embedding retrieval + model judgment, with evidence required.
- Detect drift (code that does *something similar* to the FR but not the same) and orphan code (public surface not mapped to any FR).
- Store everything in a single SQLite file. Make it queryable from CLI and from Claude Code over MCP.
- Re-runs are idempotent and diffable.

**Non-goals**
- General-purpose code search.
- Editing the codebase. fsdtrace is read-only over the code.
- Running the application under test. Coverage is inferred from static tests, not from runtime traces.

---

## 2. Stack (non-negotiable unless SPEC is updated)

| Concern | Choice |
|---|---|
| Language | Go 1.22+ |
| Build mode | `CGO_ENABLED=1` required. tree-sitter powers the Spring indexer and uses CGo. |
| SQL driver | `modernc.org/sqlite` (pure Go, works with or without CGO) |
| Vector KNN | `modernc.org/sqlite/vec` (vec0 virtual tables). Fallback: `github.com/viant/sqlite-vec` |
| Java AST | `github.com/smacker/go-tree-sitter` + `tree-sitter-java` (CGO) |
| SCIP parsing | `github.com/scip-code/scip/bindings/go/scip` (formerly `github.com/sourcegraph/scip` — module renamed at v0.7.x) |
| SCIP indexing | external `scip-java` CLI (JVM, user-installed) |
| MCP server | `github.com/mark3labs/mcp-go` |
| CLI framework | `github.com/spf13/cobra` |
| Config | `github.com/spf13/viper` (env + flags + config file) |
| Logging | `log/slog` (stdlib) |
| LLM access | Provider-selectable: default HTTP to KrakenD route → Amazon Bedrock (Anthropic + configurable 1024-dim embeddings; Titan default, Cohere supported), or direct OpenAI (`OPENAI_API_KEY`) with GPT-5.5 and OpenAI embeddings |

No FAISS. No native shared libraries beyond what tree-sitter compiles in. Build and release artifacts are CGO-enabled. Cross-compilation is not a current acceptance requirement; producing non-native artifacts requires an explicit C cross-toolchain setup for the target platform.

---

## 3. Architecture

Single binary, one SQLite file as the source of truth. Two parallel ingestion pipelines (FSD side and code side) feed a matcher orchestrator that calls the configured judgment model. Results land in the same DB and are surfaced via CLI and MCP.

```
FSD doc ──▶ atomizer ──▶ features + embeddings ─┐
                                                 ├──▶ matcher ──▶ matches ──▶ CLI / MCP
Spring Boot ──▶ indexer ──▶ artifacts + graph ──┘
```

The matcher pipeline per FR has four stages:
1. **Anchor match** — deterministic. Match by referenced URL, HTTP verb, topic, role, etc.
2. **Vector retrieval + rerank** — collect a larger internal candidate pool from anchors plus vec0 KNN, rerank deterministically with anchor, identifier, class/method, source-term, and distance signals, then keep the final top-K.
3. **Model judgment** — batched calls over the final candidates, with structured output of non-unrelated verdicts and mandatory file:line evidence.
4. **Test cross-check** — count and surface linked JUnit tests; flag `implemented-untested`.

---

## 4. Project Structure

```
fsdtrace/
├── cmd/fsdtrace/main.go            # cobra wiring
├── internal/
│   ├── db/
│   │   ├── schema.sql              # all tables + vec0 virtual tables
│   │   ├── migrations/             # numbered .sql files
│   │   ├── db.go                   # Open(path), pragmas, vec registration
│   │   └── writer.go               # single-writer goroutine + channel
│   ├── fsd/
│   │   ├── parse.go                # markdown/docx/pdf to chunks
│   │   └── atomizer.go             # chunks → FR objects via configured model
│   ├── code/
│   │   ├── scip.go                 # shell out to scip-java, parse protobuf
│   │   ├── treesitter.go           # tree-sitter-java walker
│   │   ├── spring.go               # annotation harvester
│   │   └── tests.go                # JUnit extractor + linking
│   ├── embed/
│   │   ├── bedrock.go              # HTTP client to KrakenD route
│   │   ├── openai.go               # OpenAI embeddings client
│   │   ├── cache.go                # idempotent embedding cache in SQLite
│   │   └── titan.go                # request/response shapes
│   ├── llm/
│   │   ├── bedrock.go              # Anthropic-on-Bedrock generation adapter
│   │   └── openai.go               # OpenAI Responses API generation adapter
│   ├── match/
│   │   ├── anchor.go               # deterministic stage-1 matchers
│   │   ├── retrieve.go             # vec0 KNN via internal/db
│   │   ├── judge.go                # judgment model call
│   │   ├── prompt.go               # judgment prompt (versioned)
│   │   └── pipeline.go             # orchestrate per-FR stages
│   ├── report/
│   │   ├── markdown.go
│   │   ├── csv.go
│   │   ├── html.go
│   │   └── json.go
│   └── mcp/
│       ├── server.go               # mark3labs/mcp-go setup
│       ├── tools.go                # tool definitions
│       ├── resources.go            # resource definitions
│       └── handlers.go             # tool implementations
├── testdata/
│   └── sample-spring-app/          # tiny Spring Boot fixture for smoke tests
├── Makefile
├── go.mod
├── go.sum
├── .golangci.yml
├── STATUS.md                       # current phase + decisions log
├── CLAUDE.md                       # agent operating instructions
└── SPEC.md                         # this file
```

---

## 5. CLI Surface

All subcommands operate on a single `--db` file (default `./fsdtrace.db`).

```
fsdtrace init                       # create DB, apply schema
fsdtrace ingest fsd <path>          # atomize FSD into FR objects
fsdtrace index code <repo>          # run scip-java + annotation harvest
fsdtrace embed [--what features|artifacts|all]
fsdtrace match [--fr FR-042] [--top-k 15] [--match-concurrency 1]
fsdtrace report --format md|csv|html|json --out ./trace/
fsdtrace trace github <url> --fsd <path> [--match-concurrency 1]  # download GitHub repo and run full pipeline
fsdtrace mcp serve [--transport stdio|sse]
fsdtrace install-claude-code        # writes Claude Code MCP config entry
fsdtrace status                     # latest run summary
```

Global flags: `--db`, `--config`, `--log-level`, `--run-id`.

The full pipeline is: `init → ingest fsd → index code → embed → match → report`.
`trace github` is an additive convenience wrapper over that pipeline. It accepts
only `https://github.com/<owner>/<repo>` URLs, downloads the selected ref through
GitHub's zipball API, extracts it into a temporary directory by default, and
then runs the same FSD ingestion, code indexing, matching, and report generation
steps. Existing lower-level subcommands remain the stable primitive interface.

---

## 6. Database Schema

The schema is canonical. Place it in `internal/db/schema.sql`. Migrations are forward-only and numbered (`migrations/001_initial.sql`).

```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;

-- ============ FEATURES ============
CREATE TABLE features (
  id           TEXT PRIMARY KEY,        -- e.g. FR-042
  title        TEXT NOT NULL,
  description  TEXT NOT NULL,
  acceptance   JSON NOT NULL,           -- array of criteria
  actor        TEXT,
  inputs       JSON,
  outputs      JSON,
  side_effects JSON,
  non_functional JSON,
  fsd_section  TEXT,
  fsd_anchor   TEXT,                    -- file#section
  run_id       TEXT NOT NULL,
  created_at   INTEGER NOT NULL
);

CREATE INDEX idx_features_section ON features(fsd_section);

-- ============ CODE ARTIFACTS ============
CREATE TABLE code_artifacts (
  id           INTEGER PRIMARY KEY,     -- == vec0 rowid
  kind         TEXT NOT NULL,           -- rest_endpoint | kafka_listener |
                                        -- rabbit_listener | jms_listener |
                                        -- event_listener | scheduled_job |
                                        -- service_method | security_rule |
                                        -- entity | repository | config_props |
                                        -- exception_handler
  identifier   TEXT NOT NULL,           -- e.g. POST /api/v1/orders
  scip_symbol  TEXT,                    -- fully qualified
  package      TEXT,
  class        TEXT,
  method       TEXT,
  file         TEXT NOT NULL,
  start_line   INTEGER NOT NULL,
  end_line     INTEGER NOT NULL,
  signature    TEXT,
  annotations  JSON,                    -- harvested @-tags + attrs
  source       TEXT,                    -- method body slice
  run_id       TEXT NOT NULL
);

CREATE INDEX idx_artifacts_kind ON code_artifacts(kind);
CREATE INDEX idx_artifacts_symbol ON code_artifacts(scip_symbol);

-- ============ CALL GRAPH ============
CREATE TABLE relationships (
  from_artifact INTEGER NOT NULL REFERENCES code_artifacts(id),
  to_artifact   INTEGER NOT NULL REFERENCES code_artifacts(id),
  kind          TEXT NOT NULL,          -- calls | implements | advises | overrides
  PRIMARY KEY (from_artifact, to_artifact, kind)
);

-- ============ TESTS ============
CREATE TABLE tests (
  id              INTEGER PRIMARY KEY,
  name            TEXT NOT NULL,
  file            TEXT NOT NULL,
  line            INTEGER NOT NULL,
  test_kind       TEXT,                 -- WebMvcTest | SpringBootTest | DataJpaTest | unit
  target_artifact INTEGER REFERENCES code_artifacts(id),
  asserts         JSON,                 -- summary of assertions
  run_id          TEXT NOT NULL
);

CREATE INDEX idx_tests_target ON tests(target_artifact);

-- ============ MATCHES ============
CREATE TABLE matches (
  run_id        TEXT NOT NULL,
  feature_id    TEXT NOT NULL REFERENCES features(id),
  artifact_id   INTEGER NOT NULL REFERENCES code_artifacts(id),
  verdict       TEXT NOT NULL,          -- implements | drifts | unrelated
  confidence    REAL NOT NULL,          -- 0..1
  evidence      JSON NOT NULL,          -- [{file, start, end, note}]
  notes         TEXT,                   -- prose notes; matcher appends
                                        -- "; tested=<bool> test_count=<int>"
                                        -- as the test cross-check decoration
                                        -- (see §7.4). Reporters strip it.
  model         TEXT NOT NULL,
  prompt_version TEXT NOT NULL,
  PRIMARY KEY (run_id, feature_id, artifact_id)
);

CREATE INDEX idx_matches_feature ON matches(feature_id);
CREATE INDEX idx_matches_verdict ON matches(verdict);

-- ============ RUNS ============
CREATE TABLE runs (
  id           TEXT PRIMARY KEY,
  started_at   INTEGER NOT NULL,
  finished_at  INTEGER,
  summary      JSON,
  config       JSON
);

-- ============ EMBEDDING CACHE ============
CREATE TABLE embedding_cache (
  key          TEXT PRIMARY KEY,        -- sha256(model config + text)
  model        TEXT NOT NULL,
  dim          INTEGER NOT NULL,
  embedding    BLOB NOT NULL,           -- float32 packed
  created_at   INTEGER NOT NULL
);

-- ============ VEC0 VIRTUAL TABLES ============
-- Fixed vector dimension for the current embedding index. Adjust at create time.
CREATE VIRTUAL TABLE feature_vec USING vec0(
  embedding float[1024]
);

CREATE VIRTUAL TABLE artifact_vec USING vec0(
  embedding float[1024]
);
```

**Invariants**:
- `code_artifacts.id` and `artifact_vec.rowid` are kept in lock-step. Inserting into one without the other is a bug.
- `features.id` is used as `feature_vec.rowid` via a stable hash (FR-042 → integer rowid). Maintain a `feature_rowid` mapping table if hashing is risky.
- All writes happen on a single goroutine. See `internal/db/writer.go`.

---

## 7. Component Specifications

### 7.1 FSD Atomizer (`internal/fsd`)

**Input**: a markdown, PDF, or DOCX file. Phase 2 supports markdown only; PDF/DOCX in a later phase.

**Output**: rows in `features` plus embeddings in `feature_vec`.

**Strategy**:
- Split the document on FR anchors (default regex `\bFR(?:-[A-Z0-9]+)*-\d+\b`, e.g. `FR-001` or `FR-OWN-1`). User can override with `--anchor-pattern`.
- For each chunk, call the configured generation provider with a structured-output prompt that returns the FR fields.
- Embedding text = `title + "\n" + description + "\n" + acceptance criteria`.
- Embed via the configured provider. Bedrock defaults to Titan v2 and also supports Cohere Embed v3/v4 when vectors are 1024-dimensional. OpenAI defaults to `text-embedding-3-large` with `dimensions=1024`.
- Cache embeddings keyed by sha256 of model config + input so re-runs are idempotent.

The atomization prompt is in `internal/fsd/prompt.go` and versioned. Output schema is strict JSON matching `features` columns.

### 7.2 Code Indexer (`internal/code`)

Three sublayers writing to the same DB transaction.

**SCIP layer (`scip.go`)**:
- Shell out: `scip-java index --output index.scip`. User must have `scip-java` on PATH.
- Parse with `github.com/sourcegraph/scip/bindings/go/scip`.
- Insert `code_artifacts` rows from `SymbolInformation`; insert `relationships` from `Occurrence` with `SymbolRole_Reference` linking call sites.
- The `scip_symbol` column is the fully-qualified Sourcegraph symbol string.

**Tree-sitter layer (`treesitter.go`, `spring.go`)**:
- Walk all `.java` files with `tree-sitter-java`.
- Match annotation patterns. Annotation harvester must cover at minimum:
  - `@RestController`, `@Controller`, `@RequestMapping`, `@GetMapping`, `@PostMapping`, `@PutMapping`, `@DeleteMapping`, `@PatchMapping` → `kind=rest_endpoint`
  - `@KafkaListener` → `kind=kafka_listener` (capture `topics`, `groupId`)
  - `@RabbitListener`, `@JmsListener`, `@EventListener` → corresponding kinds
  - `@Scheduled` → `kind=scheduled_job` (capture `cron`, `fixedRate`, `fixedDelay`)
  - `@PreAuthorize`, `@PostAuthorize`, `@Secured`, `@RolesAllowed` → `kind=security_rule`
  - `@Transactional` → attribute on the parent method's existing row, not a new row
  - `@Entity`, `@Table` → `kind=entity`
  - `@ConfigurationProperties` → `kind=config_props`
  - `@ConditionalOnProperty`, `@ConditionalOnBean`, etc. → attribute on the parent's row
  - `@RestControllerAdvice` + `@ExceptionHandler` → `kind=exception_handler`
  - Spring Security `SecurityFilterChain` DSL → custom recognizer for `http.authorizeHttpRequests(...)` patterns
- Each artifact row's `scip_symbol` is filled in by joining against the SCIP-layer output by FQN.

**Test layer (`tests.go`)**:
- Same tree-sitter pass extracts `@Test` methods plus `@SpringBootTest`/`@WebMvcTest`/`@DataJpaTest` slice classes.
- Capture MockMvc paths exercised (e.g., `mockMvc.perform(get("/api/v1/orders"))`) by walking method bodies.
- Link tests to `code_artifacts` by walking the SCIP call graph from the test method.

### 7.3 Embedder (`internal/embed`)

- Provider is selected by `--provider`, `FSDTRACE_PROVIDER`, or config. Supported values: `bedrock` (default) and `openai`.
- Bedrock targets `BEDROCK_BASE_URL` (env). User's KrakenD route handles AWS SigV4, model routing, and SSE pass-through.
- Bedrock default model: `amazon.titan-embed-text-v2:0`. Cohere adapters are selected from model ID: `cohere.embed-english-v3`, `cohere.embed-multilingual-v3`, and `cohere.embed-v4` use Cohere request/response shapes; v4 is invoked with `output_dimension=1024`.
- OpenAI targets `https://api.openai.com/v1` by default and requires `OPENAI_API_KEY`. OpenAI default embedding model: `text-embedding-3-large`, invoked with `dimensions=1024`.
- Stored vector dimension: 1024.
- Providers with purpose-specific retrieval modes must embed stored corpus rows as documents and lookup text as queries. Cohere uses `search_document` for `feature_vec`/`artifact_vec` rows and `search_query` for matcher/MCP search vectors.
- Batch up to 25 texts per request.
- All embeddings cached in `embedding_cache` keyed by `sha256(model config + text)` so provider-specific options such as Cohere `input_type` and output dimension do not collide.
- Exponential backoff with jitter on 429/503. Max 5 retries.
- Per-run token budget cap (configurable). Aborts the run with a clear error if exceeded.

### 7.4 Matcher (`internal/match`)

Per FR pipeline implemented in `pipeline.go`:

```go
func (p *Pipeline) MatchFeature(ctx context.Context, fr Feature) ([]Match, error)
```

Stages:
1. **Anchor** (`anchor.go`): regex/keyword extraction over FR text against extracted artifacts. URL paths, HTTP verbs, topic names, role names, scheduled cron hints.
2. **Retrieve + rerank** (`retrieve.go`): collect anchor candidates plus an internal vec0 KNN pool larger than top-K, then deterministically rerank by anchor hits, query-term overlap against identifier/class/method/source/file, and vector distance. Only the final top-K=15 by default is sent to the judge.
3. **Judge** (`judge.go`): configured model calls over candidate batches. Prompt is in `prompt.go` and carries a `prompt_version` string written into `matches.prompt_version`.
   - Required output: `[{artifact_id, verdict, confidence, evidence: [{file, start, end, note}], notes}]`
   - Verdict ∈ `implements | drifts`.
   - Unrelated candidates are omitted from the response and treated as `unrelated` by the caller.
   - No evidence → reject the verdict and downgrade to `unrelated`.
   - If a provider reports an incomplete response, the matcher retries that batch by recursively splitting it into smaller batches.
   - `--match-concurrency N` lets the pipeline match N FRs in parallel. It defaults to `1`; DB writes still go through `internal/db.Writer`.
4. **Test cross-check**: count linked tests via `tests.target_artifact`. Decorate matches with `tested: bool` and `test_count`. The matches schema has no dedicated columns; the matcher persists the decoration as a suffix on `matches.notes` in the form `"…; tested=<bool> test_count=<int>"`. Reporters and the MCP `fsd_get_feature` tool strip the suffix back out at read time.

The default Bedrock judgment model is `anthropic.claude-sonnet-*-v2:0`. The default OpenAI generation model is `gpt-5.5`; OpenAI judgment uses smaller candidate batches by default to reduce response truncation. `drifts` verdicts may be re-judged in a second pass (gated by `--rejudge-drifts`), defaulting to Opus on Bedrock and GPT-5.5 on OpenAI.

### 7.5 Reporter (`internal/report`)

Generates the traceability matrix in four formats. Default output dir `./trace/`.

- `markdown.go`: human-readable matrix, FR-by-FR with evidence snippets.
- `csv.go`: machine-readable, one row per match.
- `html.go`: same as markdown but static HTML with collapsible sections.
  When `--include-call-graph` is enabled, implemented matches with SCIP
  support also render an inline SVG call graph rooted at the matched artifact.
- `json.go`: full dump for downstream tooling.

Coverage rollup per FSD section. Drift section listing all `drifts` verdicts. Orphans section listing artifacts with no `implements` mapping.

### 7.6 MCP Server (`internal/mcp`)

`mark3labs/mcp-go` server, default stdio transport. SSE transport supported for remote setups.

**Tools** (all read-only unless noted):

| Tool | Purpose | Annotations |
|---|---|---|
| `fsd_search_features` | Semantic search over FRs. Returns id, title, acceptance, section. | readOnly |
| `fsd_search_code` | Semantic search over code artifacts. Filterable by `kind`. Returns file:line, signature. | readOnly |
| `fsd_get_feature` | Get an FR with matched artifacts and test coverage. | readOnly |
| `fsd_get_artifact` | Get a code artifact with linked FRs and tests. | readOnly |
| `fsd_list_unmatched` | FRs with no `implements` verdict. Filter by section. | readOnly |
| `fsd_list_orphans` | Public artifacts (endpoints, listeners, scheduled jobs) with no FR. Filter by kind. | readOnly |
| `fsd_drift_report` | All `drifts` verdicts with evidence. | readOnly |
| `fsd_coverage_summary` | Counts per FSD section: implemented / drifts / missing / untested. | readOnly |
| `fsd_rematch_feature` | Re-run the matcher for one FR. Costs one configured judgment call. | NOT readOnly |

Each tool has a clear description, JSON schema for inputs, and an output schema. Tool descriptions follow `mcp-builder` conventions: action-oriented, concise, with examples in field descriptions.

**Resources**:
- `fsd://coverage` → coverage matrix as markdown
- `fsd://drift` → drift report as markdown
- `fsd://features` → feature catalog as JSON

**Transport**: stdio is the default. Claude Code launches the binary as a subprocess. Startup cost must stay low — open the DB lazily, do not load embeddings into RAM.

---

## 8. Configuration

Resolution order: flag > env > config file > default.

```yaml
# fsdtrace.yaml
db: ./fsdtrace.db
provider: bedrock # bedrock|openai
bedrock:
  base_url: https://krakend.internal/v1/bedrock
  region: us-east-1
  embedding_model: amazon.titan-embed-text-v2:0
  atomizer_model: anthropic.claude-sonnet-4-v2:0
  judgment_model: anthropic.claude-sonnet-4-v2:0
  rejudge_model: anthropic.claude-opus-4-v2:0
  rejudge_drifts: false
  token_budget: 2000000
openai:
  # Optional. Defaults to https://api.openai.com/v1.
  base_url: https://api.openai.com/v1
  embedding_model: text-embedding-3-large
  atomizer_model: gpt-5.5
  judgment_model: gpt-5.5
  rejudge_model: gpt-5.5
fsd:
  anchor_pattern: '\bFR(?:-[A-Z0-9]+)*-\d+\b'
indexer:
  scip_java_bin: scip-java
  java_project_dir: ./
matcher:
  top_k: 15
  min_confidence: 0.6
mcp:
  transport: stdio
```

Model resolution order is `flag > env > config file > default`.
Provider resolution order is `--provider > FSDTRACE_PROVIDER > config file > bedrock`.
Supported provider/model env vars:
`FSDTRACE_PROVIDER`,
`FSDTRACE_EMBEDDING_MODEL`, `FSDTRACE_ATOMIZER_MODEL`,
`FSDTRACE_JUDGMENT_MODEL`, and `FSDTRACE_REJUDGE_MODEL`.
Bedrock requires `BEDROCK_BASE_URL`; OpenAI requires `OPENAI_API_KEY`.
`FSDTRACE_OPENAI_BASE_URL` can override the OpenAI API base URL for tests or
compatible gateways.

---

## 9. Build & Distribution

```makefile
build:
	CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o bin/fsdtrace ./cmd/fsdtrace

test:
	go test -race ./...

smoke:
	./scripts/smoke.sh

install:
	CGO_ENABLED=1 go install -trimpath -ldflags="-s -w" ./cmd/fsdtrace

cross:
	./scripts/cross-compile.sh   # currently fails fast; requires CGO cross toolchains

lint:
	golangci-lint run ./...
```

Release archives should be built natively per target OS/architecture, or through an explicitly configured CGO cross toolchain. There is no pure-Go release mode today.

---

## 10. Failure Modes (must be designed for)

| Failure mode | Mitigation |
|---|---|
| modernc/sqlite single-writer contention | All writes through `internal/db.Writer` (channel-fed goroutine). Reads concurrent. |
| `modernc.org/sqlite/vec` is young; may have bugs | Pin version. Provide `viant/sqlite-vec` fallback behind build tag. Smoke test on every CI run. |
| scip-java requires successful compile | Document precondition. Fail fast with actionable error if `target/` or `build/` absent. |
| Lombok/MapStruct generated code | scip-java must run *after* `mvn compile` or `gradle classes`. Document. |
| Spring meta-annotations (`@MyBusinessRule` annotated with `@Component`) | Resolve via SCIP type info; annotation harvester consults both layers and unions. |
| AOP advice (`@Around`, `@Before`) invisible to static call graph | Extract advice rows separately with `kind=advice`; surface in matcher prompt context. |
| Reflection / `BeanFactory.getBean(String)` | Flag as `unresolved_dynamic_dispatch` artifacts; surface in orphan report. |
| `@ConditionalOnProperty` / Spring Profiles | Harvester records conditions; matcher prompt mentions them so judgment notes "implemented under profile=prod". |
| Model call budget overrun | Per-run token budget cap; abort with clear error. |
| Judgment model produces no evidence | Reject verdict; downgrade to `unrelated`. Hard rule, no exceptions. |
| MCP stdio subprocess relaunch on Claude Code restart | Lazy DB open; no eager loading. Startup must be <200ms cold. |

---

## 11. Phased Plan

Each phase has explicit acceptance criteria. Do not advance phases without meeting them. Update `STATUS.md` when you complete a phase.

### Phase 1 — Foundation
- `go.mod` with module `github.com/cax/fsdtrace`.
- Cobra CLI skeleton with all subcommands (stub bodies OK except `init`).
- `internal/db` with schema.sql, `Open()`, WAL/busy_timeout/foreign_keys, `modernc.org/sqlite/vec` registration.
- Single-writer goroutine in `internal/db/writer.go`.
- `internal/db/smoke_test.go` inserts 50 random 1024-dim vectors, queries top-5 KNN, verifies sorted by distance.
- Makefile targets: `build`, `test`, `smoke`, `install`, `lint`, `cross`.
- `.golangci.yml` copied from the `golang-style` skill assets.

**Acceptance**: `make build` succeeds with `CGO_ENABLED=1`. `make test` passes. `./bin/fsdtrace init --db /tmp/t.db` creates a valid SQLite file containing all tables and vec0 virtual tables.

### Phase 2 — FSD Ingestion
- `internal/fsd/parse.go` for markdown.
- `internal/fsd/atomizer.go` calling the configured generation provider via the provider-neutral LLM adapter.
- Atomizer prompt in `prompt.go` with `PromptVersion = "fsd-atomize-v1"`.
- Embedding cache populated; FRs embedded into `feature_vec`.
- Atomizer test against `testdata/fsd-sample.md` (5 FRs).

**Acceptance**: `fsdtrace ingest fsd testdata/fsd-sample.md` populates `features` with 5 rows and `feature_vec` with 5 vectors. Re-running is idempotent (no duplicate rows).

### Phase 3 — Code Indexing
- `internal/code/scip.go` shells out to scip-java, parses Protobuf, populates `code_artifacts` and `relationships`.
- `internal/code/treesitter.go` and `spring.go` extract annotations for all kinds listed in §7.2.
- `internal/code/tests.go` extracts JUnit tests.
- Integration test against `testdata/sample-spring-app/` (a fixture with 1 controller, 1 service, 1 listener, 1 scheduled job, 1 test).

**Acceptance**: `fsdtrace index code testdata/sample-spring-app` populates `code_artifacts` with at least one row per artifact kind in the fixture, plus the test in `tests`. Embeddings populated in `artifact_vec`.

### Phase 4 — Matching
- `internal/match/anchor.go`, `retrieve.go`, `judge.go`, `pipeline.go`.
- Judgment prompt in `prompt.go` with `PromptVersion = "match-v1"`.
- Hard rule enforced: no evidence → downgrade to `unrelated`.
- Test cross-check decorates verdicts with `tested` and `test_count`.

**Acceptance**: `fsdtrace match` against the fixture produces a `matches` row per (FR, artifact) candidate with verdicts and evidence spans. `STATUS.md` records actual coverage on the fixture.

### Phase 5 — Reporting
- `internal/report/{markdown,csv,html,json}.go`.
- Coverage rollup per FSD section. Drift section. Orphan section.

**Acceptance**: `fsdtrace report --format md` produces `./trace/coverage.md`, `./trace/drift.md`, `./trace/orphans.md`.

### Phase 6 — MCP Server
- `internal/mcp/server.go` with stdio transport.
- All tools from §7.6, with descriptions, JSON schemas, and annotations.
- Resources: `fsd://coverage`, `fsd://drift`, `fsd://features`.
- `fsdtrace install-claude-code` subcommand writes an entry to `~/.claude.json`.

**Acceptance**: `fsdtrace mcp serve --transport stdio` responds to a manual MCP request via stdin. Tools listed and callable. Claude Code can list the tools after `install-claude-code` runs.

### Phase 7 — Polish
- SSE transport for MCP (`mcp serve --transport sse --addr ...`).
- `--rejudge-drifts` second pass (uses `--rejudge-model`, default Opus).
- Native CGO build tested in CI. Cross-compilation is deferred until a target C toolchain is introduced.
- Coverage Grafana dashboard config (optional; ClickHouse export). Deferred — none of the upstream consumers have ClickHouse plumbing yet; revisit when one does.

---

## 12. References

- modernc.org/sqlite: https://pkg.go.dev/modernc.org/sqlite
- modernc.org/sqlite/vec: https://pkg.go.dev/modernc.org/sqlite/vec
- sqlite-vec: https://github.com/asg017/sqlite-vec
- viant/sqlite-vec (fallback): https://github.com/viant/sqlite-vec
- SCIP: https://github.com/sourcegraph/scip
- scip-java: https://sourcegraph.github.io/scip-java/
- tree-sitter-java: https://github.com/tree-sitter/tree-sitter-java
- mark3labs/mcp-go: https://github.com/mark3labs/mcp-go
- Anthropic on Bedrock: https://docs.anthropic.com/en/api/claude-on-amazon-bedrock
- OpenAI Responses API: https://platform.openai.com/docs/api-reference/responses/create
- OpenAI Embeddings API: https://platform.openai.com/docs/api-reference/embeddings/create
- OpenAI GPT-5.5 model: https://platform.openai.com/docs/models/gpt-5.5
- MCP protocol: https://modelcontextprotocol.io/
