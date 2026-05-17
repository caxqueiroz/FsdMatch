# fsdtrace

`fsdtrace` validates a Functional Specification Document (FSD) against a
Java/Spring Boot codebase and produces a traceability matrix with file:line
evidence. It ships as a Go CLI and MCP server backed by SQLite and vec0.

## Build

`CGO_ENABLED=1` is required because the Spring indexer uses tree-sitter.

```bash
make build
```

The binary is written to:

```bash
./bin/fsdtrace
```

## Basic Pipeline

```bash
./bin/fsdtrace --db ./fsdtrace.db init
./bin/fsdtrace --db ./fsdtrace.db ingest fsd path/to/fsd.md
./bin/fsdtrace --db ./fsdtrace.db index code path/to/spring-repo
./bin/fsdtrace --db ./fsdtrace.db match
./bin/fsdtrace --db ./fsdtrace.db report --format html --out ./trace/
```

For a GitHub-hosted Spring project, use the end-to-end wrapper:

```bash
./bin/fsdtrace --db ./petclinic.db trace github \
  https://github.com/spring-projects/spring-petclinic \
  --fsd examples/petclinic/fsd.md \
  --out ./trace/petclinic \
  --format html
```

Live atomization, embedding, and judgment calls default to the existing
Bedrock/KrakenD route:

```bash
export BEDROCK_BASE_URL="https://your-krakend-bedrock-route"
```

To use OpenAI directly instead:

```bash
export OPENAI_API_KEY="sk-..."
./bin/fsdtrace --db ./petclinic.db trace github \
  https://github.com/spring-projects/spring-petclinic \
  --provider openai \
  --fsd examples/petclinic/fsd.md \
  --out ./trace/petclinic \
  --format html \
  --match-concurrency 3
```

With `--provider openai`, atomization, matching, and drift rejudging default
to `gpt-5.5`, and embeddings default to `text-embedding-3-large` with
`dimensions=1024` to match the existing vec0 schema. Bedrock remains the
default provider, with Titan v2 as the default embedding model and Cohere
Embed v3/v4 supported through the same Bedrock route.

OpenAI judgment uses deterministic reranking, smaller candidate batches, and
automatic retry on incomplete responses. The final `--top-k` candidates are
still honored, but they are judged in batches to avoid oversized JSON outputs.
Use `--match-concurrency N` to match multiple FRs in parallel; start with
`2` or `3` for live OpenAI runs to improve throughput without creating a large
burst of API calls.

Provider HTTP calls retry `429` and all `5xx` responses up to five times with
backoff and `Retry-After` support. Long runs can resume from a stable run id:
pass `--run-id <id> --resume` to `ingest fsd`, `index code`, or `match`.
For the GitHub wrapper, also pass a stable checkout path:

```bash
./bin/fsdtrace --db ./petclinic.db --run-id petclinic-v1 trace github \
  https://github.com/spring-projects/spring-petclinic \
  --provider openai \
  --fsd examples/petclinic/fsd.md \
  --checkout-dir ./work/spring-petclinic \
  --out ./trace/petclinic \
  --format html \
  --resume
```

Interactive runs show progress bars on stderr; stdout remains the summary line
for scripts.

Override models with `--embedding-model`, `--atomizer-model`,
`--judgment-model`, `--rejudge-model`, the matching `FSDTRACE_*_MODEL` env var,
or provider-specific config such as `bedrock.embedding_model` and
`openai.embedding_model` in `fsdtrace.yaml`.

## Taskfile Workflow

The repository includes a Go Task `Taskfile.yml`.

Run the bundled offline demo:

```bash
task run
```

Run against a real FSD and Spring Boot repo:

```bash
task trace FSD=/path/to/fsd.md REPO=/path/to/spring-repo
```

Run against Spring PetClinic from GitHub:

```bash
task trace-github GITHUB_REPO=https://github.com/spring-projects/spring-petclinic \
  FSD=examples/petclinic/fsd.md OUT=./trace/petclinic
```

