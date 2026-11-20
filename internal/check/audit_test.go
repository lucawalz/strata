package check

import (
	"strings"
	"testing"
)

func auditWants(t *testing.T, r *gitRepo, before, after string) []Finding {
	t.Helper()
	got, err := Audit(Repo{Root: r.dir, Config: testConfig()}, before, after)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func sideBranch(r *gitRepo, base string, signed ...bool) {
	r.git("checkout", "-q", "-b", "side", base)
	for i, s := range signed {
		r.write("AGENTS.md", "rules side "+string(rune('a'+i))+"\n")
		r.commit(s, "AGENTS.md")
	}
	r.git("checkout", "-q", "main")
	r.write("notes/m.md", "main\n")
	r.commit(false, "notes")
}

func TestAuditSingleParent(t *testing.T) {
	r, base, _ := governedBase(t)
	r.write(".github/workflows/x.yml", "a\n")
	head := r.commit(false, ".github")
	got := auditWants(t, r, base, head)
	if !hasMsg(got, "audit", Error, head[:12]) {
		t.Errorf("got %v", got)
	}
}

func TestAuditSingleParentSignedStillFails(t *testing.T) {
	r, base, _ := governedBase(t)
	r.write(".github/workflows/x.yml", "a\n")
	head := r.commit(true, ".github")
	got := auditWants(t, r, base, head)
	if !hasMsg(got, "audit", Error, head[:12]) {
		t.Errorf("got %v", got)
	}
}

func TestAuditEmptySigners(t *testing.T) {
	r, _ := newGitRepo(t)
	r.write(".github/allowed_signers", "")
	r.write("AGENTS.md", "rules\n")
	base := r.commit(false, ".github", "AGENTS.md")
	sideBranch(r, base, true)
	r.git("merge", "-q", "--no-ff", "--no-gpg-sign", "side", "-m", "m")
	head := r.git("rev-parse", "HEAD")
	got := auditWants(t, r, base, head)
	if !hasMsg(got, "audit", Error, "allowed_signers is empty") {
		t.Errorf("got %v", got)
	}
}

func TestAuditMergeAfterMainGovernanceChange(t *testing.T) {
	r, base, _ := governedBase(t)
	r.git("checkout", "-q", "-b", "side", base)
	r.write("notes/s.md", "s\n")
	r.commit(false, "notes")
	r.git("checkout", "-q", "main")
	r.write("AGENTS.md", "rules v2\n")
	before := r.commit(true, "AGENTS.md")
	r.git("merge", "-q", "--no-ff", "--no-gpg-sign", "side", "-m", "m")
	head := r.git("rev-parse", "HEAD")
	if got := auditWants(t, r, before, head); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestAuditOctopusUnsignedThirdParent(t *testing.T) {
	r, base, _ := governedBase(t)
	r.git("checkout", "-q", "-b", "s1", base)
	r.write("notes/s1.md", "s\n")
	r.commit(false, "notes")
	r.git("checkout", "-q", "-b", "s2", base)
	r.write("AGENTS.md", "rules s2\n")
	bad := r.commit(false, "AGENTS.md")
	r.git("checkout", "-q", "main")
	r.write("notes/m.md", "m\n")
	r.commit(false, "notes")
	r.git("merge", "-q", "--no-gpg-sign", "s1", "s2", "-m", "m")
	head := r.git("rev-parse", "HEAD")
	got := auditWants(t, r, base, head)
	if !hasMsg(got, "audit", Error, bad[:12]) {
		t.Errorf("got %v", got)
	}
}

func TestAuditZeroBefore(t *testing.T) {
	r, _, _ := governedBase(t)
	r.write("AGENTS.md", "rules v2\n")
	head := r.commit(false, "AGENTS.md")
	if got := auditWants(t, r, strings.Repeat("0", 40), head); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestAuditNonGovernanceSquash(t *testing.T) {
	r, base, _ := governedBase(t)
	r.write("notes/a.md", "a\n")
	r.commit(false, "notes")
	r.write("notes/b.md", "b\n")
	head := r.commit(false, "notes")
	if got := auditWants(t, r, base, head); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestAuditMergeSignedSide(t *testing.T) {
	r, base, _ := governedBase(t)
	sideBranch(r, base, true, true)
	r.git("merge", "-q", "--no-ff", "--no-gpg-sign", "side", "-m", "m")
	head := r.git("rev-parse", "HEAD")
	if got := auditWants(t, r, base, head); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestAuditMergeUnsignedSide(t *testing.T) {
	r, base, _ := governedBase(t)
	sideBranch(r, base, true, false)
	bad := r.git("rev-parse", "side")
	r.git("merge", "-q", "--no-ff", "--no-gpg-sign", "side", "-m", "m")
	head := r.git("rev-parse", "HEAD")
	got := auditWants(t, r, base, head)
	if !hasMsg(got, "audit", Error, bad[:12]) || len(got) != 1 {
		t.Errorf("got %v", got)
	}
}

func TestAuditMergeSneakedChange(t *testing.T) {
	r, base, _ := governedBase(t)
	sideBranch(r, base, true)
	r.git("merge", "-q", "--no-ff", "--no-commit", "side")
	r.write("AGENTS.md", "sneaked\n")
	head := r.commit(false, "AGENTS.md")
	got := auditWants(t, r, base, head)
	if !hasMsg(got, "audit", Error, head[:12]) {
		t.Errorf("got %v", got)
	}
}
