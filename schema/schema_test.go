package schema

import (
	"strings"
	"testing"
)

const validNote = "id: k8s-pod-disruption-budgets\ntype: concept\nstatus: active\nsummary: Synthetic summary.\ntags: [kubernetes]\nrefs: [\"adr:0001\"]\n"

func TestValidateNote(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		ok   bool
	}{
		{"valid", validNote, true},
		{"summary too long", "id: a\ntype: concept\nstatus: active\nsummary: " + strings.Repeat("x", 201) + "\n", false},
		{"summary at limit", "id: a\ntype: concept\nstatus: active\nsummary: " + strings.Repeat("x", 200) + "\n", true},
		{"bad type", "id: a\ntype: note\nstatus: active\nsummary: s\n", false},
		{"unknown key", validNote + "updated: 2026-01-01\n", false},
		{"missing summary", "id: a\ntype: concept\nstatus: active\n", false},
		{"bad id", "id: A_b\ntype: concept\nstatus: active\nsummary: s\n", false},
		{"bad ref", "id: a\ntype: concept\nstatus: active\nsummary: s\nrefs: [nocolon]\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateNote([]byte(tt.raw))
			if (len(errs) == 0) != tt.ok {
				t.Errorf("errs = %v", errs)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	if errs := ValidateConfig([]byte("caps:\n  hard_max_chars: 9000\n")); len(errs) != 0 {
		t.Errorf("errs = %v", errs)
	}
	if errs := ValidateConfig([]byte("bogus: 1\n")); len(errs) == 0 {
		t.Error("want error for unknown key")
	}
	if errs := ValidateConfig([]byte("max_tags: 0\n")); len(errs) == 0 {
		t.Error("want error for max_tags 0")
	}
}
