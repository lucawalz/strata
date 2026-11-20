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

func governed(path string, paths []string) bool {
	for _, g := range paths {
		if path == g || (strings.HasSuffix(g, "/") && strings.HasPrefix(path, g)) {
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

	changes, err := gitx.Changes(repo.Root, base, head)
	if err != nil {
		return nil, err
	}

	for _, c := range changes {
		switch {
		case c.Status == statusAdded || c.Status == statusCopied:
		case under(c.Path, rawDir) || (c.OldPath != "" && under(c.OldPath, rawDir)):
			add(CheckRaw, Error, c.Path, 0, "raw/ is append-only, file was changed, deleted or renamed")
		}
		if c.Status != statusModified || !under(c.Path, notesDir) {
			continue
		}
		if ok, err := gitx.AddedOnly(repo.Root, base, head, c.Path); err != nil {
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

	if err := governance(repo, base, head, changes, add); err != nil {
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

func governance(repo Repo, base, head string, changes []gitx.Change, add adder) error {
	cfg := config.Default()
	if raw, err := gitx.Show(repo.Root, base, configFile); err == nil {
		if cfg, err = config.Load(raw); err != nil {
			add(CheckConfig, Error, configFile, 0, "%v", err)
			cfg = config.Default()
		}
	}

	var touched, other []string
	for _, c := range changes {
		paths := []string{c.Path}
		if c.OldPath != "" {
			paths = append(paths, c.OldPath)
		}
		hit := false
		for _, p := range paths {
			if governed(p, cfg.GovernancePaths) {
				hit = true
				touched = append(touched, p)
			}
		}
		if !hit {
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

	signers, err := gitx.Show(repo.Root, base, cfg.AllowedSigners)
	if err != nil {
		signers = nil
	}
	if len(strings.TrimSpace(string(signers))) == 0 {
		add(CheckGovernance, Error, cfg.AllowedSigners, 0, "allowed_signers is empty")
		return nil
	}
	dir, err := os.MkdirTemp("", "strata-signers")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	signersPath := filepath.Join(dir, "allowed_signers")
	if err := os.WriteFile(signersPath, signers, 0o600); err != nil {
		return err
	}

	commits, err := gitx.Commits(repo.Root, base, head)
	if err != nil {
		return err
	}
	for _, sha := range commits {
		files, err := gitx.FilesChanged(repo.Root, sha)
		if err != nil {
			return err
		}
		for _, f := range files {
			if !governed(f, cfg.GovernancePaths) {
				continue
			}
			if err := gitx.Verify(repo.Root, sha, signersPath); err != nil {
				add(CheckGovernance, Error, f, 0, "commit %s touches governance files without a valid signature", sha[:min(len(sha), shortSHA)])
			}
			break
		}
	}
	return nil
}
