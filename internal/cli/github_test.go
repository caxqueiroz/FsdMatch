package cli

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGitHubRepoURLAcceptsRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "plain https",
			rawURL:    "https://github.com/spring-projects/spring-petclinic",
			wantOwner: "spring-projects",
			wantRepo:  "spring-petclinic",
		},
		{
			name:      "git suffix",
			rawURL:    "https://github.com/spring-projects/spring-petclinic.git",
			wantOwner: "spring-projects",
			wantRepo:  "spring-petclinic",
		},
		{
			name:      "extra path ignored",
			rawURL:    "https://github.com/spring-projects/spring-petclinic/tree/main",
			wantOwner: "spring-projects",
			wantRepo:  "spring-petclinic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitHubRepoURL(tt.rawURL)
			if err != nil {
				t.Fatalf("parseGitHubRepoURL: %v", err)
			}
			if got.Owner != tt.wantOwner || got.Repo != tt.wantRepo {
				t.Fatalf("repo = %#v, want owner=%q repo=%q", got, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

func TestParseGitHubRepoURLRejectsUnsupportedURL(t *testing.T) {
	_, err := parseGitHubRepoURL("https://gitlab.com/spring-projects/spring-petclinic")
	if err == nil {
		t.Fatal("expected unsupported host error")
	}
	if !strings.Contains(err.Error(), "github.com") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractGitHubZipStripsArchiveRoot(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "repo.zip")
	writeZip(t, zipPath, map[string]string{
		"spring-petclinic-main/pom.xml":                    "<project />",
		"spring-petclinic-main/src/main/java/App.java":     "class App {}",
		"spring-petclinic-main/src/test/java/AppTest.java": "class AppTest {}",
	})

	outDir := filepath.Join(dir, "repo")
	if err := extractGitHubZip(zipPath, outDir); err != nil {
		t.Fatalf("extractGitHubZip: %v", err)
	}

	for _, rel := range []string{
		"pom.xml",
		"src/main/java/App.java",
		"src/test/java/AppTest.java",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("expected extracted %s: %v", rel, err)
		}
	}
}

func TestExtractGitHubZipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "repo.zip")
	writeZip(t, zipPath, map[string]string{
		"repo-main/../escape.txt": "nope",
	})

	err := extractGitHubZip(zipPath, filepath.Join(dir, "repo"))
	if err == nil {
		t.Fatal("expected traversal error")
	}
	if !strings.Contains(err.Error(), "unsafe zip path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	defer func() { _ = zw.Close() }()
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}
