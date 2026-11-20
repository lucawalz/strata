package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucawalz/strata/internal/calibrate"
	"github.com/lucawalz/strata/internal/check"
	"github.com/lucawalz/strata/internal/config"
	"github.com/lucawalz/strata/internal/gitx"
	"github.com/lucawalz/strata/internal/skillsync"
	"github.com/lucawalz/strata/skills"
)

const (
	exitOK       = 0
	exitFindings = 1
	exitUsage    = 2

	formatText   = "text"
	formatGitHub = "github"

	anthropicURL = "https://api.anthropic.com"
	apiKeyEnv    = "ANTHROPIC_API_KEY"
	usage        = "usage: strata check tree [--root DIR] [--format text|github]\n       strata check diff --base SHA --head SHA [--root DIR] [--format text|github]\n       strata audit --before SHA --after SHA [--root DIR] [--format text|github]\n       strata calibrate --model MODEL DIR\n       strata skills sync --to DIR"
)

var githubEscapes = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
var githubPropEscapes = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")

func fail(w io.Writer, format string, args ...any) int {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
	return exitUsage
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return fail(stderr, "%s", usage)
	}
	cmd, rest := args[0], args[1:]
	if cmd == "calibrate" {
		return runCalibrate(rest, stdout, stderr)
	}
	if cmd == "skills" {
		return runSkills(rest, stdout, stderr)
	}
	var sub string
	switch cmd {
	case "check":
		if len(rest) == 0 {
			return fail(stderr, "%s", usage)
		}
		sub, rest = rest[0], rest[1:]
		if sub != "tree" && sub != "diff" {
			return fail(stderr, "%s", usage)
		}
	case "audit":
		sub = "audit"
	default:
		return fail(stderr, "%s", usage)
	}

	flags := flag.NewFlagSet(sub, flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	format := flags.String("format", formatText, "text or github")
	var base, head, before, after *string
	switch sub {
	case "diff":
		base, head = flags.String("base", "", "base revision"), flags.String("head", "", "head revision")
	case "audit":
		before, after = flags.String("before", "", "previous tip"), flags.String("after", "", "new tip")
	}
	if err := flags.Parse(rest); err != nil || flags.NArg() > 0 {
		return exitUsage
	}
	if *format != formatText && *format != formatGitHub {
		return fail(stderr, "unknown format %q", *format)
	}

	var findings []check.Finding
	var err error
	var rev string
	switch sub {
	case "tree":
		var cfg config.Config
		if cfg, err = loadConfig(*root); err == nil {
			findings, err = check.Tree(check.Repo{Root: *root, Config: cfg})
		}
	case "diff":
		if *base == "" || *head == "" {
			return fail(stderr, "--base and --head are required")
		}
		rev = *base
		findings, err = check.Diff(check.Repo{Root: *root}, *base, *head)
	case "audit":
		if *before == "" || *after == "" {
			return fail(stderr, "--before and --after are required")
		}
		rev = *before
		findings, err = check.Audit(check.Repo{Root: *root}, *before, *after)
	}
	if errors.Is(err, gitx.ErrMissingBase) {
		return fail(stderr, "base %s not found, fetch full history", rev)
	}
	if err != nil {
		return fail(stderr, "%v", err)
	}
	return report(findings, *format, stdout, stderr)
}

func runCalibrate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("calibrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	model := flags.String("model", "", "model name")
	if err := flags.Parse(args); err != nil || flags.NArg() != 1 || *model == "" {
		return fail(stderr, "%s", usage)
	}
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return fail(stderr, "%s is not set", apiKeyEnv)
	}
	var files []string
	err := filepath.WalkDir(flags.Arg(0), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".md" {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		return fail(stderr, "%v", err)
	}
	res, err := calibrate.Run(context.Background(), http.DefaultClient, anthropicURL, apiKey, *model, files)
	if err != nil {
		return fail(stderr, "%v", err)
	}
	caps := res.ProposedCaps()
	_, _ = fmt.Fprintf(stdout, "notes %d chars %d tokens %d chars_per_token %.2f\ntarget_min_chars %d\ntarget_max_chars %d\nhard_max_chars %d\n",
		res.Notes, res.Chars, res.Tokens, res.CharsPerToken, caps[0], caps[1], caps[2])
	return exitOK
}

func runSkills(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "sync" {
		return fail(stderr, "%s", usage)
	}
	flags := flag.NewFlagSet("skills sync", flag.ContinueOnError)
	flags.SetOutput(stderr)
	to := flags.String("to", "", "target skills directory")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() > 0 || *to == "" {
		return fail(stderr, "%s", usage)
	}
	if err := skillsync.Sync(skills.FS, *to); err != nil {
		return fail(stderr, "%v", err)
	}
	_, _ = fmt.Fprintf(stdout, "synced skills to %s\n", *to)
	return exitOK
}

func loadConfig(root string) (config.Config, error) {
	raw, err := os.ReadFile(filepath.Join(root, "strata.yml"))
	if errors.Is(err, fs.ErrNotExist) {
		return config.Default(), nil
	}
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := config.Load(raw)
	if err != nil {
		return config.Default(), nil
	}
	return cfg, nil
}

func report(findings []check.Finding, format string, stdout, stderr io.Writer) int {
	code := exitOK
	summary := os.Getenv("GITHUB_STEP_SUMMARY")
	for _, f := range findings {
		if f.Level == check.Error {
			code = exitFindings
		}
		if f.Check == check.CheckGovernanceSummary && summary != "" {
			if err := appendFile(summary, f.Message+"\n"); err != nil {
				return fail(stderr, "%v", err)
			}
			if format == formatGitHub {
				continue
			}
		}
		if format == formatGitHub {
			_, _ = fmt.Fprintln(stdout, githubLine(f))
		} else {
			_, _ = fmt.Fprintln(stdout, textLine(f))
		}
	}
	return code
}

func appendFile(path, s string) error {
	fh, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = fh.WriteString(s)
	return errors.Join(err, fh.Close())
}

func textLine(f check.Finding) string {
	loc := ""
	if f.Path != "" {
		loc = " " + f.Path
		loc += fmt.Sprintf(":%d", max(f.Line, 1))
	}
	return fmt.Sprintf("%s %s%s %s", strings.ToUpper(f.Level.String()), f.Check, loc, f.Message)
}

func githubLine(f check.Finding) string {
	props := []string{}
	if f.Path != "" {
		props = append(props, "file="+githubPropEscapes.Replace(f.Path))
		props = append(props, fmt.Sprintf("line=%d", max(f.Line, 1)))
	}
	props = append(props, "title="+githubPropEscapes.Replace(f.Check))
	kind := "notice"
	switch f.Level {
	case check.Error:
		kind = "error"
	case check.Warning:
		kind = "warning"
	}
	return fmt.Sprintf("::%s %s::%s", kind, strings.Join(props, ","), githubEscapes.Replace(f.Message))
}
