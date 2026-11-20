package inject

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	LevelError   = "error"
	LevelWarning = "warning"

	base64Threshold = 120
	bom             = 0xFEFF
	zwj             = 0x200D
	vs16            = 0xFE0F
	skinToneFirst   = 0x1F3FB
	skinToneLast    = 0x1F3FF
)

type Match struct {
	Kind  string
	Line  int
	Col   int
	Level string
}

type pattern struct {
	kind string
	re   *regexp.Regexp
}

var patterns = []pattern{
	{"html-comment", regexp.MustCompile("<!--")},
	{"hidden-html", regexp.MustCompile(`(?i)<[a-z][^>]*(\bhidden\b|display\s*:\s*none|visibility\s*:\s*hidden)`)},
	{"remote-image", regexp.MustCompile(`(?i)!\[[^\]]*\]\(\s*(https?:)?//`)},
	{"base64", regexp.MustCompile(fmt.Sprintf(`[A-Za-z0-9+/]{%d,}={0,2}`, base64Threshold))},
}

var instructionPhrases = []string{
	"ignore all previous instructions",
	"ignore previous instructions",
	"ignore the above instructions",
	"disregard all previous instructions",
	"disregard previous instructions",
	"forget all previous instructions",
	"ignoriere alle vorherigen anweisungen",
	"ignoriere alle bisherigen anweisungen",
	"ignoriere die vorherigen anweisungen",
	"vergiss alle vorherigen anweisungen",
	"missachte alle vorherigen anweisungen",
}

func hiddenKind(r rune) string {
	switch {
	case r >= 0xE0000 && r <= 0xE007F:
		return "unicode-tag"
	case r >= 0x200B && r <= 0x200D, r == 0x2060, r == bom:
		return "zero-width"
	case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069,
		r == 0x200E, r == 0x200F, r == 0x061C:
		return "bidi"
	}
	return ""
}

func emoji(r rune) bool {
	return unicode.Is(unicode.So, r) || (r >= skinToneFirst && r <= skinToneLast)
}

func emojiJoin(runes []rune, i int) bool {
	prev := i - 1
	if prev > 0 && runes[prev] == vs16 {
		prev--
	}
	return prev >= 0 && i+1 < len(runes) && emoji(runes[prev]) && emoji(runes[i+1])
}

func Scan(text []byte) []Match {
	var out []Match
	for i, line := range bytes.Split(text, []byte("\n")) {
		out = scanLine(out, string(line), i+1, i == 0)
	}
	return out
}

func scanLine(out []Match, line string, n int, first bool) []Match {
	add := func(kind string, col int) {
		out = append(out, Match{Kind: kind, Line: n, Col: col, Level: LevelError})
	}
	runes := []rune(line)
	for i, r := range runes {
		if r == bom && first && i == 0 {
			continue
		}
		if r == zwj && emojiJoin(runes, i) {
			continue
		}
		if k := hiddenKind(r); k != "" {
			add(k, i+1)
		}
	}
	for _, p := range patterns {
		for _, loc := range p.re.FindAllStringIndex(line, -1) {
			add(p.kind, utf8.RuneCountInString(line[:loc[0]])+1)
		}
	}
	lower := strings.ToLower(line)
	for _, phrase := range instructionPhrases {
		if off := strings.Index(lower, phrase); off >= 0 {
			add("instruction", utf8.RuneCountInString(lower[:off])+1)
		}
	}
	return mixedScript(out, line, n)
}

func mixedScript(out []Match, line string, n int) []Match {
	col, start := 0, 0
	var latin, other bool
	flush := func() {
		if latin && other {
			out = append(out, Match{Kind: "mixed-script", Line: n, Col: start, Level: LevelWarning})
		}
		latin, other = false, false
	}
	for _, r := range line {
		col++
		if !unicode.IsLetter(r) {
			flush()
			continue
		}
		if !latin && !other {
			start = col
		}
		switch {
		case unicode.Is(unicode.Latin, r):
			latin = true
		case unicode.Is(unicode.Cyrillic, r), unicode.Is(unicode.Greek, r):
			other = true
		}
	}
	flush()
	return out
}
