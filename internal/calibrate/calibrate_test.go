package calibrate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func fixtures(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	body := strings.Repeat("a", 1000)
	var files []string
	for _, name := range []string{"a.md", "b.md"} {
		p := filepath.Join(dir, name)
		src := "---\nid: x\ntype: concept\nstatus: active\nsummary: s\n---\n" + body
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, p)
	}
	return files
}

func TestRunRatio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages/count_tokens" || r.Header.Get("x-api-key") != "k" || r.Header.Get("anthropic-version") != "2023-06-01" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"input_tokens": 250}`))
	}))
	defer srv.Close()
	res, err := Run(context.Background(), srv.Client(), srv.URL, "k", "m", fixtures(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 2 || res.Chars != 2000 || res.Tokens != 500 || res.CharsPerToken != 4.0 {
		t.Fatalf("unexpected result %+v", res)
	}
	if got := res.ProposedCaps(); got != [3]int{1200, 6000, 12000} {
		t.Fatalf("caps %v", got)
	}
}

func TestRunUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := Run(context.Background(), srv.Client(), srv.URL, "k", "m", fixtures(t))
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want 401 error, got %v", err)
	}
}

func TestRunCountsNotesWithoutFrontmatter(t *testing.T) {
	dir := t.TempDir()
	withFM := filepath.Join(dir, "a.md")
	plain := filepath.Join(dir, "b.md")
	plainSrc := "# Plain note\n\nno frontmatter here \xc3\xa4\xc3\xb6\n"
	if err := os.WriteFile(withFM, []byte("---\nid: x\ntype: concept\nstatus: active\nsummary: s\n---\n"+strings.Repeat("a", 1000)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, []byte(plainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		_, _ = w.Write([]byte(`{"input_tokens": 250}`))
	}))
	defer srv.Close()
	res, err := Run(context.Background(), srv.Client(), srv.URL, "k", "m", []string{withFM, plain})
	if err != nil {
		t.Fatal(err)
	}
	if want := 1000 + utf8.RuneCountInString(plainSrc); res.Notes != 2 || res.Chars != want {
		t.Fatalf("want 2 notes and %d chars, got %+v", want, res)
	}
	if len(bodies) != 2 || !strings.Contains(bodies[1], "no frontmatter here") || !strings.Contains(bodies[1], "# Plain note") {
		t.Fatalf("second request did not carry the whole file: %v", bodies)
	}
}

func TestRunNamesFileWithBrokenFrontmatter(t *testing.T) {
	p := filepath.Join(t.TempDir(), "broken.md")
	if err := os.WriteFile(p, []byte("---\nid: x\nno closing delimiter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"input_tokens": 1}`))
	}))
	defer srv.Close()
	_, err := Run(context.Background(), srv.Client(), srv.URL, "k", "m", []string{p})
	if err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("want error naming %s, got %v", p, err)
	}
}

func TestRunSkipsEmptyNotes(t *testing.T) {
	dir := t.TempDir()
	normal := filepath.Join(dir, "a.md")
	empty := filepath.Join(dir, "b.md")
	emptyBody := filepath.Join(dir, "c.md")
	if err := os.WriteFile(normal, []byte("---\nid: x\ntype: concept\nstatus: active\nsummary: s\n---\n"+strings.Repeat("a", 1000)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(emptyBody, []byte("---\nid: y\ntype: concept\nstatus: active\nsummary: s\n---\n \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"input_tokens": 250}`))
	}))
	defer srv.Close()
	res, err := Run(context.Background(), srv.Client(), srv.URL, "k", "m", []string{normal, empty, emptyBody})
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 1 || res.Chars != 1000 || res.Tokens != 250 || requests != 1 {
		t.Fatalf("want 1 note and 1 request, got %+v with %d requests", res, requests)
	}
}
