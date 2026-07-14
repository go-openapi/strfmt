// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"math"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestDurationISO8601_Values(t *testing.T) {
	strict := DurationStrict{}.isoDurationConfig()
	cases := map[string]time.Duration{
		"P1Y":            isoYearD(1),
		"P1M":            30 * 24 * time.Hour,
		"P2W":            14 * 24 * time.Hour,
		"P1D":            24 * time.Hour,
		"PT1H":           time.Hour,
		"PT1M":           time.Minute,
		"PT30S":          30 * time.Second,
		"P4DT12H30M5S":   4*24*time.Hour + 12*time.Hour + 30*time.Minute + 5*time.Second,
		"P1Y2M3DT4H5M6S": isoYearD(1) + 2*30*24*time.Hour + 3*24*time.Hour + 4*time.Hour + 5*time.Minute + 6*time.Second,
		"PT36H":          36 * time.Hour,
		"P1DT12H":        36 * time.Hour,
		"PT0S":           0,
		"P0D":            0,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := parseISO8601Duration(in, strict)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func isoYearD(n int64) time.Duration { return time.Duration(n) * 365 * 24 * time.Hour }

func TestDurationISO8601_Fractions(t *testing.T) {
	lenient := DurationLenient{}.isoDurationConfig()
	cases := map[string]time.Duration{
		"PT1.5S":         time.Second + 500*time.Millisecond,
		"PT1,5S":         time.Second + 500*time.Millisecond,
		"P1.5D":          36 * time.Hour,
		"PT0.5M":         30 * time.Second,
		"PT0.25H":        15 * time.Minute,
		"PT0.001S":       time.Millisecond,
		"PT0.000000001S": time.Nanosecond,
		"PT0.123456789S": 123456789 * time.Nanosecond,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := parseISO8601Duration(in, lenient)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

// TestDurationISO8601_Overflow guards the silent-overflow class of bug: a value past the time.Duration ceiling must
// error, never wrap to a small value.
func TestDurationISO8601_Overflow(t *testing.T) {
	lenient := DurationLenient{}.isoDurationConfig()

	mustErr := []string{
		"PT18446744073S",           // integer, way over 1<<63
		"PT18446744073.9S",         // the fraction-wrap case: must error, not 190ms
		"PT18446744073.999999999S", //
		"PT9223372037S",            // just over MaxInt64 seconds
		"PT9223372036.854775808S",  // MaxInt64 + 1
		"P999999999999999999999999Y",
		"P100000000Y",
	}
	for _, in := range mustErr {
		t.Run("err/"+in, func(t *testing.T) {
			d, err := parseISO8601Duration(in, lenient)
			assert.Errorf(t, err, "expected overflow error, got %v", d)
		})
	}

	// Exact boundaries must parse.
	maxD, err := parseISO8601Duration("PT9223372036.854775807S", lenient)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(math.MaxInt64), maxD)

	minD, err := parseISO8601Duration("-PT9223372036.854775808S", lenient)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(math.MinInt64), minD)
}

func TestDurationISO8601_RoundTrip(t *testing.T) {
	// Strict-representable values: whole-second, non-negative.
	// Canonical output must re-parse under the STRICT policy to the same value (true symmetry), including the zero-filler
	// that keeps a gap contiguous (1h5s -> PT1H0M5S).
	strictCases := map[time.Duration]string{
		0:                               "PT0S",
		time.Second:                     "PT1S",
		90 * time.Minute:                "PT1H30M",
		25 * time.Hour:                  "P1DT1H",
		36 * time.Hour:                  "P1DT12H",
		time.Hour + 5*time.Second:       "PT1H0M5S", // gap filled with 0M
		4*24*time.Hour + 30*time.Minute: "P4DT30M",
	}
	for d, want := range strictCases {
		t.Run("strict/"+want, func(t *testing.T) {
			assert.Equal(t, want, isoFormat(d))
			back, err := parseISO8601Duration(want, DurationStrict{}.isoDurationConfig())
			require.NoError(t, err)
			assert.Equal(t, d, back)
		})
	}

	// Lossless (lenient) emit for values strict cannot represent.
	lenientCases := map[time.Duration]string{
		time.Second + 500*time.Millisecond: "PT1.5S",
		-(time.Hour + 30*time.Minute):      "-PT1H30M",
	}
	for d, want := range lenientCases {
		t.Run("lenient/"+want, func(t *testing.T) {
			assert.Equal(t, want, isoFormat(d))
			back, err := parseISO8601Duration(want, DurationLenient{}.isoDurationConfig())
			require.NoError(t, err)
			assert.Equal(t, d, back)
		})
	}
}

func TestDurationISO8601_PolicyAwareEmit(t *testing.T) {
	// A strict duration holding a value it cannot represent must fail to marshal, not emit output a strict parser would
	// reject.
	subSecond := DurationISO8601(1500 * time.Millisecond)
	_, err := subSecond.MarshalText()
	require.Error(t, err, "strict must not emit sub-second precision")
	_, err = subSecond.MarshalJSON()
	require.Error(t, err)
	// String stays lossless for display.
	assert.Equal(t, "PT1.5S", subSecond.String())

	negative := DurationISO8601(-time.Hour)
	_, err = negative.MarshalText()
	require.Error(t, err, "strict must not emit a negative sign")

	// The lenient policy emits both losslessly.
	lenient := ISODuration[DurationLenient](1500 * time.Millisecond)
	b, err := lenient.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "PT1.5S", string(b))
}

func TestDurationISO8601_TypeMethods(t *testing.T) {
	// UnmarshalText applies the type's policy.
	var strict DurationISO8601
	require.NoError(t, strict.UnmarshalText([]byte("P1DT2H")))
	assert.Equal(t, 26*time.Hour, time.Duration(strict))

	require.Error(t, strict.UnmarshalText([]byte("PT0.5S")), "strict type must reject fractions")
	require.Error(t, strict.UnmarshalText([]byte("P1Y2D")), "strict type must reject gaps")

	var lenient ISODuration[DurationLenient]
	require.NoError(t, lenient.UnmarshalText([]byte("PT0.5S")))
	assert.Equal(t, 500*time.Millisecond, time.Duration(lenient))

	// JSON round-trip.
	b, err := strict.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"P1DT2H"`, string(b))

	var back DurationISO8601
	require.NoError(t, back.UnmarshalJSON(b))
	assert.Equal(t, strict, back)
}
