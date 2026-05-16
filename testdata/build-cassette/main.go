// build-cassette records Bedrock cassettes by running real fsdtrace
// pipelines against a fake-Bedrock httptest server. Recording from the
// production code path guarantees byte-identical replay against the
// same build.
//
// Subcommands:
//
//	go run ./testdata/build-cassette fsd   --fsd testdata/fsd-sample.md      --out testdata/fsd-sample.cassette.json
//	go run ./testdata/build-cassette code  --repo testdata/sample-spring-app --out testdata/sample-spring-app.cassette.json
//	go run ./testdata/build-cassette match --fsd testdata/fsd-sample.md --repo testdata/sample-spring-app --out testdata/match.cassette.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cax/fsdtrace/internal/code"
	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
	"github.com/cax/fsdtrace/internal/match"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s {fsd|code|match} [flags]", os.Args[0])
	}
	switch os.Args[1] {
	case "fsd":
		recordFSD(os.Args[2:])
	case "code":
		recordCode(os.Args[2:])
	case "match":
		recordMatch(os.Args[2:])
	default:
		log.Fatalf("unknown subcommand %q (want fsd|code|match)", os.Args[1])
	}
}

func recordMatch(args []string) {
	fs := flag.NewFlagSet("match", flag.ExitOnError)
	fsdPath := fs.String("fsd", "testdata/fsd-sample.md", "FSD markdown")
	repo := fs.String("repo", "testdata/sample-spring-app", "Java repo to index")
	outPath := fs.String("out", "testdata/match.cassette.json", "output cassette path")
	_ = fs.Parse(args)

	repoAbs, err := filepath.Abs(*repo)
	if err != nil {
		log.Fatal(err)
	}

	ts := httptest.NewServer(fakeBedrockHandler())
	defer ts.Close()

	cas := embed.NewEmptyCassette()
	bedrock := bedrockWithRecorder(ts.URL, cas)
	d, cleanup := tempDB()
	defer cleanup()

	ctx := context.Background()
	emb := embed.NewTitanEmbedder(bedrock, embed.TitanModelID)

	// Stage 1: ingest FSD so features + feature_vec are populated.
	chunks, err := fsd.ParseFile(*fsdPath, "")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := fsd.NewAtomizer(d, bedrock, emb).Ingest(ctx, chunks, "build-cassette"); err != nil {
		log.Fatal(err)
	}

	// Stage 2: index code so artifacts + artifact_vec are populated.
	if _, err := code.NewIndexer(d, emb).Index(ctx, repoAbs, "", "build-cassette"); err != nil {
		log.Fatal(err)
	}

	// Stage 3: run the matcher. Every Bedrock call lands in the cassette.
	pipe := match.NewPipeline(d, bedrock, emb)
	if _, err := pipe.MatchAll(ctx, "build-cassette", nil); err != nil {
		log.Fatal(err)
	}

	if err := cas.Save(*outPath); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("recorded match cassette to %s\n", *outPath)
}

func recordFSD(args []string) {
	fs := flag.NewFlagSet("fsd", flag.ExitOnError)
	fsdPath := fs.String("fsd", "testdata/fsd-sample.md", "FSD markdown")
	outPath := fs.String("out", "testdata/fsd-sample.cassette.json", "output cassette path")
	_ = fs.Parse(args)

	chunks, err := fsd.ParseFile(*fsdPath, "")
	if err != nil {
		log.Fatal(err)
	}

	ts := httptest.NewServer(fakeBedrockHandler())
	defer ts.Close()

	cas := embed.NewEmptyCassette()
	bedrock := bedrockWithRecorder(ts.URL, cas)
	d, cleanup := tempDB()
	defer cleanup()

	emb := embed.NewTitanEmbedder(bedrock, embed.TitanModelID)
	atomizer := fsd.NewAtomizer(d, bedrock, emb)
	if _, err := atomizer.Ingest(context.Background(), chunks, "build-cassette"); err != nil {
		log.Fatal(err)
	}
	if err := cas.Save(*outPath); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("recorded fsd cassette to %s\n", *outPath)
}

