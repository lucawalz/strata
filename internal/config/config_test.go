package config

import (
	"reflect"
	"testing"
)

func TestLoadEmptyIsDefault(t *testing.T) {
	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("got %+v", got)
	}
}

func TestLoadKeepsOtherDefaults(t *testing.T) {
	got, err := Load([]byte("caps:\n  hard_max_chars: 9000\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.Caps.HardMaxChars = 9000
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestLoadRejects(t *testing.T) {
	for name, raw := range map[string]string{
		"hard below target max": "caps:\n  hard_max_chars: 5000\n",
		"target min above max":  "caps:\n  target_min_chars: 7000\n",
		"unknown key":           "bogus: 1\n",
		"bad yaml":              "caps: [\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load([]byte(raw)); err == nil {
				t.Error("want error")
			}
		})
	}
}
