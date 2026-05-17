package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
	"github.com/cax/fsdtrace/internal/match"
)

const (
	// EnvEmbeddingModel overrides the embedding model.
	EnvEmbeddingModel = "FSDTRACE_EMBEDDING_MODEL"
	// EnvAtomizerModel overrides the Claude model used for FSD atomization.
	EnvAtomizerModel = "FSDTRACE_ATOMIZER_MODEL"
	// EnvJudgmentModel overrides the Claude model used for matching.
	EnvJudgmentModel = "FSDTRACE_JUDGMENT_MODEL"
	// EnvRejudgeModel overrides the stronger Claude model used for drift rejudging.
	EnvRejudgeModel = "FSDTRACE_REJUDGE_MODEL"

	// DefaultRejudgeModel is the built-in fallback when no flag, env, or
	// config file value is provided.
	DefaultRejudgeModel = "anthropic.claude-opus-4-v2:0"
)

type modelKind string

const (
	modelEmbedding modelKind = "embedding"
	modelAtomizer  modelKind = "atomizer"
	modelJudgment  modelKind = "judgment"
	modelRejudge   modelKind = "rejudge"
)

type appConfig struct {
	bedrockBaseURL string
	embeddingModel string
	atomizerModel  string
	judgmentModel  string
	rejudgeModel   string
}

func loadAppConfig(path string, getenv func(string) string) (appConfig, error) {
	cfg := appConfig{}
	if getenv == nil {
		getenv = os.Getenv
	}

	fileCfg, err := readAppConfigFile(path)
	if err != nil {
		return cfg, err
	}
	cfg = fileCfg

	cfg.bedrockBaseURL = firstNonEmpty(getenv(EnvBedrockBaseURL), cfg.bedrockBaseURL)
	cfg.embeddingModel = firstNonEmpty(getenv(EnvEmbeddingModel), cfg.embeddingModel)
	cfg.atomizerModel = firstNonEmpty(getenv(EnvAtomizerModel), cfg.atomizerModel)
	cfg.judgmentModel = firstNonEmpty(getenv(EnvJudgmentModel), cfg.judgmentModel)
	cfg.rejudgeModel = firstNonEmpty(getenv(EnvRejudgeModel), cfg.rejudgeModel)
	return cfg, nil
}

func (c appConfig) model(kind modelKind, flagValue string, flagChanged bool) string {
	if flagChanged && strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	switch kind {
	case modelEmbedding:
		return firstNonEmpty(c.embeddingModel, embed.TitanModelID)
	case modelAtomizer:
		return firstNonEmpty(c.atomizerModel, fsd.DefaultAtomizerModel)
	case modelJudgment:
		return firstNonEmpty(c.judgmentModel, match.DefaultJudgmentModel)
	case modelRejudge:
		return firstNonEmpty(c.rejudgeModel, DefaultRejudgeModel)
	default:
		return ""
	}
}

func (c appConfig) bedrockURL() string {
	return strings.TrimSpace(c.bedrockBaseURL)
}

func readAppConfigFile(path string) (appConfig, error) {
	if path != "" {
		return parseConfigPath(path, true)
	}
	for _, p := range defaultConfigPaths() {
		cfg, err := parseConfigPath(p, false)
		if err == nil {
			return cfg, nil
		}
		if !os.IsNotExist(err) {
			return appConfig{}, err
		}
	}
	return appConfig{}, nil
}

func defaultConfigPaths() []string {
	paths := []string{"fsdtrace.yaml"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths, filepath.Join(home, ".fsdtrace.yaml"))
	}
	return paths
}

func parseConfigPath(path string, required bool) (appConfig, error) {
	f, err := os.Open(path) // #nosec G304 -- user-selected config path
	if err != nil {
		if !required && os.IsNotExist(err) {
			return appConfig{}, err
		}
		return appConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var cfg appConfig
	section := ""
	scanner := bufio.NewScanner(f)
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := stripComment(scanner.Text())
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingSpaces(line)
		key, value, ok := splitConfigLine(strings.TrimSpace(line))
		if !ok {
			return cfg, fmt.Errorf("parse config %s:%d: expected key: value", path, lineNum)
		}
		if indent == 0 && value == "" {
			section = key
			continue
		}
		fullKey := key
		if indent > 0 && section != "" {
			fullKey = section + "." + key
		}
		setConfigValue(&cfg, fullKey, value)
	}
	if err := scanner.Err(); err != nil {
		return cfg, fmt.Errorf("scan config %s: %w", path, err)
	}
	return cfg, nil
}

func stripComment(s string) string {
	if i := strings.Index(s, "#"); i >= 0 {
		return s[:i]
	}
	return s
}

func leadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

func splitConfigLine(s string) (key, value string, ok bool) {
	k, v, found := strings.Cut(s, ":")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(k)
	value = strings.TrimSpace(v)
	value = strings.Trim(value, `"'`)
	return key, value, key != ""
}

func setConfigValue(cfg *appConfig, key, value string) {
	value = strings.TrimSpace(value)
	switch key {
	case "bedrock.base_url":
		cfg.bedrockBaseURL = value
	case "bedrock.embedding_model", "embedding_model":
		cfg.embeddingModel = value
	case "bedrock.atomizer_model", "atomizer_model":
		cfg.atomizerModel = value
	case "bedrock.judgment_model", "judgment_model":
		cfg.judgmentModel = value
	case "bedrock.rejudge_model", "rejudge_model":
		cfg.rejudgeModel = value
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
