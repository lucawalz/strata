package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrMissingBase = errors.New("base revision not found")

type Change struct {
	Status  byte
	Path    string
	OldPath string
}

func run(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_OPTIONAL_LOCKS=0",
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

func requireCommit(root, rev string) error {
	if _, err := run(root, "rev-parse", "--verify", "--quiet", "--end-of-options", rev+"^{commit}"); err != nil {
		return fmt.Errorf("%w: %s", ErrMissingBase, rev)
	}
	return nil
}

func lines(b []byte) []string {
	return strings.Fields(string(b))
}

func Changes(root, base, head string) ([]Change, error) {
	if err := requireCommit(root, base); err != nil {
		return nil, err
	}
	out, err := run(root, "diff", "--name-status", "-M", "-z", base, head)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	var changes []Change
	for i := 0; i < len(parts) && parts[i] != ""; {
		status := parts[i][0]
		n := 2
		if status == 'R' || status == 'C' {
			n = 3
		}
		if i+n > len(parts) {
			return nil, fmt.Errorf("unexpected diff output")
		}
		c := Change{Status: status, Path: parts[i+n-1]}
		if n == 3 {
			c.OldPath = parts[i+1]
		}
		changes = append(changes, c)
		i += n
	}
	return changes, nil
}

func Commits(root, base, head string) ([]string, error) {
	if err := requireCommit(root, base); err != nil {
		return nil, err
	}
	out, err := run(root, "rev-list", "--reverse", base+".."+head)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

func FirstParentCommits(root, before, after string) ([]string, error) {
	if err := requireCommit(root, before); err != nil {
		return nil, err
	}
	out, err := run(root, "rev-list", "--reverse", "--first-parent", before+".."+after)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

func Parents(root, sha string) ([]string, error) {
	out, err := run(root, "rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return nil, err
	}
	f := lines(out)
	if len(f) == 0 {
		return nil, fmt.Errorf("unknown commit %s", sha)
	}
	return f[1:], nil
}

func FilesChanged(root, sha string) ([]string, error) {
	parents, err := Parents(root, sha)
	if err != nil {
		return nil, err
	}
	var out []byte
	if len(parents) == 0 {
		out, err = run(root, "diff-tree", "-r", "--root", "--no-commit-id", "--name-only", "-z", sha)
	} else {
		out, err = run(root, "diff", "--name-only", "-z", parents[0], sha)
	}
	if err != nil {
		return nil, err
	}
	var files []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

func Show(root, rev, path string) ([]byte, error) {
	return run(root, "show", rev+":"+path)
}

func Verify(root, sha, allowedSigners string) error {
	_, err := run(root, "-c", "gpg.ssh.allowedSignersFile="+allowedSigners, "verify-commit", sha)
	return err
}

func AddedOnly(root, base, head, path string) (bool, error) {
	out, err := run(root, "diff", "--numstat", base, head, "--", path)
	if err != nil {
		return false, err
	}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l == "" {
			continue
		}
		f := strings.Split(l, "\t")
		if len(f) < 3 || f[1] != "0" {
			return false, nil
		}
	}
	return true, nil
}
