package skillsync

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	dirMode  = 0o755
	fileMode = 0o644
)

func Sync(src fs.FS, to string) error {
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(to, dirMode); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			if err := replace(src, e.Name(), to); err != nil {
				return err
			}
		}
	}
	return nil
}

func replace(src fs.FS, name, to string) error {
	tmp, err := os.MkdirTemp(to, "."+name+"-new-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	err = fs.WalkDir(src, name, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(tmp, filepath.FromSlash(path[len(name):]))
		if d.IsDir() {
			return os.MkdirAll(dst, dirMode)
		}
		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, fileMode)
	})
	if err != nil {
		return err
	}
	final := filepath.Join(to, name)
	old := tmp + "-old"
	if err := os.Rename(final, old); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Rename(old, final)
		return err
	}
	return os.RemoveAll(old)
}
