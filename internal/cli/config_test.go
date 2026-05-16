package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
	"github.com/cax/fsdtrace/internal/match"
)

func TestModelConfigPrecedenceFlagEnvFileDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "fsdtrace.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
bedrock:
  embedding_model: file-embed
  atomizer_model: file-atomizer
  judgment_model: file-judge
  rejudge_model: file-rejudge
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadAppConfig(cfgPath, func(k string) string {
		switch k {
		case EnvEmbeddingModel:
			return "env-embed"
		case EnvAtomizerModel:
			return "env-atomizer"
		case EnvJudgmentModel:
			return "env-judge"
		case EnvRejudgeModel:
			return "env-rejudge"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.model(modelEmbedding, "flag-embed", true); got != "flag-embed" {
		t.Fatalf("flag precedence got %q", got)
	}
	if got := cfg.model(modelEmbedding, "", false); got != "env-embed" {
		t.Fatalf("env precedence got %q", got)
	}
	if got := cfg.model(modelAtomizer, "", false); got != "env-atomizer" {
		t.Fatalf("atomizer env got %q", got)
	}
	if got := cfg.model(modelJudgment, "", false); got != "env-judge" {
		t.Fatalf("judgment env got %q", got)
	}
	if got := cfg.model(modelRejudge, "", false); got != "env-rejudge" {
		t.Fatalf("rejudge env got %q", got)
	}
}

func TestModelConfigFallsBackToFileThenDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "fsdtrace.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
bedrock:
  embedding_model: file-embed
  atomizer_model: file-atomizer
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadAppConfig(cfgPath, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.model(modelEmbedding, "", false); got != "file-embed" {
		t.Fatalf("file embedding got %q", got)
	}
	if got := cfg.model(modelAtomizer, "", false); got != "file-atomizer" {
		t.Fatalf("file atomizer got %q", got)
	}
	if got := cfg.model(modelJudgment, "", false); got != match.DefaultJudgmentModel {
		t.Fatalf("default judgment got %q", got)
	}
	if got := cfg.model(modelRejudge, "", false); got != DefaultRejudgeModel {
		t.Fatalf("default rejudge got %q", got)
	}
}

func TestLoadAppConfigSearchesDefaultLocations(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("fsdtrace.yaml", []byte(`
bedrock:
  embedding_model: local-embed
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadAppConfig("", func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.model(modelEmbedding, "", false); got != "local-embed" {
		t.Fatalf("default config search got %q", got)
	}
	if got := cfg.model(modelAtomizer, "", false); got != fsd.DefaultAtomizerModel {
		t.Fatalf("atomizer default got %q", got)
	}
	if got := cfg.model(modelJudgment, "", false); got != match.DefaultJudgmentModel {
		t.Fatalf("judgment default got %q", got)
	}
	if got := cfg.model(modelEmbedding, "", false); got == embed.TitanModelID {
		t.Fatalf("expected file value to override default")
	}
}
