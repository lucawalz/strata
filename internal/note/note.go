package note

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.yaml.in/yaml/v4"
)

const (
	fenceLine = "---"
	bom       = "\xef\xbb\xbf"
)

var (
	ErrNoFrontmatter           = errors.New("no frontmatter")
	ErrUnterminatedFrontmatter = errors.New("unterminated frontmatter")
)

type Frontmatter struct {
	ID           string   `yaml:"id"`
	Type         string   `yaml:"type"`
	Status       string   `yaml:"status"`
	Summary      string   `yaml:"summary"`
	SupersededBy string   `yaml:"superseded_by"`
	Tags         []string `yaml:"tags"`
	Sources      []string `yaml:"sources"`
	Refs         []string `yaml:"refs"`
}

type Link struct {
	Target string
	Line   int
	Image  bool
}

type Heading struct {
	Level, Line int
	Text        string
}

type Note struct {
	Path             string
	Frontmatter      Frontmatter
	RawFrontmatter   []byte
	FrontmatterLines int
	Body             []byte
	Headings         []Heading
	Links            []Link
	Chars            int
	HashLineBeforeH1 int
}

func Parse(path string, src []byte) (*Note, error) {
	s := strings.TrimPrefix(string(src), bom)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.SplitAfter(s, "\n")
	if strings.TrimRight(lines[0], "\n") != fenceLine {
		return nil, ErrNoFrontmatter
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\n") == fenceLine {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, ErrUnterminatedFrontmatter
	}

	n := &Note{
		Path:             path,
		RawFrontmatter:   []byte(strings.Join(lines[1:end], "")),
		FrontmatterLines: end + 1,
		Body:             []byte(strings.Join(lines[end+1:], "")),
	}
	if err := yaml.Unmarshal(n.RawFrontmatter, &n.Frontmatter); err != nil {
		return nil, fmt.Errorf("%s: frontmatter: %w", path, err)
	}
	for i := 1; i < end; i++ {
		if strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), "#") {
			n.HashLineBeforeH1 = i + 1
			break
		}
	}
	n.Chars = utf8.RuneCount(n.Body)
	n.walk()
	return n, nil
}

func (n *Note) walk() {
	root := goldmark.New().Parser().Parse(text.NewReader(n.Body))
	lineAt := func(offset int) int {
		return n.FrontmatterLines + 1 + bytes.Count(n.Body[:offset], []byte("\n"))
	}
	last := n.FrontmatterLines + 1
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := node.(type) {
		case *ast.Heading:
			h := Heading{Level: v.Level, Line: last}
			if v.Lines().Len() > 0 {
				seg := v.Lines().At(0)
				h.Line = lineAt(seg.Start)
				h.Text = strings.TrimSpace(string(seg.Value(n.Body)))
			}
			n.Headings = append(n.Headings, h)
		case *ast.Link:
			n.Links = append(n.Links, Link{Target: string(v.Destination), Line: firstLine(v, lineAt, last)})
		case *ast.Image:
			n.Links = append(n.Links, Link{Target: string(v.Destination), Line: firstLine(v, lineAt, last), Image: true})
		case *ast.Text:
			last = lineAt(v.Segment.Start)
		}
		return ast.WalkContinue, nil
	})
}

func firstLine(node ast.Node, lineAt func(int) int, fallback int) int {
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			return lineAt(t.Segment.Start)
		}
	}
	return fallback
}
