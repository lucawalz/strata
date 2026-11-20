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

func luhnWith(prefix string) string {
	for d := 0; d < 10; d++ {
		if c := fmt.Sprintf("%s%d", prefix, d); Luhn(c) {
			return c
		}
	}
	return ""
}

func TestScanAdjacentGroups(t *testing.T) {
	card12 := luhnWith("41111111111")
	cases := []struct {
		in   string
		want []Match
	}{
		{"2026-11-20 4111 1111 1111 1111", []Match{{KindCard, 1, 12}}},
		{"4111 1111 1111 1111 2026", []Match{{KindCard, 1, 1}}},
		{"AT611904300234573201 2026", []Match{{KindIBAN, 1, 1}}},
		{card12, []Match{{KindCard, 1, 1}}},
		{"L01X00T47", []Match{{KindIDDocument, 1, 1}}},
		{"CHRYSTHMY", nil},
	}
	for _, c := range cases {
		if got := Scan([]byte(c.in)); fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("Scan(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestScanPrintedForms(t *testing.T) {
	steuer := fmt.Sprintf("1234567891%d", steuerCheckDigit("1234567891"))
	rvnr := rvnrWith("65", "170839", 'J', "00")
	cases := []struct {
		in   string
		want []Match
	}{
		{steuer[:2] + " " + steuer[2:5] + " " + steuer[5:8] + " " + steuer[8:], []Match{{KindSteuerID, 1, 1}}},
		{rvnr[:2] + " " + rvnr[2:8] + " " + rvnr[8:9] + " " + rvnr[9:], []Match{{KindRVNR, 1, 1}}},
		{"de89 3704 0044 0532 0130 00", []Match{{KindIBAN, 1, 1}}},
		{"Konto:\u00a0DE89\u00a03704\u00a00044\u00a00532\u00a00130\u00a000", []Match{{KindIBAN, 1, 8}}},
		{"DE89\u202f3704\u202f0044\u202f0532\u202f0130\u202f00", []Match{{KindIBAN, 1, 1}}},
		{"DE89-3704-0044-0532-0130-00", []Match{{KindIBAN, 1, 1}}},
		{"4111.1111.1111.1111", []Match{{KindCard, 1, 1}}},
		{"4111/1111/1111/1111", []Match{{KindCard, 1, 1}}},
	}
	for _, c := range cases {
		if got := Scan([]byte(c.in)); fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("Scan(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestScanSorted(t *testing.T) {
	text := "4111 1111 1111 1111\nDE89370400440532013000 and 4111111111111111\n"
	got := Scan([]byte(text))
	want := []Match{{KindCard, 1, 1}, {KindIBAN, 2, 1}, {KindCard, 2, 28}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("Scan = %v, want %v", got, want)
	}
}
