package note

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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

var yamlLine = regexp.MustCompile(`line (\d+)`)

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
		return nil, fmt.Errorf("%s: frontmatter: %s", path, offsetYAMLLines(err.Error()))
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
			h := Heading{Level: v.Level, Line: n.emptyHeadingLine(v.Level, last)}
			if v.Lines().Len() > 0 {
				seg := v.Lines().At(0)
				h.Line = lineAt(seg.Start)
				h.Text = strings.TrimSpace(string(seg.Value(n.Body)))
			}
			n.Headings = append(n.Headings, h)
		case *ast.Link:
			n.Links = append(n.Links, Link{Target: string(v.Destination), Line: n.linkLine(v, v.Destination, lineAt, last)})
		case *ast.Image:
			n.Links = append(n.Links, Link{Target: string(v.Destination), Line: n.linkLine(v, v.Destination, lineAt, last), Image: true})
		case *ast.Text:
			last = lineAt(v.Segment.Start)
		}
		return ast.WalkContinue, nil
	})
}

func (n *Note) linkLine(node ast.Node, dest []byte, lineAt func(int) int, fallback int) int {
	line := 0
	_ = ast.Walk(node, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			line = lineAt(t.Segment.Start)
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if line > 0 {
		return line
	}
	needle := append([]byte("]("), dest...)
	for p := node.Parent(); p != nil; p = p.Parent() {
		segs := p.Lines()
		if p.Type() != ast.TypeBlock || segs.Len() == 0 {
			continue
		}
		for i := 0; i < segs.Len(); i++ {
			if seg := segs.At(i); bytes.Contains(seg.Value(n.Body), needle) {
				return lineAt(seg.Start)
			}
		}
		return lineAt(segs.At(0).Start)
	}
	return fallback
}

func (n *Note) emptyHeadingLine(level, from int) int {
	marker := strings.Repeat("#", level)
	lines := strings.Split(string(n.Body), "\n")
	for i := max(from-n.FrontmatterLines-1, 0); i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == marker {
			return n.FrontmatterLines + 1 + i
		}
	}
	return from
}

func offsetYAMLLines(msg string) string {
	return yamlLine.ReplaceAllStringFunc(msg, func(m string) string {
		n, _ := strconv.Atoi(m[len("line "):])
		return "line " + strconv.Itoa(n+1)
	})
}
