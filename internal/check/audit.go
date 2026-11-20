package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucawalz/strata/internal/config"
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
	cfg := config.Default()
	var out []Finding
	if raw, err := gitx.Show(repo.Root, after, configFile); err == nil {
		if cfg, err = config.Load(raw); err != nil {
			out = append(out, Finding{Check: CheckConfig, Level: Error, Path: configFile, Message: err.Error()})
			cfg = config.Default()
		}
	}
	signers, err := gitx.Show(repo.Root, after, cfg.AllowedSigners)
	if err != nil {
		signers = nil
	}
	dir, err := os.MkdirTemp("", "strata-signers")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	signersPath := filepath.Join(dir, "allowed_signers")
	if err := os.WriteFile(signersPath, signers, 0o600); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	require := func(sha string, files []string) {
		if seen[sha] {
			return
		}
		for _, f := range files {
			if !governed(f, cfg.GovernancePaths) {
				continue
			}
			seen[sha] = true
			if gitx.Verify(repo.Root, sha, signersPath) != nil {
				out = append(out, Finding{Check: CheckAudit, Level: Error, Path: f, Message: fmt.Sprintf("commit %s lacks a valid signature while governance files change", sha[:min(len(sha), shortSHA)])})
			}
			return
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
			require(sha, files)
			continue
		}
		side, err := gitx.Commits(repo.Root, parents[0], parents[1])
		if err != nil {
			return nil, err
		}
		for _, c := range side {
			files, err := gitx.FilesChanged(repo.Root, c)
			if err != nil {
				return nil, err
			}
			require(c, files)
		}
		changes, err := gitx.Changes(repo.Root, parents[1], sha)
		if err != nil {
			return nil, err
		}
		var files []string
		for _, c := range changes {
			files = append(files, c.Path)
			if c.OldPath != "" {
				files = append(files, c.OldPath)
			}
		}
		require(sha, files)
	}
	return out, nil
}
