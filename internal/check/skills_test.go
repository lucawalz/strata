package check

import (
	"io/fs"
	"testing"

	"github.com/lucawalz/strata/internal/inject"
	"github.com/lucawalz/strata/internal/pii"
	"github.com/lucawalz/strata/skills"
)

func TestShippedSkillsPassContentScanners(t *testing.T) {
	err := fs.WalkDir(skills.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		src, err := skills.FS.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range pii.Scan(src) {
			t.Errorf("%s:%d: pii %s", path, m.Line, m.Kind)
		}
		for _, m := range inject.Scan(src) {
			t.Errorf("%s:%d: hidden content %s", path, m.Line, m.Kind)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
