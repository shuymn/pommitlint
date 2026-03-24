package main

import "testing"

func BenchmarkNormalizePreset(b *testing.B) {
	raw := rawPreset{
		Rules:        validRawRules(),
		ParserPreset: validRawParserPreset(),
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := normalizePreset(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeJSONValue(b *testing.B) {
	value := rawArray(
		rawNumber(2),
		rawString("always"),
		rawArray(
			rawString("sentence-case"),
			rawString("start-case"),
			rawString("pascal-case"),
			rawString("upper-case"),
		),
	)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := normalizeJSONValue(value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeRule(b *testing.B) {
	rawRule := rawArray(rawNumber(2), rawString("always"), rawNumber(100))

	b.ReportAllocs()
	for b.Loop() {
		if _, err := normalizeRule("header-max-length", rawRule); err != nil {
			b.Fatal(err)
		}
	}
}
