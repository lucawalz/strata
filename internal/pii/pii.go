package pii

import (
	"bytes"
	"cmp"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	KindIBAN       = "iban"
	KindCard       = "card"
	KindSteuerID   = "steuer-id"
	KindRVNR       = "rvnr"
	KindIDDocument = "id-document"

	ibanLenDE     = 22
	ibanMinLen    = 15
	ibanMaxLen    = 34
	ibanModulus   = 97
	cardMinDigits = 12
	cardMinGroup  = 4
	cardMaxDigits = 19
	steuerLen     = 11
	rvnrLen       = 12
	idDocBase     = 9

	separators       = "\u00a0\u202f-./"
	idPrefixAlphabet = "LMNPRTVWXYCFGHJK"
	idAlphabet       = "0123456789CFGHJKLMNPRTVWXYZ"
)

var (
	rvnrWeights = [12]int{2, 1, 2, 5, 7, 1, 2, 1, 2, 1, 2, 1}
	icaoWeights = [3]int{7, 3, 1}

	ibanRunRe    = regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}[A-Z0-9]*(?: [A-Z0-9]+)*`)
	cardRunRe    = regexp.MustCompile(`[0-9]+(?:[ -][0-9]+)*`)
	digitGroupRe = regexp.MustCompile(`[0-9]+`)
	steuerRe     = regexp.MustCompile(`\b[0-9]{2} ?[0-9]{3} ?[0-9]{3} ?[0-9]{3}\b`)
	rvnrRe       = regexp.MustCompile(`\b[0-9]{2} ?[0-9]{6} ?[A-Z] ?[0-9]{3}\b`)
	idDocRe      = regexp.MustCompile(`\b[CFGHJKLMNPRTVWXY][0-9CFGHJKLMNPRTVWXYZ]{7,8}[0-9]\b`)
)

type Match struct {
	Kind      string
	Line, Col int
}

func IBAN(s string) bool {
	s = strings.ReplaceAll(s, " ", "")
	n := len(s)
	if n < ibanMinLen || n > ibanMaxLen || (strings.HasPrefix(s, "DE") && n != ibanLenDE) {
		return false
	}
	rem := 0
	for _, c := range s[4:] + s[:4] {
		switch {
		case c >= '0' && c <= '9':
			rem = (rem*10 + int(c-'0')) % ibanModulus
		case c >= 'A' && c <= 'Z':
			rem = (rem*100 + int(c-'A') + 10) % ibanModulus
		default:
			return false
		}
	}
	return rem == 1
}

func Luhn(digits string) bool {
	sum, double := 0, false
	if digits == "" {
		return false
	}
	for i := len(digits) - 1; i >= 0; i-- {
		c := digits[i]
		if c < '0' || c > '9' {
			return false
		}
		d := int(c - '0')
		if double {
			if d *= 2; d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

func steuerCheckDigit(first10 string) int {
	product := 10
	for _, c := range first10 {
		sum := (int(c-'0') + product) % 10
		if sum == 0 {
			sum = 10
		}
		product = sum * 2 % 11
	}
	return (11 - product) % 10
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func steuerRepetitionOK(first10 string) bool {
	var counts [10]int
	for _, c := range first10 {
		counts[c-'0']++
	}
	repeated := -1
	for d, n := range counts {
		switch {
		case n > 3:
			return false
		case n >= 2:
			if repeated >= 0 {
				return false
			}
			repeated = d
		}
	}
	if repeated < 0 {
		return false
	}
	if counts[repeated] == 3 {
		return !strings.Contains(first10, strings.Repeat(string(rune('0'+repeated)), 3))
	}
	return true
}

func SteuerID(s string) bool {
	if len(s) != steuerLen || !isDigits(s) || s[0] == '0' || !steuerRepetitionOK(s[:10]) {
		return false
	}
	return steuerCheckDigit(s[:10]) == int(s[10]-'0')
}

func RVNR(s string) bool {
	if len(s) != rvnrLen || !isDigits(s[:8]) || s[8] < 'A' || s[8] > 'Z' || !isDigits(s[9:]) {
		return false
	}
	digits := s[:8] + twoDigits(int(s[8]-'A')+1) + s[9:11]
	sum := 0
	for i, c := range digits {
		p := int(c-'0') * rvnrWeights[i]
		sum += p/10 + p%10
	}
	return sum%10 == int(s[11]-'0')
}

func twoDigits(n int) string {
	return string([]byte{byte('0' + n/10), byte('0' + n%10)})
}

func icaoValue(c byte) (int, bool) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), true
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10, true
	}
	return 0, false
}

func IDDocument(s string) bool {
	if len(s) != idDocBase && len(s) != idDocBase+1 {
		return false
	}
	body, check := s[:len(s)-1], s[len(s)-1]
	if !strings.ContainsRune(idPrefixAlphabet, rune(body[0])) || check < '0' || check > '9' {
		return false
	}
	sum := 0
	for i := 0; i < len(body); i++ {
		v, ok := icaoValue(body[i])
		if !ok || strings.IndexByte(idAlphabet, body[i]) < 0 {
			return false
		}
		sum += v * icaoWeights[i%3]
	}
	return sum%10 == int(check-'0')
}

type span struct{ start, end int }

func scanCards(text []byte, add func(int)) {
	for _, run := range cardRunRe.FindAllIndex(text, -1) {
		var groups []span
		for _, g := range digitGroupRe.FindAllIndex(text[run[0]:run[1]], -1) {
			groups = append(groups, span{run[0] + g[0], run[0] + g[1]})
		}
		for i := 0; i < len(groups); {
			end := longestCard(text, groups, i)
			if end < 0 {
				i++
				continue
			}
			add(groups[i].start)
			i = end + 1
		}
	}
}

func longestCard(text []byte, groups []span, i int) int {
	for j := len(groups) - 1; j >= i; j-- {
		if j > i && !groupsWide(groups[i:j+1]) {
			continue
		}
		var digits strings.Builder
		for _, g := range groups[i : j+1] {
			digits.Write(text[g.start:g.end])
		}
		d := digits.String()
		if len(d) >= cardMinDigits && len(d) <= cardMaxDigits && d[0] >= '2' && d[0] <= '6' && Luhn(d) {
			return j
		}
	}
	return -1
}

func groupsWide(groups []span) bool {
	for _, g := range groups {
		if g.end-g.start < cardMinGroup {
			return false
		}
	}
	return true
}

func scanIBANs(text []byte, add func(int)) {
	for _, loc := range ibanRunRe.FindAllIndex(text, -1) {
		run := text[loc[0]:loc[1]]
		for end := len(run); end > 0; {
			if IBAN(string(run[:end])) {
				add(loc[0])
				break
			}
			k := bytes.LastIndexByte(run[:end], ' ')
			if k < 0 {
				break
			}
			end = k
		}
	}
}

func idSerialShape(s string) bool {
	if len(s) != idDocBase || !strings.ContainsRune(idPrefixAlphabet, rune(s[0])) {
		return false
	}
	letters, digits := false, false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
			digits = true
		case strings.IndexByte(idAlphabet, s[i]) >= 0:
			letters = true
		default:
			return false
		}
	}
	return letters && digits
}

func normalize(text []byte) ([]byte, []int) {
	norm := make([]byte, 0, len(text))
	offs := make([]int, 0, len(text))
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRune(text[i:])
		switch {
		case strings.ContainsRune(separators, r):
			norm, offs = append(norm, ' '), append(offs, i)
		case r >= 'a' && r <= 'z':
			norm, offs = append(norm, byte(r-'a'+'A')), append(offs, i)
		default:
			for k := range n {
				norm, offs = append(norm, text[i+k]), append(offs, i+k)
			}
		}
		i += n
	}
	return norm, offs
}

func unspaced(b []byte) string {
	return strings.ReplaceAll(string(b), " ", "")
}

func Scan(orig []byte) []Match {
	var out []Match
	text, offs := normalize(orig)
	collect := func(kind string) func(int) {
		return func(off int) { out = append(out, position(orig, offs[off], kind)) }
	}
	scanIBANs(text, collect(KindIBAN))
	scanCards(text, collect(KindCard))
	for _, loc := range steuerRe.FindAllIndex(text, -1) {
		if SteuerID(unspaced(text[loc[0]:loc[1]])) {
			collect(KindSteuerID)(loc[0])
		}
	}
	for _, loc := range rvnrRe.FindAllIndex(text, -1) {
		if RVNR(unspaced(text[loc[0]:loc[1]])) {
			collect(KindRVNR)(loc[0])
		}
	}
	for _, loc := range idDocRe.FindAllIndex(text, -1) {
		if s := string(text[loc[0]:loc[1]]); IDDocument(s) || idSerialShape(s) {
			collect(KindIDDocument)(loc[0])
		}
	}
	slices.SortStableFunc(out, func(a, b Match) int {
		return cmp.Or(cmp.Compare(a.Line, b.Line), cmp.Compare(a.Col, b.Col))
	})
	return out
}

func position(text []byte, off int, kind string) Match {
	before := text[:off]
	line := bytes.Count(before, []byte("\n")) + 1
	start := bytes.LastIndexByte(before, '\n') + 1
	return Match{Kind: kind, Line: line, Col: len([]rune(string(before[start:]))) + 1}
}
