package gitx

import "testing"

func TestMergeBaseAndNoRenames(t *testing.T) {
	r := newRepo(t)
	r.write("a.md", "alpha line one\nalpha line two\nalpha line three\n")
	fork := r.commit("one")
	r.git("checkout", "-q", "-b", "side")
	r.write("b", "b\n")
	side := r.commit("side")
	r.git("checkout", "-q", "main")
	r.write("c", "c\n")
	main := r.commit("main")
	mb, err := MergeBase(r.dir, main, side)
	if err != nil || mb != fork {
		t.Errorf("MergeBase: %q %v", mb, err)
	}
	r.git("checkout", "-q", "side")
	r.git("mv", "a.md", "z.md")
	ren := r.commit("rename")
	fc, err := FilesChanged(r.dir, ren)
	if err != nil || len(fc) != 2 {
		t.Errorf("FilesChanged: %v %v", fc, err)
	}
}
