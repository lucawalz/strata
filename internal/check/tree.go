package check

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/lucawalz/strata/internal/config"
	"github.com/lucawalz/strata/internal/gitx"
	"github.com/lucawalz/strata/internal/inject"
	"github.com/lucawalz/strata/internal/note"
	"github.com/lucawalz/strata/internal/pii"
	"github.com/lucawalz/strata/schema"
)

const (
	notesDir     = "notes"
	archiveDir   = "archive"
	inboxDir     = "inbox"
	rawDir       = "raw"
	configFile   = "strata.yml"
	statusSuper  = "superseded"
	binarySniff  = 8000
	markdownExt  = ".md"
	frontmatterL = 1
)

type parsed struct {
	rel     string
	dir     string
	archive bool
	note    *note.Note
}

func Tree(repo Repo) ([]Finding, error) {
	var out []Finding
	add := func(check string, level Level, path string, line int, format string, args ...any) {
		out = append(out, Finding{Check: check, Level: level, Path: path, Line: line, Message: fmt.Sprintf(format, args...)})
	}

	if raw, err := os.ReadFile(filepath.Join(repo.Root, configFile)); err == nil {
		if _, err := config.Load(raw); err != nil {
			add(CheckConfig, Error, configFile, 0, "%v", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	tracked, err := gitx.ListFiles(repo.Root)
	if err != nil {
		return nil, err
	}

	var notes []parsed
	for _, d := range []string{notesDir, archiveDir} {
		for _, rel := range markdownFiles(tracked, d) {
			src, ok, err := readTracked(repo.Root, rel)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			n, err := note.Parse(rel, src)
			if err != nil {
				add(CheckFrontmatter, Error, rel, frontmatterL, "%v", err)
				continue
			}
			for _, e := range schema.ValidateNote(n.RawFrontmatter) {
				add(CheckFrontmatter, Error, rel, frontmatterL, "%v", e)
			}
			notes = append(notes, parsed{rel: rel, dir: filepath.Dir(filepath.Join(repo.Root, filepath.FromSlash(rel))), archive: d == archiveDir, note: n})
		}
	}

	ids := map[string]string{}
	for _, p := range notes {
		if first, dup := ids[p.note.Frontmatter.ID]; dup {
			add(CheckID, Error, p.rel, frontmatterL, "id %q already used by %s", p.note.Frontmatter.ID, first)
		} else {
			ids[p.note.Frontmatter.ID] = p.rel
		}
	}

	for _, p := range notes {
		n, fm := p.note, p.note.Frontmatter
		stem := strings.TrimSuffix(filepath.Base(p.rel), markdownExt)
		if fm.ID != stem {
			add(CheckID, Error, p.rel, frontmatterL, "id %q differs from file name %q", fm.ID, stem)
		}
		if len(fm.Tags) > repo.Config.MaxTags {
			add(CheckFrontmatter, Error, p.rel, frontmatterL, "%d tags exceed max_tags %d", len(fm.Tags), repo.Config.MaxTags)
		}

		if !p.archive {
			caps := repo.Config.Caps
			switch {
			case n.Chars > caps.HardMaxChars:
				add(CheckSize, Error, p.rel, 0, "body has %d characters, hard maximum is %d", n.Chars, caps.HardMaxChars)
			case n.Chars > caps.TargetMaxChars:
				add(CheckSize, Warning, p.rel, 0, "body has %d characters, target maximum is %d", n.Chars, caps.TargetMaxChars)
			case n.Chars < caps.TargetMinChars:
				add(CheckSize, Warning, p.rel, 0, "body has %d characters, target minimum is %d", n.Chars, caps.TargetMinChars)
			}

			var h1 []note.Heading
			for _, h := range n.Headings {
				if h.Level == 1 {
					h1 = append(h1, h)
				}
			}
			switch {
			case len(h1) == 0:
				add(CheckStructure, Error, p.rel, n.FrontmatterLines+1, "note has no H1")
			case len(h1) > 1:
				add(CheckStructure, Error, p.rel, h1[1].Line, "note has %d H1 headings", len(h1))
			}
			if n.HashLineBeforeH1 > 0 {
				add(CheckStructure, Error, p.rel, n.HashLineBeforeH1, "line starting with # inside frontmatter")
			}
		}

		for _, l := range n.Links {
			if missingLink(p.dir, l.Target) {
				add(CheckLinks, Error, p.rel, l.Line, "link target %q does not exist", l.Target)
			}
		}
		for _, s := range fm.Sources {
			if _, err := os.Stat(filepath.Join(repo.Root, filepath.FromSlash(s))); err != nil {
				add(CheckLinks, Error, p.rel, frontmatterL, "source %q does not exist", s)
			}
		}
		if fm.SupersededBy != "" {
			if _, ok := ids[fm.SupersededBy]; !ok {
				add(CheckLinks, Error, p.rel, frontmatterL, "superseded_by %q names no note", fm.SupersededBy)
			}
		}
		if fm.Status == statusSuper && fm.SupersededBy == "" {
			add(CheckLinks, Error, p.rel, frontmatterL, "superseded note lacks superseded_by")
		}

		if fm.Status == statusSuper && !p.archive {
			add(CheckArchive, Error, p.rel, frontmatterL, "superseded note belongs in archive/")
		}
		if p.archive && fm.Status != statusSuper {
			add(CheckArchive, Error, p.rel, frontmatterL, "archived note must have status superseded, got %q", fm.Status)
		}
	}

	for _, d := range []string{notesDir, inboxDir} {
		for _, rel := range markdownFiles(tracked, d) {
			src, ok, err := readTracked(repo.Root, rel)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			for _, m := range inject.Scan(src) {
				add(CheckInject, injectLevel(m), rel, m.Line, "%s at column %d", m.Kind, m.Col)
			}
		}
	}

	for _, rel := range tracked {
		if strings.HasPrefix(rel, rawDir+"/") {
			continue
		}
		src, ok, err := readTracked(repo.Root, rel)
		if err != nil {
			return nil, err
		}
		if !ok || !isText(src) {
			continue
		}
		for _, m := range pii.Scan(src) {
			add(CheckPII, Error, rel, m.Line, "%s at column %d", m.Kind, m.Col)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Check < b.Check
	})
	return out, nil
}

func readTracked(root, rel string) ([]byte, bool, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil, false, nil
	}
	src, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return src, err == nil, err
}

func markdownFiles(files []string, dir string) []string {
	var out []string
	for _, rel := range files {
		if strings.HasPrefix(rel, dir+"/") && filepath.Ext(rel) == markdownExt {
			out = append(out, rel)
		}
	}
	return out
}

func missingLink(dir, target string) bool {
	u, err := url.Parse(target)
	if err != nil || u.Scheme != "" || u.Host != "" || u.Path == "" {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, filepath.FromSlash(u.Path)))
	return err != nil
}

func isText(src []byte) bool {
	head := src[:min(len(src), binarySniff)]
	return !bytes.Contains(head, []byte{0}) && utf8.Valid(src)
}
