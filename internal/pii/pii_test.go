package pii

import (
	"fmt"
	"strings"
	"testing"
)

func TestIBAN(t *testing.T) {
	cases := map[string]bool{
		"DE89370400440532013000":      true,
		"DE89 3704 0044 0532 0130 00": true,
		"DE89370400440532013001":      false,
		"DE8937040044053201300":       false,
	}
	for in, want := range cases {
		if got := IBAN(in); got != want {
			t.Errorf("IBAN(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestLuhn(t *testing.T) {
	if !Luhn("4111111111111111") || Luhn("4111111111111112") {
		t.Error("luhn vectors failed")
	}
}

func TestSteuerID(t *testing.T) {
	if got := steuerCheckDigit("0247629135"); got != 8 {
		t.Fatalf("check digit = %d, want 8", got)
	}
	with := func(first10 string) string { return fmt.Sprintf("%s%d", first10, steuerCheckDigit(first10)) }
	cases := map[string]bool{
		"02476291358":      false,
		with("1234567891"): true,
		with("1112345678"): false,
		with("1234567890"): false,
	}
	for in, want := range cases {
		if got := SteuerID(in); got != want {
			t.Errorf("SteuerID(%q) = %v, want %v", in, got, want)
		}
	}
}

func rvnrWith(area, birth string, letter byte, serial string) string {
	digits := area + birth + twoDigits(int(letter-'A')+1) + serial
	sum := 0
	for i, c := range digits {
		p := int(c-'0') * rvnrWeights[i]
		sum += p/10 + p%10
	}
	return fmt.Sprintf("%s%s%c%s%d", area, birth, letter, serial, sum%10)
}

func TestRVNR(t *testing.T) {
	good := rvnrWith("65", "170839", 'J', "00")
	if !RVNR(good) {
		t.Errorf("RVNR(%q) = false", good)
	}
	bad := good[:11] + string(rune('0'+(int(good[11]-'0')+1)%10))
	if RVNR(bad) {
		t.Errorf("RVNR(%q) = true", bad)
	}
}

func TestIDDocument(t *testing.T) {
	if !IDDocument("L898902C3") || IDDocument("L898902C4") {
		t.Error("id document vectors failed")
	}
}

func TestScanIgnoresLookalikes(t *testing.T) {
	for _, in := range []string{"ISBN 9780306406157", "arXiv 2604.15597", "2026-11-20", "+49 30 1234567", "doi:10.1000/182"} {
		if m := Scan([]byte(in)); len(m) != 0 {
			t.Errorf("Scan(%q) = %v", in, m)
		}
	}
}

func TestScanFindsAndLocates(t *testing.T) {
	text := "x\nkonto DE89 3704 0044 0532 0130 00\n" + strings.Repeat(" ", 2) + "4111 1111 1111 1111\n"
	got := Scan([]byte(text))
	want := []Match{{KindIBAN, 2, 7}, {KindCard, 3, 3}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("Scan = %v, want %v", got, want)
	}
}