func recordCode(args []string) {
	fs := flag.NewFlagSet("code", flag.ExitOnError)
	repo := fs.String("repo", "testdata/sample-spring-app", "Java repo to index")
	outPath := fs.String("out", "testdata/sample-spring-app.cassette.json", "output cassette path")
	_ = fs.Parse(args)

	repoAbs, err := filepath.Abs(*repo)
	if err != nil {
		log.Fatal(err)
	}

	ts := httptest.NewServer(fakeBedrockHandler())
	defer ts.Close()

	cas := embed.NewEmptyCassette()
	bedrock := bedrockWithRecorder(ts.URL, cas)
	d, cleanup := tempDB()
	defer cleanup()

	emb := embed.NewTitanEmbedder(bedrock, embed.TitanModelID)
	indexer := code.NewIndexer(d, emb)
	if _, err := indexer.Index(context.Background(), repoAbs, "", "build-cassette"); err != nil {
		log.Fatal(err)
	}
	if err := cas.Save(*outPath); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("recorded code cassette to %s\n", *outPath)
}

func bedrockWithRecorder(baseURL string, cas *embed.Cassette) *embed.BedrockClient {
	rec := &embed.RecordingTransport{Inner: http.DefaultTransport, Cassette: cas}
	bedrock, err := embed.NewClient(baseURL, embed.WithHTTPClient(&http.Client{Transport: rec}))
	if err != nil {
		log.Fatal(err)
	}
	return bedrock
}

func tempDB() (*db.DB, func()) {
	ctx := context.Background()
	tmp, err := os.MkdirTemp("", "build-cassette-*")
	if err != nil {
		log.Fatal(err)
	}
	d, err := db.Open(ctx, filepath.Join(tmp, "ing.db"))
	if err != nil {
		log.Fatal(err)
	}
	if err := d.ApplySchema(ctx); err != nil {
		log.Fatal(err)
	}
	cleanup := func() {
		_ = d.Close()
		_ = os.RemoveAll(tmp)
	}
	return d, cleanup
}

