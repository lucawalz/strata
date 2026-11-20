package check

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lucawalz/strata/internal/config"
	"github.com/lucawalz/strata/internal/gitx"
	"github.com/lucawalz/strata/internal/inject"
	"github.com/lucawalz/strata/internal/pii"
)

const (
	CheckRaw                 = "raw"
	CheckAppend              = "append"
	CheckGovernance          = "governance"
	CheckGovernanceIsolation = "governance-isolation"
	CheckGovernanceSummary   = "governance-summary"

	statusAdded    = 'A'
	statusModified = 'M'
	statusCopied   = 'C'
	statusRenamed  = 'R'
	shortSHA       = 12
)

func injectLevel(m inject.Match) Level {
	if m.Level == inject.LevelWarning {
		return Warning
	}
	return Error
}

func under(path, dir string) bool {
	return strings.HasPrefix(path, dir+"/")
}

func governed(path string, cfg config.Config) bool {
	p := strings.ToLower(path)
	switch p[strings.LastIndexByte(p, '/')+1:] {
	case "agents.md", "claude.md":
		return true
	}
	if p == configFile || p == strings.ToLower(cfg.AllowedSigners) {
		return true
	}
	for _, g := range cfg.GovernancePaths {
		g = strings.TrimSuffix(strings.ToLower(g), "/")
		if p == g || strings.HasPrefix(p, g+"/") {
			return true
		}
	}
	return false
}

func Diff(repo Repo, base, head string) ([]Finding, error) {
	var out []Finding
	add := func(check string, level Level, path string, line int, format string, args ...any) {
		out = append(out, Finding{Check: check, Level: level, Path: path, Line: line, Message: fmt.Sprintf(format, args...)})
	}

	mb, err := gitx.MergeBase(repo.Root, base, head)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", gitx.ErrMissingBase, base)
	}
	changes, err := gitx.Changes(repo.Root, mb, head)
	if err != nil {
		return nil, err
	}

	for _, c := range changes {
		src := c.Path
		if c.Status == statusRenamed {
			src = c.OldPath
		}
		if c.Status != statusAdded && c.Status != statusCopied && under(src, rawDir) {
			add(CheckRaw, Error, c.Path, 0, "raw/ is append-only, file was changed, deleted or renamed")
		}
		if c.Status != statusModified || !under(c.Path, notesDir) {
			continue
		}
		if ok, err := gitx.AddedOnly(repo.Root, mb, head, c.Path); err != nil {
			return nil, err
		} else if ok {
			add(CheckAppend, Warning, c.Path, 0, "note changed by appending lines only")
		}
	}

	for _, c := range changes {
		if c.Status == 'D' || under(c.Path, rawDir) {
			continue
		}
		src, err := gitx.Show(repo.Root, head, c.Path)
		if err != nil {
			return nil, err
		}
		if !isText(src) {
			continue
		}
		if under(c.Path, notesDir) || under(c.Path, inboxDir) {
			for _, m := range inject.Scan(src) {
				add(CheckInject, injectLevel(m), c.Path, m.Line, "%s at column %d", m.Kind, m.Col)
			}
		}
		for _, m := range pii.Scan(src) {
			add(CheckPII, Error, c.Path, m.Line, "%s at column %d", m.Kind, m.Col)
		}
	}

	if err := governance(repo, base, mb, head, changes, add); err != nil {
		return nil, err
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

type adder func(check string, level Level, path string, line int, format string, args ...any)

type govSetup struct {
	cfg         config.Config
	signersPath string
	empty       bool
	cleanup     func()
	findings    []Finding
}

func loadGovernance(root, rev string) (govSetup, error) {
	g := govSetup{cfg: config.Default(), cleanup: func() {}}
	if raw, err := gitx.Show(root, rev, configFile); err == nil {
		cfg, err := config.Load(raw)
		if err != nil {
			g.findings = append(g.findings, Finding{Check: CheckConfig, Level: Error, Path: configFile, Message: err.Error()})
		} else {
			g.cfg = cfg
		}
	}
	signers, err := gitx.Show(root, rev, g.cfg.AllowedSigners)
	if err != nil || len(strings.TrimSpace(string(signers))) == 0 {
		g.empty = true
		return g, nil
	}
	dir, err := os.MkdirTemp("", "strata-signers")
	if err != nil {
		return g, err
	}
	g.cleanup = func() { _ = os.RemoveAll(dir) }
	g.signersPath = filepath.Join(dir, "allowed_signers")
	if err := os.WriteFile(g.signersPath, signers, 0o600); err != nil {
		g.cleanup()
		return g, err
	}
	return g, nil
}

func governance(repo Repo, base, mb, head string, changes []gitx.Change, add adder) error {
	g, err := loadGovernance(repo.Root, base)
	if err != nil {
		return err
	}
	defer g.cleanup()
	for _, f := range g.findings {
		add(f.Check, f.Level, f.Path, f.Line, "%s", f.Message)
	}
	cfg := g.cfg

	var touched, other []string
	for _, c := range changes {
		paths := []string{c.Path}
		if c.OldPath != "" {
			paths = append(paths, c.OldPath)
		}
		ungoverned := false
		for _, p := range paths {
			if governed(p, cfg) {
				touched = append(touched, p)
			} else {
				ungoverned = true
			}
		}
		if ungoverned {
			other = append(other, c.Path)
		}
	}
	if len(touched) == 0 {
		return nil
	}
	sort.Strings(touched)
	add(CheckGovernanceSummary, Info, "", 0, "governance files changed: %s", strings.Join(touched, ", "))
	if len(other) > 0 {
		add(CheckGovernanceIsolation, Error, touched[0], 0, "governance files must change in their own pull request, also changed: %s", strings.Join(other, ", "))
	}
	if g.empty {
		add(CheckGovernance, Error, cfg.AllowedSigners, 0, "allowed_signers is empty")
		return nil
	}

	commits, err := gitx.Commits(repo.Root, mb, head)
	if err != nil {
		return err
	}
	for _, sha := range commits {
		if err := gitx.Verify(repo.Root, sha, g.signersPath); err != nil {
			add(CheckGovernance, Error, touched[0], 0, "commit %s lacks a valid signature while governance files change", sha[:min(len(sha), shortSHA)])
		}
	}
	return nil
}
