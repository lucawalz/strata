package skillsync

import (
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
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(to, e.Name())); err != nil {
			return err
		}
		err := fs.WalkDir(src, e.Name(), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			dst := filepath.Join(to, filepath.FromSlash(path))
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
	}
	return nil
}