func fakeBedrockHandler() http.Handler {
	anchorRe := regexp.MustCompile(`Anchor:\s*([A-Z]+-\d+)`)
	candidateRe := regexp.MustCompile(`\[(\d+)\] kind=(\w+) identifier="([^"]+)" file=([^:]+):(\d+)-(\d+)`)
	frTitleRe := regexp.MustCompile(`FR:\s*([A-Z]+-\d+)\s*—\s*(.*)`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/model/"+embed.TitanModelID+"/invoke"):
			v := make([]float32, db.EmbeddingDim)
			v[0] = 1
			payload, _ := json.Marshal(embed.TitanEmbedResponse{Embedding: v, InputTextTokenCount: 1})
			_, _ = w.Write(payload)
		case strings.Contains(r.URL.Path, "/invoke"):
			// Decode the user message so escaped quotes resolve.
			var env struct {
				Messages []struct {
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.Unmarshal(raw, &env)
			user := ""
			if len(env.Messages) > 0 {
				user = env.Messages[0].Content
			}

			// Branch: matcher (has "[N] kind=...") vs atomizer (has "Anchor: FR-NNN").
			if cands := candidateRe.FindAllStringSubmatch(user, -1); len(cands) > 0 {
				replyJudgment(w, frTitleRe, user, cands)
				return
			}
			anchor := "FR-000"
			if m := anchorRe.FindStringSubmatch(user); len(m) > 1 {
				anchor = m[1]
			}
			f := fsd.Feature{
				ID:            anchor,
				Title:         "Title for " + anchor,
				Description:   "Description for " + anchor,
				Acceptance:    []string{"criterion 1", "criterion 2"},
				Inputs:        []string{"input"},
				Outputs:       []string{"output"},
				SideEffects:   []string{},
				NonFunctional: []string{},
			}
			frJSON, _ := json.Marshal(f)
			resp := map[string]any{
				"content":     []map[string]any{{"type": "text", "text": string(frJSON)}},
				"stop_reason": "end_turn",
			}
			payload, _ := json.Marshal(resp)
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	})
}

// replyJudgment composes a deterministic judgment array for a matcher
// request body. A candidate becomes "implements" when its identifier
// shares a path segment with one in the FR description; otherwise
// "unrelated". Always cites real file:line evidence so the §7.4 rule
// doesn't fire spuriously in cassette playback.
func replyJudgment(w http.ResponseWriter, frTitleRe *regexp.Regexp, user string, cands [][]string) {
	frID := ""
	if m := frTitleRe.FindStringSubmatch(user); len(m) > 0 {
		frID = m[1]
	}
	// Only the FR-side portion of the prompt counts when judging
	// path/topic overlap; the candidate listing trivially includes
	// the very identifiers we'd otherwise "find".
	frBody := user
	if i := strings.Index(user, "Candidate artifacts:"); i > 0 {
		frBody = user[:i]
	}

	type entry struct {
		ArtifactID int64               `json:"artifact_id"`
		Verdict    string              `json:"verdict"`
		Confidence float64             `json:"confidence"`
		Evidence   []map[string]any    `json:"evidence"`
		Notes      string              `json:"notes,omitempty"`
	}
	out := make([]entry, 0, len(cands))
	for _, c := range cands {
		id := atoi64(c[1])
		kind := c[2]
		ident := c[3]
		file := c[4]
		start := atoi(c[5])
		end := atoi(c[6])

		verdict := match.VerdictUnrelated
		conf := 0.0
		var ev []map[string]any
		if kind == "rest_endpoint" && pathOverlapInBody(ident, frBody) {
			verdict = match.VerdictImplements
			conf = 0.85
			ev = []map[string]any{
				{"file": file, "start": start, "end": end,
					"note": fmt.Sprintf("identifier %q overlaps FR text", ident)},
			}
		} else if kind == "kafka_listener" && strings.Contains(frBody, "Kafka") {
			verdict = match.VerdictImplements
			conf = 0.7
			ev = []map[string]any{{"file": file, "start": start, "end": end,
				"note": "kafka listener referenced in FR"}}
		} else if kind == "scheduled_job" && strings.Contains(strings.ToLower(frBody), "scheduled") {
			verdict = match.VerdictImplements
			conf = 0.8
			ev = []map[string]any{{"file": file, "start": start, "end": end,
				"note": "scheduled job referenced in FR"}}
		}
		out = append(out, entry{ArtifactID: id, Verdict: verdict, Confidence: conf, Evidence: ev,
			Notes: fmt.Sprintf("FR=%s candidate=%s", frID, ident)})
	}
	body, _ := json.Marshal(out)
	resp := map[string]any{
		"content":     []map[string]any{{"type": "text", "text": string(body)}},
		"stop_reason": "end_turn",
	}
	payload, _ := json.Marshal(resp)
	_, _ = w.Write(payload)
}

// pathOverlapInBody is true when the URL path component of ident
// (e.g. "POST /api/v1/notes" → "/api/v1/notes") appears anywhere in
// frBody.
func pathOverlapInBody(ident, frBody string) bool {
	parts := strings.SplitN(ident, " ", 2)
	if len(parts) != 2 {
		return false
	}
	path := parts[1]
	// Strip any "{id}" suffix so /api/v1/notes/{id} also matches the
	// FR's bare /api/v1/notes mention.
	stripped := path
	if i := strings.Index(stripped, "/{"); i > 0 {
		stripped = stripped[:i]
	}
	return strings.Contains(frBody, path) || strings.Contains(frBody, stripped)
}

func atoi64(s string) int64 {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

func atoi(s string) int { return int(atoi64(s)) }
