package check

import (
	"fmt"

	"github.com/lucawalz/strata/internal/gitx"
)

const (
	CheckAudit = "audit"
	zeroSHA    = "0000000000000000000000000000000000000000"
)

func Audit(repo Repo, before, after string) ([]Finding, error) {
	if before == zeroSHA {
		return nil, nil
	}
	g, err := loadGovernance(repo.Root, before)
	if err != nil {
		return nil, err
	}
	defer g.cleanup()
	out := g.findings

	seen := map[string]bool{}
	emptyReported := false
	fail := func(sha, path, format string) {
		out = append(out, Finding{Check: CheckAudit, Level: Error, Path: path, Message: fmt.Sprintf(format, sha[:min(len(sha), shortSHA)])})
	}
	first := func(files []string) string {
		for _, f := range files {
			if governed(f, g.cfg) {
				return f
			}
		}
		return ""
	}
	verify := func(sha string, files []string) {
		f := first(files)
		if f == "" || seen[sha] {
			return
		}
		seen[sha] = true
		if g.empty {
			if !emptyReported {
				emptyReported = true
				out = append(out, Finding{Check: CheckAudit, Level: Error, Path: g.cfg.AllowedSigners, Message: "allowed_signers is empty"})
			}
			return
		}
		if gitx.Verify(repo.Root, sha, g.signersPath) != nil {
			fail(sha, f, "commit %s lacks a valid signature while governance files change")
		}
	}

	firstParents, err := gitx.FirstParentCommits(repo.Root, before, after)
	if err != nil {
		return nil, err
	}
	for _, sha := range firstParents {
		parents, err := gitx.Parents(repo.Root, sha)
		if err != nil {
			return nil, err
		}
		if len(parents) < 2 {
			files, err := gitx.FilesChanged(repo.Root, sha)
			if err != nil {
				return nil, err
			}
			if f := first(files); f != "" {
				fail(sha, f, "commit %s changes governance files outside a signed merge")
			}
			continue
		}
		var common map[string]bool
		for i, p := range parents {
			if i > 0 {
				side, err := gitx.Commits(repo.Root, parents[0], p)
				if err != nil {
					return nil, err
				}
				for _, c := range side {
					files, err := gitx.FilesChanged(repo.Root, c)
					if err != nil {
						return nil, err
					}
					verify(c, files)
				}
			}
			changes, err := gitx.Changes(repo.Root, p, sha)
			if err != nil {
				return nil, err
			}
			here := map[string]bool{}
			for _, c := range changes {
				here[c.Path] = true
				if c.OldPath != "" {
					here[c.OldPath] = true
				}
			}
			if common == nil {
				common = here
				continue
			}
			for f := range common {
				if !here[f] {
					delete(common, f)
				}
			}
		}
		var own []string
		for f := range common {
			own = append(own, f)
		}
		verify(sha, own)
	}
	return out, nil
}
