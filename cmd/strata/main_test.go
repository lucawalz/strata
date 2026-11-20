package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const noteText = "---\nid: %s\ntype: concept\nstatus: active\nsummary: Synthetic summary.\n---\n\n# Title\n\nBody text long enough to pass the minimum size check here.\n"

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repoWith(t *testing.T, caps string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "strata.yml", "caps:\n"+caps)
	writeFile(t, root, "notes/x.md", strings.Replace(noteText, "%s", "x", 1))
	return root
}

func exec1(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

const (
	okCaps   = "  target_min_chars: 1\n  target_max_chars: 1000\n  hard_max_chars: 2000\n"
	warnCaps = "  target_min_chars: 500\n  target_max_chars: 1000\n  hard_max_chars: 2000\n"
	failCaps = "  target_min_chars: 1\n  target_max_chars: 2\n  hard_max_chars: 3\n"
)

func TestTreeExitCodes(t *testing.T) {
	cases := []struct {
		name string
		caps string
		code int
		want string
	}{
		{"clean", okCaps, 0, ""},
		{"warning", warnCaps, 0, "WARNING size notes/x.md"},
		{"error", failCaps, 1, "ERROR size notes/x.md"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, _ := exec1(t, "check", "tree", "--root", repoWith(t, c.caps))
			if code != c.code || !strings.Contains(out, c.want) {
				t.Errorf("code %d out %q", code, out)
			}
			if c.want == "" && out != "" {
				t.Errorf("out %q", out)
			}
		})
	}
}

func TestGithubFormat(t *testing.T) {
	code, out, _ := exec1(t, "check", "tree", "--format", "github", "--root", repoWith(t, failCaps))
	if code != 1 || !strings.Contains(out, "::error file=notes/x.md,line=1,title=size::") {
		t.Errorf("code %d out %q", code, out)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"bogus"},
		{"check", "tree", "--nope"},
		{"check", "tree", "--format", "xml"},
		{"check", "diff", "--head", "HEAD"},
		{"audit", "--before", "a"},
	} {
		if code, _, _ := exec1(t, args...); code != 2 {
			t.Errorf("%v: code %d", args, code)
		}
	}
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestDiffAndAuditMissingBase(t *testing.T) {
	root := repoWith(t, okCaps)
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "--no-gpg-sign", "-m", "c")
	head := gitIn(t, root, "rev-parse", "HEAD")
	missing := strings.Repeat("a", 40)

	code, _, errOut := exec1(t, "check", "diff", "--root", root, "--base", missing, "--head", head)
	if code != 2 || !strings.Contains(errOut, "base "+missing+" not found, fetch full history") {
		t.Errorf("diff: code %d err %q", code, errOut)
	}
	code, _, errOut = exec1(t, "audit", "--root", root, "--before", missing, "--after", head)
	if code != 2 || !strings.Contains(errOut, "not found, fetch full history") {
		t.Errorf("audit: code %d err %q", code, errOut)
	}
	if code, out, _ := exec1(t, "check", "diff", "--root", root, "--base", head, "--head", head); code != 0 {
		t.Errorf("same rev: code %d out %q", code, out)
	}
}

func TestGovernanceSummaryFile(t *testing.T) {
	root := repoWith(t, okCaps)
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "--no-gpg-sign", "-m", "c")
	base := gitIn(t, root, "rev-parse", "HEAD")
	writeFile(t, root, "AGENTS.md", "rules\n")
	gitIn(t, root, "add", "-f", "AGENTS.md")
	gitIn(t, root, "commit", "-q", "--no-gpg-sign", "-m", "g")
	summary := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	_, out, _ := exec1(t, "check", "diff", "--format", "github", "--root", root, "--base", base, "--head", "HEAD")
	got, _ := os.ReadFile(summary)
	if !strings.Contains(string(got), "AGENTS.md") || strings.Contains(out, "::notice") {
		t.Errorf("summary %q out %q", got, out)
	}
}

func TestCalibrateMissingKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	code, _, errb := exec1(t, "calibrate", "--model", "m", t.TempDir())
	if code != exitUsage || !strings.Contains(errb, "ANTHROPIC_API_KEY") {
		t.Fatalf("code %d stderr %q", code, errb)
	}
}
