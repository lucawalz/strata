package calibrate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
