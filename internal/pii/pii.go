package pii

import (
	"bytes"
	"regexp"
	"strings"
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
	cardMinDigits = 13
	cardMaxDigits = 19
	steuerLen     = 11
	rvnrLen       = 12
	idDocBase     = 9
)

var (
	rvnrWeights = [12]int{2, 1, 2, 5, 7, 1, 2, 1, 2, 1, 2, 1}
	icaoWeights = [3]int{7, 3, 1}

	ibanRe   = regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}(?: ?[A-Z0-9]{4}){2,7}(?: ?[A-Z0-9]{1,4})?\b`)
	cardRe   = regexp.MustCompile(`\b[2-6][0-9]{2,3}(?:[ -]?[0-9]{2,6}){2,5}\b`)
	steuerRe = regexp.MustCompile(`\b[0-9]{11}\b`)
	rvnrRe   = regexp.MustCompile(`\b[0-9]{8}[A-Z][0-9]{3}\b`)
	idDocRe  = regexp.MustCompile(`\b[CFGHJKLMNPRTVWXY][0-9CFGHJKLMNPRTVWXYZ]{7,8}[0-9]\b`)
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
	if !strings.ContainsRune("LMNPRTVWXYCFGHJK", rune(body[0])) || check < '0' || check > '9' {
		return false
	}
	sum := 0
	for i := 0; i < len(body); i++ {
		v, ok := icaoValue(body[i])
		if !ok || !strings.ContainsRune("0123456789CFGHJKLMNPRTVWXYZ", rune(body[i])) {
			return false
		}
		sum += v * icaoWeights[i%3]
	}
	return sum%10 == int(check-'0')
}

func cardValid(s string) bool {
	digits := strings.NewReplacer(" ", "", "-", "").Replace(s)
	return len(digits) >= cardMinDigits && len(digits) <= cardMaxDigits && Luhn(digits)
}

func Scan(text []byte) []Match {
	var out []Match
	add := func(kind string, re *regexp.Regexp, valid func(string) bool) {
		for _, loc := range re.FindAllIndex(text, -1) {
			if valid(string(text[loc[0]:loc[1]])) {
				out = append(out, position(text, loc[0], kind))
			}
		}
	}
	add(KindIBAN, ibanRe, IBAN)
	add(KindCard, cardRe, cardValid)
	add(KindSteuerID, steuerRe, SteuerID)
	add(KindRVNR, rvnrRe, RVNR)
	add(KindIDDocument, idDocRe, IDDocument)
	return out
}

func position(text []byte, off int, kind string) Match {
	before := text[:off]
	line := bytes.Count(before, []byte("\n")) + 1
	start := bytes.LastIndexByte(before, '\n') + 1
	return Match{Kind: kind, Line: line, Col: len([]rune(string(before[start:]))) + 1}
}
