package config

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/lucawalz/strata/schema"
	"go.yaml.in/yaml/v4"
)

const (
	defaultTargetMinChars = 700
	defaultTargetMaxChars = 3500
	defaultHardMaxChars   = 6900
	defaultMaxTags        = 8
	defaultAllowedSigners = ".github/allowed_signers"
)

type Caps struct {
	TargetMinChars int `yaml:"target_min_chars"`
	TargetMaxChars int `yaml:"target_max_chars"`
	HardMaxChars   int `yaml:"hard_max_chars"`
}

type Config struct {
	Caps            Caps     `yaml:"caps"`
	MaxTags         int      `yaml:"max_tags"`
	GovernancePaths []string `yaml:"governance_paths"`
	AllowedSigners  string   `yaml:"allowed_signers"`
}

func Default() Config {
	return Config{
		Caps: Caps{
			TargetMinChars: defaultTargetMinChars,
			TargetMaxChars: defaultTargetMaxChars,
			HardMaxChars:   defaultHardMaxChars,
		},
		MaxTags:         defaultMaxTags,
		GovernancePaths: []string{".github/", "AGENTS.md", "CLAUDE.md", "CLAUDE.local.md", ".claude/", ".mcp.json", "justfile", "scripts/", "strata.yml"},
		AllowedSigners:  defaultAllowedSigners,
	}
}

func Load(raw []byte) (Config, error) {
	cfg := Default()
	if len(bytes.TrimSpace(raw)) == 0 {
		return cfg, nil
	}
	if errs := schema.ValidateConfig(raw); len(errs) > 0 {
		return Config{}, fmt.Errorf("strata.yml: %w", errors.Join(errs...))
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("strata.yml: %w", err)
	}
	c := cfg.Caps
	if c.TargetMinChars > c.TargetMaxChars || c.TargetMaxChars > c.HardMaxChars {
		return Config{}, fmt.Errorf("strata.yml: caps must satisfy target_min_chars <= target_max_chars <= hard_max_chars, got %d, %d, %d",
			c.TargetMinChars, c.TargetMaxChars, c.HardMaxChars)
	}
	return cfg, nil
}
