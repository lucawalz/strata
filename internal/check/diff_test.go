package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type gitRepo struct {
	t    *testing.T
	dir  string
	home string
}

func (r *gitRepo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(r.home, "gitconfig"), "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *gitRepo) write(name, content string) {
	r.t.Helper()
	p := filepath.Join(r.dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *gitRepo) commit(sign bool, files ...string) string {
	r.t.Helper()
	r.git(append([]string{"add", "-f", "--"}, files...)...)
	flag := "--no-gpg-sign"
	if sign {
		flag = "-S"
	}
	r.git("commit", "-q", flag, "-m", "c")
	return r.git("rev-parse", "HEAD")
}

func newGitRepo(t *testing.T) (*gitRepo, string) {
	t.Helper()
	r := &gitRepo{t: t, dir: t.TempDir(), home: t.TempDir()}
	key := filepath.Join(t.TempDir(), "k")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v\n%s", err, out)
	}
	pub, _ := os.ReadFile(key + ".pub")
	cfg := "[user]\n\tname = T\n\temail = t@example.com\n\tsigningkey = " + key + "\n[gpg]\n\tformat = ssh\n[commit]\n\tgpgsign = false\n"
	if err := os.WriteFile(filepath.Join(r.home, "gitconfig"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	r.git("init", "-q", "-b", "main")
	return r, strings.TrimSpace(string(pub))
}

func diffWants(t *testing.T, r *gitRepo, base, head string) []Finding {
	t.Helper()
	got, err := Diff(Repo{Root: r.dir, Config: testConfig()}, base, head)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func has(fs []Finding, check string, level Level, path string) bool {
	for _, f := range fs {
		if f.Check == check && f.Level == level && (path == "" || f.Path == path) {
			return true
		}
	}
	return false
}

func TestDiffRawAndAppend(t *testing.T) {
	r, _ := newGitRepo(t)
	r.write("raw/a.txt", "one\n")
	r.write("raw/b.txt", "two\n")
	r.write("raw/c.txt", "three\n")
	r.write("notes/x.md", "line one\nline two\n")
	base := r.commit(false, "raw", "notes")

	r.write("raw/a.txt", "changed\n")
	r.git("rm", "-q", "raw/b.txt")
	r.git("mv", "raw/c.txt", "raw/d.txt")
	r.write("raw/new.txt", "new\n")
	r.write("notes/x.md", "line one\nline two\nline three\n")
	head := r.commit(false, "raw", "notes")

	got := diffWants(t, r, base, head)
	for _, p := range []string{"raw/a.txt", "raw/b.txt", "raw/d.txt"} {
		if !has(got, "raw", Error, p) {
			t.Errorf("missing raw error for %s: %v", p, got)
		}
	}
	if has(got, "raw", Error, "raw/new.txt") {
		t.Errorf("added raw file flagged: %v", got)
	}
	if !has(got, "append", Warning, "notes/x.md") {
		t.Errorf("missing append warning: %v", got)
	}
}

func TestDiffRawRenames(t *testing.T) {
	r, _ := newGitRepo(t)
	r.write("inbox/x.md", "item\n")
	r.write("raw/y.txt", "source\n")
	base := r.commit(false, "inbox", "raw")
	if err := os.MkdirAll(filepath.Join(r.dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.git("mv", "inbox/x.md", "raw/x.md")
	r.git("mv", "raw/y.txt", "notes/y.txt")
	head := r.commit(false, "raw", "notes")
	got := diffWants(t, r, base, head)
	if has(got, "raw", Error, "raw/x.md") {
		t.Errorf("rename into raw flagged: %v", got)
	}
	if !has(got, "raw", Error, "notes/y.txt") {
		t.Errorf("rename out of raw not flagged: %v", got)
	}
}

func TestDiffGovernance(t *testing.T) {
	r, pub := newGitRepo(t)
	r.write(".github/allowed_signers", "t@example.com "+pub+"\n")
	r.write("AGENTS.md", "rules\n")
	base := r.commit(false, ".github", "AGENTS.md")

	r.write("AGENTS.md", "rules v2\n")
	signed := r.commit(true, "AGENTS.md")
	got := diffWants(t, r, base, signed)
	if !has(got, "governance-summary", Info, "") || has(got, "governance", Error, "") {
		t.Errorf("signed: %v", got)
	}
	if !strings.Contains(got[0].Message, "AGENTS.md") {
		t.Errorf("summary: %v", got)
	}

	r.write("AGENTS.md", "rules v3\n")
	unsigned := r.commit(false, "AGENTS.md")
	got = diffWants(t, r, base, unsigned)
	found := false
	for _, f := range got {
		if f.Check == "governance" && f.Level == Error && strings.Contains(f.Message, unsigned[:12]) {
			found = true
		}
	}
	if !found {
		t.Errorf("unsigned commit not flagged: %v", got)
	}
}

func TestDiffGovernanceEmptySigners(t *testing.T) {
	r, _ := newGitRepo(t)
	r.write(".github/allowed_signers", "")
	r.write("AGENTS.md", "rules\n")
	base := r.commit(false, ".github", "AGENTS.md")
	r.write("AGENTS.md", "rules v2\n")
	head := r.commit(true, "AGENTS.md")
	got := diffWants(t, r, base, head)
	for _, f := range got {
		if f.Check == "governance" && f.Level == Error && strings.Contains(f.Message, "allowed_signers is empty") {
			return
		}
	}
	t.Errorf("got %v", got)
}

func TestDiffGovernanceIsolation(t *testing.T) {
	r, pub := newGitRepo(t)
	r.write(".github/allowed_signers", "t@example.com "+pub+"\n")
	r.write("AGENTS.md", "rules\n")
	base := r.commit(false, ".github", "AGENTS.md")
	r.write("AGENTS.md", "rules v2\n")
	r.write("notes/x.md", "text\n")
	head := r.commit(true, "AGENTS.md", "notes")
	if got := diffWants(t, r, base, head); !has(got, "governance-isolation", Error, "") {
		t.Errorf("got %v", got)
	}
}

func TestDiffInjectAndPII(t *testing.T) {
	r, _ := newGitRepo(t)
	r.write("notes/x.md", "text\n")
	base := r.commit(false, "notes")
	r.write("notes/x.md", "text \u202e hidden\n")
	r.write("docs/y.txt", "pay DE89370400440532013000 now\n")
	r.write("raw/z.txt", "pay DE89370400440532013000 now\n")
	head := r.commit(false, "notes", "docs", "raw")
	got := diffWants(t, r, base, head)
	if !has(got, "inject", Error, "notes/x.md") || !has(got, "pii", Error, "docs/y.txt") || has(got, "pii", Error, "raw/z.txt") {
		t.Errorf("got %v", got)
	}
}

func (r *gitRepo) useKey(key string) {
	r.t.Helper()
	r.git("config", "--file", filepath.Join(r.home, "gitconfig"), "user.signingkey", key)
}

func newKey(t *testing.T) (string, string) {
	t.Helper()
	key := filepath.Join(t.TempDir(), "k")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v\n%s", err, out)
	}
	pub, _ := os.ReadFile(key + ".pub")
	return key, strings.TrimSpace(string(pub))
}

func hasMsg(fs []Finding, check string, level Level, sub string) bool {
	for _, f := range fs {
		if f.Check == check && f.Level == level && strings.Contains(f.Message, sub) {
			return true
		}
	}
	return false
}

func governedBase(t *testing.T) (*gitRepo, string, string) {
	r, pub := newGitRepo(t)
	r.write(".github/allowed_signers", "t@example.com "+pub+"\n")
	r.write("AGENTS.md", "rules\n")
	return r, r.commit(false, ".github", "AGENTS.md"), pub
}

func TestDiffGovernanceRename(t *testing.T) {
	r, base, _ := governedBase(t)
	r.git("mv", "AGENTS.md", "notes-a.md")
	if err := os.MkdirAll(filepath.Join(r.dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.git("mv", "notes-a.md", "notes/a.md")
	head := r.commit(false, "notes")
	got := diffWants(t, r, base, head)
	if !has(got, "governance", Error, "") || !has(got, "governance-isolation", Error, "") {
		t.Errorf("got %v", got)
	}
}

func TestDiffGovernanceMergeRollback(t *testing.T) {
	r, pub := newGitRepo(t)
	r.write(".github/allowed_signers", "t@example.com "+pub+"\n")
	r.write("AGENTS.md", "rules v1\n")
	old := r.commit(false, ".github", "AGENTS.md")
	r.write("AGENTS.md", "rules v2\n")
	base := r.commit(true, "AGENTS.md")
	r.git("checkout", "-q", "-b", "side", old)
	r.write("x.txt", "x\n")
	r.commit(false, "x.txt")
	r.git("merge", "-q", "--no-gpg-sign", "-s", "ours", base, "-m", "m")
	head := r.git("rev-parse", "HEAD")
	got := diffWants(t, r, base, head)
	if !has(got, "governance", Error, "") {
		t.Errorf("got %v", got)
	}
}

func TestDiffGovernedPathVariants(t *testing.T) {
	cases := []struct {
		path    string
		entries []string
		want    bool
	}{
		{"Claude.md", []string{"CLAUDE.md"}, true},
		{".Claude/settings.json", []string{".claude/"}, true},
		{".github/workflows/x.yml", []string{".github"}, true},
		{".claude", []string{".claude/"}, true},
		{".github/workflows/x.yml", []string{".github/"}, true},
		{".githubx/a", []string{".github"}, false},
		{"notes/a.md", []string{".github/", "AGENTS.md"}, false},
	}
	for _, c := range cases {
		if got := governed(c.path, c.entries); got != c.want {
			t.Errorf("governed(%q, %v) = %v", c.path, c.entries, got)
		}
	}
}

func TestDiffGovernanceHeadKeySwap(t *testing.T) {
	r, base, _ := governedBase(t)
	key2, pub2 := newKey(t)
	r.write(".github/allowed_signers", "t@example.com "+pub2+"\n")
	r.write("AGENTS.md", "rules v2\n")
	r.useKey(key2)
	head := r.commit(true, ".github", "AGENTS.md")
	got := diffWants(t, r, base, head)
	if !hasMsg(got, "governance", Error, head[:12]) {
		t.Errorf("got %v", got)
	}
}

func TestDiffMovedBase(t *testing.T) {
	r, pub := newGitRepo(t)
	r.write(".github/allowed_signers", "t@example.com "+pub+"\n")
	r.write("AGENTS.md", "rules\n")
	r.write("notes/x.md", "text\n")
	fork := r.commit(false, ".github", "AGENTS.md", "notes")
	r.git("checkout", "-q", "-b", "pr")
	r.write("notes/y.md", "other\n")
	head := r.commit(false, "notes")
	r.git("checkout", "-q", "main")
	r.write("raw/new.txt", "new\n")
	r.write("AGENTS.md", "rules v2\n")
	moved := r.commit(true, "raw", "AGENTS.md")
	_ = fork
	got := diffWants(t, r, moved, head)
	for _, f := range got {
		if f.Check == "raw" || f.Check == "governance-isolation" || f.Check == "governance" || f.Check == "governance-summary" {
			t.Errorf("unexpected %v in %v", f, got)
		}
	}
}
