# 5. Detect personal data with validated identifiers

Date: 2026-11-20

## Status

Accepted

## Context

Notes must not carry bank, card, tax, pension or identity document numbers. Plain digit patterns flag years, phone numbers and code tokens. No maintained Go library for these identifiers has meaningful adoption, and the maintained prompt-injection scanners are runtime red-teaming tools, not file checks.

## Decision

The scanner matches candidates and keeps only those that pass the identifier's own check: IBAN mod 97-10 with the country length, Luhn for card numbers of 12 to 19 digits, the ELSTER rules for the Steuer-ID, the VKVV check digit for the pension number, and the ICAO 9303 check digit for ID cards and passports. A 9-character ID serial without a check digit is reported only after a label such as "Ausweis" or "passport" on the same line. Health insurance numbers are left out. The validators use the Go standard library.

## Consequences

- Few false positives, each validator tested against published sample numbers.
- An unlabeled ID serial and identifiers outside the list are missed.
- New identifier types need new code.
