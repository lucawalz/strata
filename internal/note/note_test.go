package note

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

const minimalPath = "../../testdata/notes/minimal.md"

func readMinimal(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile(minimalPath)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func TestParseMinimal(t *testing.T) {
	src := readMinimal(t)
	n, err := Parse(minimalPath, src)
	if err != nil {
		t.Fatal(err)
	}
	if n.Frontmatter.ID != "k8s-pod-disruption-budgets" {
		t.Errorf("id = %q", n.Frontmatter.ID)
	}
	if len(n.Headings) != 3 {
		t.Fatalf("headings = %d", len(n.Headings))
	}
	if n.Headings[0].Level != 1 || n.Headings[0].Line != 9 || n.Headings[0].Text != "Pod disruption budgets" {
		t.Errorf("h1 = %+v", n.Headings[0])
	}
	if len(n.Links) != 1 || n.Links[0].Target != "other-note.md" || n.Links[0].Line != 12 || n.Links[0].Image {
		t.Errorf("links = %+v", n.Links)
	}
	if n.Chars != utf8.RuneCount(n.Body) {
		t.Errorf("chars = %d", n.Chars)
	}
	if n.FrontmatterLines != 7 {
		t.Errorf("frontmatter lines = %d", n.FrontmatterLines)
	}
	if n.HashLineBeforeH1 != 0 {
		t.Errorf("hash line = %d", n.HashLineBeforeH1)
	}
}

func TestParseCRLFAndBOM(t *testing.T) {
	src := readMinimal(t)
	want, err := Parse(minimalPath, src)
	if err != nil {
		t.Fatal(err)
	}
	crlf := append([]byte("\xef\xbb\xbf"), []byte(strings.ReplaceAll(string(src), "\n", "\r\n"))...)
	got, err := Parse(minimalPath, crlf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %+v, got %+v", want, got)
	}
}

func TestParseGermanCounts(t *testing.T) {
	src := "---\nid: a\ntype: concept\nstatus: active\nsummary: s\n---\nGr\xc3\xb6\xc3\x9fen\xc3\xa4nderung"
	n, err := Parse("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if n.Chars != 14 {
		t.Errorf("chars = %d, bytes = %d", n.Chars, len(n.Body))
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	_, err := Parse("a.md", []byte("# Title\n"))
	if !errors.Is(err, ErrNoFrontmatter) {
		t.Errorf("err = %v", err)
	}
}

func TestParseUnterminatedFrontmatter(t *testing.T) {
	_, err := Parse("a.md", []byte("---\nid: a\n"))
	if !errors.Is(err, ErrUnterminatedFrontmatter) {
		t.Errorf("err = %v", err)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	_, err := Parse("a.md", []byte("---\nid: a\ntags: x\n---\nbody\n"))
	if err == nil || !strings.Contains(err.Error(), "line 3:") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseLinkAndHeadingLines(t *testing.T) {
	src := "---\nid: a\n---\nfirst paragraph\n\n![](x.png)\n\n[`f`](a.md)\n\n[*e*](b.md)\n\n##\n\ntext [t](c.md)\n"
	n, err := Parse("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	wantLinks := []Link{
		{Target: "x.png", Line: 6, Image: true},
		{Target: "a.md", Line: 8},
		{Target: "b.md", Line: 10},
		{Target: "c.md", Line: 14},
	}
	if !reflect.DeepEqual(n.Links, wantLinks) {
		t.Errorf("links = %+v", n.Links)
	}
	if len(n.Headings) != 1 || n.Headings[0].Line != 12 {
		t.Errorf("headings = %+v", n.Headings)
	}
}

func TestParseHashCommentInFrontmatter(t *testing.T) {
	src := "---\nid: a\n# comment\ntype: concept\n---\n# Title\n"
	n, err := Parse("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if n.HashLineBeforeH1 != 3 {
		t.Errorf("hash line = %d", n.HashLineBeforeH1)
	}
}

func TestParseImageLink(t *testing.T) {
	src := "---\nid: a\n---\n![a](https://example.invalid/x.png)\n"
	n, err := Parse("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Links) != 1 || !n.Links[0].Image || n.Links[0].Line != 4 {
		t.Errorf("links = %+v", n.Links)
	}
}
