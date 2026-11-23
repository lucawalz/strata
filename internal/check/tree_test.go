package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lucawalz/strata/internal/config"
)

type want struct {
	check string
	level Level
	path  string
}

func testConfig() config.Config {
	c := config.Default()
	c.Caps = config.Caps{TargetMinChars: 50, TargetMaxChars: 300, HardMaxChars: 600}
	c.MaxTags = 3
	return c
}

func TestTree(t *testing.T) {
	cases := map[string][]want{
		"clean":                 nil,
		"frontmatter_schema":    {{"frontmatter", Error, "notes/good.md"}},
		"id_mismatch":           {{"id", Error, "notes/good.md"}},
		"id_duplicate":          {{"id", Error, "archive/good.md"}},
		"size_hard":             {{"size", Error, "notes/good.md"}},
		"size_min":              {{"size", Warning, "notes/good.md"}},
		"h1_zero":               {{"structure", Error, "notes/good.md"}},
		"h1_two":                {{"structure", Error, "notes/good.md"}},
		"h1_frontmatter":        {{"structure", Error, "notes/good.md"}},
		"link_missing":          {{"links", Error, "notes/good.md"}},
		"source_missing":        {{"links", Error, "notes/good.md"}},
		"superseded_by_missing": {{"links", Error, "archive/old.md"}},
		"superseded_no_by":      {{"links", Error, "archive/old.md"}},
		"superseded_in_notes":   {{"archive", Error, "notes/old.md"}},
		"archive_active":        {{"archive", Error, "archive/good.md"}},
		"moved_with_link":       {{"links", Error, "notes/good.md"}},
		"tags_over":             {{"frontmatter", Error, "notes/good.md"}},
		"config_invalid":        {{"config", Error, "strata.yml"}},
		"inject_bidi":           {{"inject", Error, "notes/good.md"}},
		"pii_iban":              {{"pii", Error, "notes/good.md"}},
		"german":                nil,
		"archive_loose":         nil,
	}
	for name, exp := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := Tree(Repo{Root: filepath.Join("..", "..", "testdata", "repos", name), Config: testConfig()})
			if err != nil {
				t.Fatal(err)
			}
			var have []want
			for _, f := range got {
				have = append(have, want{f.Check, f.Level, f.Path})
			}
			if !sameSet(have, exp) {
				t.Fatalf("got %v, want %v (%v)", have, exp, got)
			}
		})
	}
}

func sameSet(a, b []want) bool {
	if len(a) != len(b) {
		return false
	}
	for _, x := range a {
		found := false
		for _, y := range b {
			if reflect.DeepEqual(x, y) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestLevelString(t *testing.T) {
	if Info.String() != "info" || Warning.String() != "warning" || Error.String() != "error" {
		t.Fatal("level names")
	}
}

const syntheticIBAN = "DE89370400440532013000"

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	dst := t.TempDir()
	if err := os.CopyFS(dst, os.DirFS(filepath.Join("..", "..", "testdata", "repos", name))); err != nil {
		t.Fatal(err)
	}
	return dst
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestTreeIgnoredFile(t *testing.T) {
	root := copyFixture(t, "pii_ignored")
	gitIn(t, root, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "local.md"), []byte("Account "+syntheticIBAN+" in body text.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Tree(Repo{Root: root, Config: testConfig()})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want no findings", got)
	}
}

func TestTreeNestedRepository(t *testing.T) {
	root := copyFixture(t, "clean")
	gitIn(t, root, "init", "-q", "-b", "main")
	nested := filepath.Join(root, "tools", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, nested, "init", "-q")
	if err := os.WriteFile(filepath.Join(nested, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Tree(Repo{Root: root, Config: testConfig()}); err != nil {
		t.Fatal(err)
	}
}

func TestTreeGitlinkDirectory(t *testing.T) {
	root := copyFixture(t, "clean")
	gitIn(t, root, "init", "-q", "-b", "main")
	const gitlinkMode = "160000,1111111111111111111111111111111111111111,vendor/sub"
	gitIn(t, root, "update-index", "--add", "--cacheinfo", gitlinkMode)
	if err := os.MkdirAll(filepath.Join(root, "vendor", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Tree(Repo{Root: root, Config: testConfig()}); err != nil {
		t.Fatal(err)
	}
}

func TestTreeOutsideGit(t *testing.T) {
	_, err := Tree(Repo{Root: copyFixture(t, "clean"), Config: testConfig()})
	if err == nil || !strings.Contains(err.Error(), "git") {
		t.Fatalf("got %v, want an error naming git", err)
	}
}

func TestTreeDeletedTracked(t *testing.T) {
	root := copyFixture(t, "clean")
	inbox := filepath.Join(root, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(inbox, "gone.md")
	if err := os.WriteFile(gone, []byte("inbox text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "add", ".")
	gitIn(t, root, "commit", "-q", "-m", "add notes")
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	if _, err := Tree(Repo{Root: root, Config: testConfig()}); err != nil {
		t.Fatal(err)
	}
}
