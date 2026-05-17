package cli

import (
	"strings"
	"testing"
)

func TestIngestResumeRequiresRunIDBeforeProviderSetup(t *testing.T) {
	_, err := executeRoot(t,
		"ingest", "fsd", "../../testdata/fsd-sample.md",
		"--resume",
	)
	if err == nil {
		t.Fatal("expected --resume without --run-id to fail")
	}
	if !strings.Contains(err.Error(), "--resume requires --run-id") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), EnvBedrockBaseURL) {
		t.Fatalf("resume validation should run before provider setup: %v", err)
	}
}

func TestIndexResumeRequiresRunIDBeforeProviderSetup(t *testing.T) {
	_, err := executeRoot(t,
		"index", "code", "../../testdata/sample-spring-app",
		"--resume",
	)
	if err == nil {
		t.Fatal("expected --resume without --run-id to fail")
	}
	if !strings.Contains(err.Error(), "--resume requires --run-id") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), EnvBedrockBaseURL) {
		t.Fatalf("resume validation should run before provider setup: %v", err)
	}
}

func TestMatchResumeRequiresRunIDBeforeProviderSetup(t *testing.T) {
	_, err := executeRoot(t, "match", "--resume")
	if err == nil {
		t.Fatal("expected --resume without --run-id to fail")
	}
	if !strings.Contains(err.Error(), "--resume requires --run-id") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), EnvBedrockBaseURL) {
		t.Fatalf("resume validation should run before provider setup: %v", err)
	}
}

func TestTraceGithubResumeRequiresStableCheckoutDir(t *testing.T) {
	_, err := executeRoot(t,
		"--run-id", "trace-resume",
		"trace", "github", "https://github.com/spring-projects/spring-petclinic",
		"--fsd", "examples/petclinic/fsd.md",
		"--resume",
	)
	if err == nil {
		t.Fatal("expected trace github --resume without --checkout-dir to fail")
	}
	if !strings.Contains(err.Error(), "--resume requires --checkout-dir") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), EnvBedrockBaseURL) {
		t.Fatalf("resume validation should run before provider setup: %v", err)
	}
}
