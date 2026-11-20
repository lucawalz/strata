package skillsync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucawalz/strata/skills"
)

func TestSyncReplacesShippedAndKeepsLocal(t *testing.T) {
	to := t.TempDir()
	mine := filepath.Join(to, "mine", "SKILL.md")
	stale := filepath.Join(to, "capture", "stale.md")
	for path, body := range map[string]string{mine: "local", stale: "old", filepath.Join(to, "capture", "SKILL.md"): "outdated"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Sync(skills.FS, to); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(mine); string(got) != "local" {
		t.Errorf("local skill changed: %q", got)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Error("stale file in shipped skill survived")
	}
	want, err := skills.FS.ReadFile("capture/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(to, "capture", "SKILL.md"))
	if string(got) != string(want) {
		t.Error("capture does not match embedded copy")
	}
	if _, err := os.Stat(filepath.Join(to, "curate", "SKILL.md")); err != nil {
		t.Error("curate not written")
	}
}
