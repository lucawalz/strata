package gitx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type repo struct {
	t    *testing.T
	dir  string
	home string
}

func (r *repo) git(args ...string) string {
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

func (r *repo) write(name, content string) {
	r.t.Helper()
	p := filepath.Join(r.dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *repo) commit(msg string) string {
	r.t.Helper()
	r.git("add", "-A")
	r.git("commit", "-q", "-m", msg)
	return r.git("rev-parse", "HEAD")
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	r := &repo{t: t, dir: t.TempDir(), home: t.TempDir()}
	r.write("../"+filepath.Base(r.home)+"/gitconfig", "[user]\n\tname = Test\n\temail = test@example.com\n[commit]\n\tgpgsign = false\n")
	r.git("init", "-q", "-b", "main")
	return r
}

func keygen(t *testing.T, name string) (keyPath, pub string) {
	t.Helper()
	keyPath = filepath.Join(t.TempDir(), name)
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", keyPath).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v\n%s", err, out)
	}
	b, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	return keyPath, strings.TrimSpace(string(b))
}

func TestChanges(t *testing.T) {
	r := newRepo(t)
	r.write("a.md", "alpha line one\nalpha line two\nalpha line three\n")
	r.write("b.md", "bravo\n")
	r.write("c.md", "charlie\n")
	r.write("move.md", "a note that will be moved into the archive\nsecond line\nthird line\n")
	base := r.commit("base")
	r.write("a.md", "alpha line one\nalpha line two\nalpha line three\nmore\n")
	r.write("new.md", "new\n")
	if err := os.Remove(filepath.Join(r.dir, "c.md")); err != nil {
		t.Fatal(err)
	}
	r.git("mv", "move.md", "archive-move.md")
	r.write("archive/x", "")
	if err := os.Remove(filepath.Join(r.dir, "archive/x")); err != nil {
		t.Fatal(err)
	}
	r.git("mv", "archive-move.md", "archive/move.md")
	head := r.commit("change")

	got, err := Changes(r.dir, base, head)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Change{
		"a.md":            {Status: 'M', Path: "a.md"},
		"new.md":          {Status: 'A', Path: "new.md"},
		"c.md":            {Status: 'D', Path: "c.md"},
		"archive/move.md": {Status: 'R', Path: "archive/move.md", OldPath: "move.md"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for _, c := range got {
		if want[c.Path] != c {
			t.Errorf("unexpected %+v", c)
		}
	}
}

func TestVerify(t *testing.T) {
	r := newRepo(t)
	key1, pub1 := keygen(t, "k1")
	key2, _ := keygen(t, "k2")
	signers := filepath.Join(t.TempDir(), "allowed_signers")
	if err := os.WriteFile(signers, []byte("test@example.com "+pub1+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.git("config", "--file", filepath.Join(r.home, "gitconfig"), "gpg.format", "ssh")
	sign := func(key, file string) string {
		r.write(file, file)
		r.git("config", "--file", filepath.Join(r.home, "gitconfig"), "user.signingkey", key)
		r.git("add", file)
		r.git("commit", "-q", "-S", "-m", file)
		return r.git("rev-parse", "HEAD")
	}
	good := sign(key1, "good")
	other := sign(key2, "other")
	r.write("plain", "plain")
	r.git("add", "plain")
	r.git("commit", "-q", "--no-gpg-sign", "-m", "plain")
	plain := r.git("rev-parse", "HEAD")

	if err := Verify(r.dir, good, signers); err != nil {
		t.Errorf("good: %v", err)
	}
	if err := Verify(r.dir, other, signers); err == nil {
		t.Error("unlisted key verified")
	}
	if err := Verify(r.dir, plain, signers); err == nil {
		t.Error("unsigned verified")
	}
}

func TestMissingBase(t *testing.T) {
	r := newRepo(t)
	r.write("a", "a")
	head := r.commit("one")
	missing := strings.Repeat("0", 40)
	if _, err := Commits(r.dir, missing, head); !errors.Is(err, ErrMissingBase) {
		t.Errorf("Commits: %v", err)
	}
	if _, err := Changes(r.dir, missing, head); !errors.Is(err, ErrMissingBase) {
		t.Errorf("Changes: %v", err)
	}
}

func TestRangesAndShow(t *testing.T) {
	r := newRepo(t)
	r.write("a", "a\n")
	c1 := r.commit("one")
	r.write("b", "b\n")
	c2 := r.commit("two")
	r.write("c", "c\n")
	c3 := r.commit("three")

	cs, err := Commits(r.dir, c1, c3)
	if err != nil || len(cs) != 2 || cs[0] != c2 || cs[1] != c3 {
		t.Errorf("Commits: %v %v", cs, err)
	}
	fp, err := FirstParentCommits(r.dir, c1, c3)
	if err != nil || len(fp) != 2 || fp[0] != c2 {
		t.Errorf("FirstParentCommits: %v %v", fp, err)
	}
	ps, err := Parents(r.dir, c2)
	if err != nil || len(ps) != 1 || ps[0] != c1 {
		t.Errorf("Parents: %v %v", ps, err)
	}
	if ps, _ := Parents(r.dir, c1); len(ps) != 0 {
		t.Errorf("root parents: %v", ps)
	}
	fc, err := FilesChanged(r.dir, c2)
	if err != nil || len(fc) != 1 || fc[0] != "b" {
		t.Errorf("FilesChanged: %v %v", fc, err)
	}
	if fc, _ := FilesChanged(r.dir, c1); len(fc) != 1 || fc[0] != "a" {
		t.Errorf("root FilesChanged: %v", fc)
	}
	b, err := Show(r.dir, c3, "b")
	if err != nil || string(b) != "b\n" {
		t.Errorf("Show: %q %v", b, err)
	}
}

func TestAddedOnly(t *testing.T) {
	r := newRepo(t)
	r.write("n.md", "one\ntwo\n")
	base := r.commit("base")
	r.write("n.md", "one\ntwo\nthree\n")
	appended := r.commit("append")
	r.write("n.md", "one\nTWO\nthree\n")
	changed := r.commit("change")

	if ok, err := AddedOnly(r.dir, base, appended, "n.md"); err != nil || !ok {
		t.Errorf("append: %v %v", ok, err)
	}
	if ok, err := AddedOnly(r.dir, appended, changed, "n.md"); err != nil || ok {
		t.Errorf("change: %v %v", ok, err)
	}
}

func listSorted(t *testing.T, root string) []string {
	t.Helper()
	got, err := ListFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	return got
}

func TestListFiles(t *testing.T) {
	r := newRepo(t)
	r.write("a.md", "a")
	r.commit("add a")
	r.write("b.md", "b")
	r.write(".gitignore", "ignored.md\n")
	r.write("ignored.md", "x")
	want := []string{".gitignore", "a.md", "b.md"}
	if got := listSorted(t, r.dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListFilesUserIgnore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	if err := os.MkdirAll(filepath.Join(home, ".config", "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "git", "ignore"), []byte(".scratch/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := newRepo(t)
	r.write("a.md", "a")
	r.write(".scratch/plan.md", "p")
	want := []string{"a.md"}
	if got := listSorted(t, r.dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListFilesOddNames(t *testing.T) {
	r := newRepo(t)
	r.write("notes/a b.md", "a")
	r.write("notes/ü.md", "u")
	want := []string{"notes/a b.md", "notes/ü.md"}
	if got := listSorted(t, r.dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListFilesSubdir(t *testing.T) {
	r := newRepo(t)
	r.write("sub/x.md", "x")
	r.write("top.md", "t")
	r.commit("add files")
	want := []string{"x.md"}
	if got := listSorted(t, filepath.Join(r.dir, "sub")); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
