package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucawalz/strata/internal/check"
	"github.com/lucawalz/strata/internal/config"
	"github.com/lucawalz/strata/internal/gitx"
)

const (
	exitOK       = 0
	exitFindings = 1
	exitUsage    = 2

	formatText   = "text"
	formatGitHub = "github"
	usage        = "usage: strata check tree [--root DIR] [--format text|github]\n       strata check diff --base SHA --head SHA [--root DIR] [--format text|github]\n       strata audit --before SHA --after SHA [--root DIR] [--format text|github]"
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
