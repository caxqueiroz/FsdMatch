package cli

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const githubAPIBaseURL = "https://api.github.com"
const maxGitHubZipEntryBytes int64 = 512 << 20

type githubRepo struct {
	Owner string
	Repo  string
}

func parseGitHubRepoURL(rawURL string) (githubRepo, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return githubRepo{}, fmt.Errorf("parse github url: %w", err)
	}
	if u.Scheme != "https" || strings.ToLower(u.Host) != "github.com" {
		return githubRepo{}, errors.New("github url must use https://github.com/<owner>/<repo>")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return githubRepo{}, errors.New("github url must include owner and repo")
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	if repo == "" {
		return githubRepo{}, errors.New("github url must include repo name")
	}
	return githubRepo{Owner: parts[0], Repo: repo}, nil
}

func downloadGitHubRepo(ctx context.Context, rawURL, ref, dest string) error {
	repo, err := parseGitHubRepoURL(rawURL)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	if ref == "" {
		ref, err = githubDefaultBranch(ctx, client, repo)
		if err != nil {
			return err
		}
	}
	zipPath, err := downloadGitHubZip(ctx, client, repo, ref)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(zipPath) }()

	if err := os.MkdirAll(dest, 0o750); err != nil {
		return fmt.Errorf("create repo dir: %w", err)
	}
	return extractGitHubZip(zipPath, dest)
}

func githubDefaultBranch(ctx context.Context, client *http.Client, repo githubRepo) (string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s",
		githubAPIBaseURL, url.PathEscape(repo.Owner), url.PathEscape(repo.Repo))
	var out struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := githubGetJSON(ctx, client, endpoint, &out); err != nil {
		return "", err
	}
	if out.DefaultBranch == "" {
		return "", errors.New("github response missing default_branch")
	}
	return out.DefaultBranch, nil
}

func downloadGitHubZip(ctx context.Context, client *http.Client, repo githubRepo, ref string) (string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/zipball/%s",
		githubAPIBaseURL, url.PathEscape(repo.Owner), url.PathEscape(repo.Repo), url.PathEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fsdtrace")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download github zip: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("download github zip: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	f, err := os.CreateTemp("", "fsdtrace-github-*.zip")
	if err != nil {
		return "", fmt.Errorf("create temp zip: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("write temp zip: %w", err)
	}
	return f.Name(), nil
}

func githubGetJSON(ctx context.Context, client *http.Client, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fsdtrace")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("github request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("github request: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode github response: %w", err)
	}
	return nil
}

func extractGitHubZip(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open github zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	cleanDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	for _, f := range r.File {
		rel, ok, err := githubZipRelPath(f.Name)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		target := filepath.Join(cleanDest, filepath.FromSlash(rel))
		if !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe zip path %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return fmt.Errorf("create directory %s: %w", target, err)
			}
			continue
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe zip path %q: symlinks are not extracted", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return fmt.Errorf("create parent %s: %w", target, err)
		}
		if err := extractZipFile(f, target); err != nil {
			return err
		}
	}
	return nil
}

func githubZipRelPath(name string) (string, bool, error) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) < 2 {
		return "", false, nil
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return "", false, fmt.Errorf("unsafe zip path %q", name)
		}
	}
	return strings.Join(parts[1:], "/"), true, nil
}

func extractZipFile(f *zip.File, target string) error {
	if f.UncompressedSize64 > uint64(maxGitHubZipEntryBytes) {
		return fmt.Errorf("zip entry %s exceeds %d bytes", f.Name, maxGitHubZipEntryBytes)
	}
	src, err := f.Open()
	if err != nil {
		return fmt.Errorf("open zip entry %s: %w", f.Name, err)
	}
	defer func() { _ = src.Close() }()
	perm := extractedFilePerm(f.FileInfo().Mode().Perm())
	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm) // #nosec G304 -- target is validated by extractGitHubZip before this helper is called.
	if err != nil {
		return fmt.Errorf("create file %s: %w", target, err)
	}
	defer func() { _ = dst.Close() }()
	n, err := io.Copy(dst, io.LimitReader(src, maxGitHubZipEntryBytes+1))
	if err != nil {
		return fmt.Errorf("extract file %s: %w", target, err)
	}
	if n > maxGitHubZipEntryBytes {
		return fmt.Errorf("zip entry %s exceeds %d bytes", f.Name, maxGitHubZipEntryBytes)
	}
	return nil
}

func extractedFilePerm(perm os.FileMode) os.FileMode {
	if perm&0o111 != 0 {
		return 0o700
	}
	return 0o600
}
