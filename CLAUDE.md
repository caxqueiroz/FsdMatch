# Operating Instructions for Claude Code

You are working on **fsdtrace**. Read `SPEC.md` for the full design before doing anything substantive. Read `STATUS.md` for the current phase and recent decisions. This file is the operating contract.

---

## What this project is

A Go CLI + MCP server that validates a Functional Specification Document against a Java/Spring Boot codebase. Single static binary, SQLite + vec0 storage, Bedrock Claude for atomization and judgment. See `SPEC.md` §3 for the architecture.

---

## Build & test commands

```
make build      # CGO_ENABLED=1 go build → ./bin/fsdtrace
make test       # go test -race ./...
make smoke      # end-to-end against testdata/sample-spring-app
make lint       # golangci-lint run ./...
make cross      # unsupported unless a CGO cross toolchain is added
make install    # go install
```

Run `make lint test` before declaring any task done. If lint fails, fix the lints; do not disable rules to make them pass.

---

## Hard constraints — do not violate without explicit user approval

1. **`CGO_ENABLED=1` is required.** The Spring indexer depends on tree-sitter, which uses CGo. Do not promise or add a `CGO_ENABLED=0` build unless the indexer is replaced or a real no-indexer build split is implemented and documented. modernc.org/sqlite remains the only SQL driver.
2. **modernc.org/sqlite is single-writer.** All writes must go through `internal/db.Writer` (a single goroutine fed by a channel). Reads are concurrent and unrestricted.
3. **WAL mode + `busy_timeout=5000` + `foreign_keys=ON`** on every connection. These pragmas live in `internal/db/db.go`.
4. **Embedding/judgment calls go via `internal/embed.BedrockClient`.** Never hardcode AWS endpoints. Always read `BEDROCK_BASE_URL` from env. This client targets the user's KrakenD route, not Bedrock directly.
5. **Prompts are versioned.** Any change to a prompt template requires bumping its `PromptVersion` constant. The version is written into `matches.prompt_version` or `features.atomizer_version` so historical runs remain attributable.
6. **Evidence is mandatory.** Any match verdict from Claude that does not include `file:start_line-end_line` evidence is downgraded to `unrelated`. No exceptions.
7. **vec0 rowid invariant.** `code_artifacts.id` must equal `artifact_vec.rowid` for the same artifact. Inserts must be paired in the same transaction.
8. **No proprietary code in fixtures.** `testdata/sample-spring-app/` is a synthetic minimal Spring Boot app authored for this project. Do not copy code from real Spring projects.
9. **Public CLI surface is fixed.** The subcommand list in `SPEC.md` §5 is the contract. Do not add, rename, or remove subcommands without updating SPEC and asking the user.

---

## Go style

Follow the `golang-style` skill. Key points enforced here:
- Package names: lowercase, single word.
- Receivers: 1–2 letter, consistent across a type.
- Errors: always check; wrap with `fmt.Errorf("doing X: %w", err)`.
- Logging: `log/slog` only; structured fields, no string formatting.
- Contexts: every public function that does I/O takes `ctx context.Context` as first arg.
- No `panic` outside `main()` and `init()`; return errors.
- No global mutable state. DI through constructors.
- `.golangci.yml` is at the repo root; do not relax it.

---

## Test policy

- Every new package gets a `_test.go`. Use the stdlib `testing` package; testify is allowed for assertions but not required.
- Table-driven tests for anything with multiple cases.
- Integration tests live next to unit tests, gated on `//go:build integration`.
- Tests that hit Bedrock are gated on `//go:build live` and an env var (`FSDTRACE_LIVE_TESTS=1`). Default CI does not run them.
- `make smoke` runs the full pipeline against `testdata/sample-spring-app` using a recorded Bedrock cassette (see `internal/embed/cassette.go`).

---

## When to proceed vs ask

**Proceed without asking** for:
- Internal naming, code organization within an existing package, choice between two stdlib options.
- Adding test cases, fixtures, or documentation comments.
- Bug fixes that don't change public API or schema.

**Stop and ask** for:
- Any change to `SPEC.md` (schema, CLI surface, MCP tool catalog, phased plan).
- New external dependency.
- Prompt rewrites (versioning rule applies even if you ask).
- Adding a new artifact kind in the code indexer.
- Anything labelled "hard constraint" above.
- Phase advancement (the user reviews and advances).

When asking, propose one specific option with rationale rather than presenting a menu.

---

## Working agreements

- **Phase discipline.** Work the current phase from `STATUS.md`. Do not get ahead of it. If something blocks the current phase, document the block in `STATUS.md` and ask.
- **Small, reviewable commits.** Per `golang-style` PR guidelines: features ≤400 lines, bugfixes ≤200 lines. Split larger work.
- **Update `STATUS.md`** when you finish a unit of work. Format: date, phase, what was done, what's next, open questions.
- **Don't invent fixtures.** When you need test data, put it in `testdata/` and reference it from tests. No inline 200-line literals.
- **Don't silently retry failing tests.** If something is flaky, mark it `t.Skip` with a TODO and surface it in `STATUS.md`.

---

## External tooling the project depends on

- **`scip-java`** (JVM-based, user-installed). Required only for `fsdtrace index code`. If missing, fail fast with: "scip-java not found on PATH. Install: https://sourcegraph.github.io/scip-java/docs/getting-started.html". Do not attempt to embed scip-java.
- **A Spring Boot project that compiles.** scip-java needs `mvn compile` or `gradle classes` to have run. Document this in error messages.

The MCP server (`fsdtrace mcp serve`) has no external tooling dependencies and must start in <200ms cold.

---

## When the user adds a new Spring annotation to support

The annotation harvester (`internal/code/spring.go`) is the most likely place to be extended. Process:
1. Add the annotation to the SPEC §7.2 table.
2. Add a tree-sitter query for it.
3. Add a fixture file in `testdata/sample-spring-app/` that uses it.
4. Add a test asserting the annotation is harvested.
5. Update the SPEC and STATUS.

Do not add support for new annotations as side-effects of unrelated work.

---

## What this file is not

This file does not duplicate `SPEC.md`. If a working agreement and the SPEC conflict, the SPEC wins and you should ask the user to reconcile.
