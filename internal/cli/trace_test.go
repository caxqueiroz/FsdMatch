package cli

import (
	"strings"
	"testing"
)

func TestTraceGithubRequiresFSDFlag(t *testing.T) {
	_, err := executeRoot(t, "trace", "github", "https://github.com/spring-projects/spring-petclinic")
	if err == nil {
		t.Fatal("expected missing --fsd error")
	}
	if !strings.Contains(err.Error(), "--fsd is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), EnvBedrockBaseURL) {
		t.Fatalf("missing --fsd should fail before Bedrock setup: %v", err)
	}
}

func TestTraceGithubRejectsNonGitHubURL(t *testing.T) {
	_, err := executeRoot(t,
		"trace", "github", "https://gitlab.com/spring-projects/spring-petclinic",
		"--fsd", "examples/petclinic/fsd.md",
	)
	if err == nil {
		t.Fatal("expected unsupported URL error")
	}
	if !strings.Contains(err.Error(), "github.com") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTraceGithubRejectsMissingFSDPathBeforeDownload(t *testing.T) {
	_, err := executeRoot(t,
		"trace", "github", "https://github.com/spring-projects/spring-petclinic",
		"--fsd", "/definitely/missing/fsd.md",
	)
	if err == nil {
		t.Fatal("expected missing FSD error")
	}
	if !strings.Contains(err.Error(), "stat fsd") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), EnvBedrockBaseURL) {
		t.Fatalf("missing FSD should fail before Bedrock setup: %v", err)
	}
}
