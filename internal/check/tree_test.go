package check

import (
	"path/filepath"
	"reflect"
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
