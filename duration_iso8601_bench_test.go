// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"fmt"
	"io"
	"testing"
	"time"
)

// BenchmarkParseISO8601Duration measures the single-pass parser and pins it against the standard library.
//
// The parser is designed to be zero-alloc and to stay in the same order of magnitude as time.ParseDuration.
// The "strict-time-only" and "stdlib-ParseDuration" sub-benchmarks parse the *same* durations (ISO- vs Go-encoded)
// for an apples-to-apples comparison; the calendar designators (Y/M/W) have no time.Duration-string equivalent.
func BenchmarkParseISO8601Duration(b *testing.B) {
	// Full calendar+time inputs exercising every designator.
	isoInputs := []string{
		"P1Y2M3DT4H5M6S",
		"P4DT12H30M5S",
		"P1DT12H",
		"P2W",
		"PT1H30M",
		"PT0S",
	}
	b.Run("strict", benchmarkParseISO(isoInputs, DurationStrict{}.isoDurationConfig()))
	b.Run("lenient", benchmarkParseISO(isoInputs, DurationLenient{}.isoDurationConfig()))

	// Fractional inputs exercise the float path; fractions are lenient-only.
	fracInputs := []string{
		"PT0.5S",
		"PT1H30M0.25S",
		"PT23.999999999S",
	}
	b.Run("lenient-fraction", benchmarkParseISO(fracInputs, DurationLenient{}.isoDurationConfig()))

	// Apples-to-apples yardstick: the same time-only durations, ISO- vs Go-encoded.
	isoTimeInputs := []string{
		"PT4H5M6S",
		"PT12H30M5S",
		"PT1H30M",
		"PT0S",
	}
	goInputs := []string{
		"4h5m6s",
		"12h30m5s",
		"1h30m",
		"0s",
	}
	b.Run("strict-time-only", benchmarkParseISO(isoTimeInputs, DurationStrict{}.isoDurationConfig()))
	b.Run("stdlib-ParseDuration", func(b *testing.B) {
		var (
			d   time.Duration
			err error
			i   int
		)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			d, err = time.ParseDuration(goInputs[i%len(goInputs)])
			i++
		}
		fmt.Fprintln(io.Discard, d, err)
	})
}

// BenchmarkISODurationFormat measures the canonical marshaler (isoFormat).
func BenchmarkISODurationFormat(b *testing.B) {
	durations := []time.Duration{
		4*24*time.Hour + 12*time.Hour + 30*time.Minute + 5*time.Second,
		90 * time.Minute,
		time.Second + 500*time.Millisecond,
		-(time.Hour + 30*time.Minute),
		0,
	}
	var (
		s string
		i int
	)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s = isoFormat(durations[i%len(durations)])
		i++
	}
	fmt.Fprintln(io.Discard, s)
}

// benchmarkParseISO benchmarks parseISO8601Duration over a rotating set of inputs under a single policy.
func benchmarkParseISO(inputs []string, cfg isoDurationConfig) func(*testing.B) {
	return func(b *testing.B) {
		var (
			d   time.Duration
			err error
			i   int
		)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			d, err = parseISO8601Duration(inputs[i%len(inputs)], cfg)
			i++
		}
		fmt.Fprintln(io.Discard, d, err)
	}
}