That real run uses live Bedrock through `BEDROCK_BASE_URL` by default. Pass
`PROVIDER=openai` to the Taskfile, or `--provider openai` to the binary, and
set `OPENAI_API_KEY` to use OpenAI directly. Cassette paths are Bedrock-only
and remain intended for offline tests/demos.

Enable the optional SCIP layer during a real run:

```bash
task trace FSD=/path/to/fsd.md REPO=/path/to/spring-repo SCIP=true
```

Include SCIP call graph support artifacts in the generated report:

```bash
task trace FSD=/path/to/fsd.md REPO=/path/to/spring-repo SCIP=true INCLUDE_CALL_GRAPH=true
```

HTML reports generated with `--include-call-graph` include inline SVG call
graphs for implemented matches with SCIP support, while retaining the detailed
text rows for exact file:line evidence.

Or use a prebuilt SCIP index:

```bash
task trace FSD=/path/to/fsd.md REPO=/path/to/spring-repo SCIP_INDEX=/path/to/index.scip
```

## Zip Distribution

Build a platform-specific zip:

```bash
task dist
```

or:

```bash
make dist
```

The archive is written under `./dist/`, for example:

```text
dist/fsdtrace-darwin-arm64.zip
```

The zip contains:

- `bin/fsdtrace`
- `Taskfile.yml`
- `README.md`
- `CHANGELOG.md`
- `examples/petclinic/fsd.md`
- `demo/fsdtrace.db`
- bundled sample FSD, sample Spring app, and recorded Bedrock cassettes

After unzip:

```bash
cd fsdtrace-darwin-arm64
task run
```

`task run` renders the bundled offline demo report from `demo/fsdtrace.db` and
writes the HTML report to `./trace/`. For a real project, use
`task trace FSD=... REPO=...`.

The zip does not bundle Go Task itself, a JDK, `scip-java`, Maven, Gradle, a
Bedrock gateway, or OpenAI credentials. Go Task is required only for the
Taskfile workflow. The `fsdtrace` binary itself is included.

## Role of scip-java

`scip-java` provides compiler-aware Java semantics for richer code indexing.
It is separate from fsdtrace and must be installed by the user when semantic
call graph support is needed.

The code indexer has two layers:

1. Tree-sitter harvesting finds Spring surfaces directly from source files,
   such as `@RestController`, `@GetMapping`, `@KafkaListener`, `@Scheduled`,
   `@Entity`, and `@Repository`. This creates `code_artifacts`.
2. SCIP merge reads an `index.scip` file generated by `scip-java`. This fills
   `scip_symbol` on artifacts and inserts call graph edges into the
   `relationships` table.

Tree-sitter answers: what Spring surfaces exist?
`scip-java` helps answer: what does this code call or depend on?

The semantic call graph is optional because the tree-sitter layer can already
find the Spring public surface that fsdtrace must validate. That keeps the
default workflow usable with only the fsdtrace binary and source files. The
SCIP layer improves matching quality by adding compiler-aware relationships,
but it needs a JDK, `scip-java`, and a target project that has already been
compiled.

Reports are surface-only by default. To show SCIP-derived support artifacts
reachable from directly implemented matches, pass:

```bash
./bin/fsdtrace --db ./fsdtrace.db report --format html --out ./trace/ --include-call-graph
```

This option enriches the report; it does not reclassify FRs or orphan public
surfaces as covered.

Basic indexing works without `scip-java`:

```bash
./bin/fsdtrace --db ./fsdtrace.db index code path/to/spring-repo
```

To run `scip-java` during indexing:

```bash
./bin/fsdtrace --db ./fsdtrace.db index code path/to/spring-repo --run-scip-java
```

To use an existing SCIP index:

```bash
./bin/fsdtrace --db ./fsdtrace.db index code path/to/spring-repo --scip-index path/to/index.scip
```

If `--run-scip-java` is used and `scip-java` is missing from `PATH`, fsdtrace
fails fast with an install hint. Install instructions are here:

```text
https://sourcegraph.github.io/scip-java/docs/getting-started.html
```

`scip-java` expects the target Spring Boot project to compile. Run `mvn compile`
or `gradle classes` in the target repo before generating the SCIP index.

## Verification

```bash
make lint test
make smoke
```
