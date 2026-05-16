-- Migration 001: initial schema. Mirrors internal/db/schema.sql.
-- Migrations are forward-only and numbered.
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS features (
  id           TEXT PRIMARY KEY,
  title        TEXT NOT NULL,
  description  TEXT NOT NULL,
  acceptance   JSON NOT NULL,
  actor        TEXT,
  inputs       JSON,
  outputs      JSON,
  side_effects JSON,
  non_functional JSON,
  fsd_section  TEXT,
  fsd_anchor   TEXT,
  run_id       TEXT NOT NULL,
  created_at   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_features_section ON features(fsd_section);

CREATE TABLE IF NOT EXISTS code_artifacts (
  id           INTEGER PRIMARY KEY,
  kind         TEXT NOT NULL,
  identifier   TEXT NOT NULL,
  scip_symbol  TEXT,
  package      TEXT,
  class        TEXT,
  method       TEXT,
  file         TEXT NOT NULL,
  start_line   INTEGER NOT NULL,
  end_line     INTEGER NOT NULL,
  signature    TEXT,
  annotations  JSON,
  source       TEXT,
  run_id       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_artifacts_kind ON code_artifacts(kind);
CREATE INDEX IF NOT EXISTS idx_artifacts_symbol ON code_artifacts(scip_symbol);

CREATE TABLE IF NOT EXISTS relationships (
  from_artifact INTEGER NOT NULL REFERENCES code_artifacts(id),
  to_artifact   INTEGER NOT NULL REFERENCES code_artifacts(id),
  kind          TEXT NOT NULL,
  PRIMARY KEY (from_artifact, to_artifact, kind)
);

CREATE TABLE IF NOT EXISTS tests (
  id              INTEGER PRIMARY KEY,
  name            TEXT NOT NULL,
  file            TEXT NOT NULL,
  line            INTEGER NOT NULL,
  test_kind       TEXT,
  target_artifact INTEGER REFERENCES code_artifacts(id),
  asserts         JSON,
  run_id          TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tests_target ON tests(target_artifact);

CREATE TABLE IF NOT EXISTS matches (
  run_id        TEXT NOT NULL,
  feature_id    TEXT NOT NULL REFERENCES features(id),
  artifact_id   INTEGER NOT NULL REFERENCES code_artifacts(id),
  verdict       TEXT NOT NULL,
  confidence    REAL NOT NULL,
  evidence      JSON NOT NULL,
  notes         TEXT,
  model         TEXT NOT NULL,
  prompt_version TEXT NOT NULL,
  PRIMARY KEY (run_id, feature_id, artifact_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_feature ON matches(feature_id);
CREATE INDEX IF NOT EXISTS idx_matches_verdict ON matches(verdict);

CREATE TABLE IF NOT EXISTS runs (
  id           TEXT PRIMARY KEY,
  started_at   INTEGER NOT NULL,
  finished_at  INTEGER,
  summary      JSON,
  config       JSON
);

CREATE TABLE IF NOT EXISTS embedding_cache (
  key          TEXT PRIMARY KEY,
  model        TEXT NOT NULL,
  dim          INTEGER NOT NULL,
  embedding    BLOB NOT NULL,
  created_at   INTEGER NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS feature_vec USING vec0(
  embedding float[1024]
);

CREATE VIRTUAL TABLE IF NOT EXISTS artifact_vec USING vec0(
  embedding float[1024]
);
