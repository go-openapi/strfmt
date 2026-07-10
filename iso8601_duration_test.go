// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestParseISO8601Duration_Success(t *testing.T) {
	testCases := []struct {
		input    string
		expected time.Duration
	}{
		// Years
		{"P1Y", 365 * 24 * time.Hour},
		{"P2Y", 2 * 365 * 24 * time.Hour},
		// Months
		{"P1M", 30 * 24 * time.Hour},
		{"P3M", 3 * 30 * 24 * time.Hour},
		// Weeks
		{"P1W", 7 * 24 * time.Hour},
		{"P4W", 4 * 7 * 24 * time.Hour},
		// Days
		{"P1D", 24 * time.Hour},
		{"P5D", 5 * 24 * time.Hour},
		// Hours
		{"PT1H", time.Hour},
		{"PT12H", 12 * time.Hour},
		// Minutes
		{"PT1M", time.Minute},
		{"PT45M", 45 * time.Minute},
		// Seconds
		{"PT1S", time.Second},
		{"PT30S", 30 * time.Second},

		// Combinations
		{"P1Y2M3W4DT5H6M7S", (1*365+2*30+3*7+4)*24*time.Hour + 5*time.Hour + 6*time.Minute + 7*time.Second},
		{"P1DT1H", 25 * time.Hour},
		{"PT1H30M", time.Hour + 30*time.Minute},
		{"P1YT1S", 365*24*time.Hour + time.Second},

		// Signs
		{"+P1D", 24 * time.Hour},
		{"-P1D", -24 * time.Hour},
		{"-PT1H30M", -(time.Hour + 30*time.Minute)},

		// Decimal Fractions (dot and comma)
		{"PT1.5S", time.Second + 500*time.Millisecond},
		{"PT1,5S", time.Second + 500*time.Millisecond},
		{"P1.5D", 36 * time.Hour},
		{"P1,5D", 36 * time.Hour},
		{"PT0.5M", 30 * time.Second},
		{"PT0.25H", 15 * time.Minute},
		{"PT0.001S", time.Millisecond},
		{"PT0.000001S", time.Microsecond},
		{"PT0.000000001S", time.Nanosecond},

		// Extremely precise fractional digits
		{"PT0.123456789S", 123456789 * time.Nanosecond},

		// Boundary Limits (Max/Min Duration)
		{"PT9223372036.854775807S", time.Duration(math.MaxInt64)},
		{"-PT9223372036.854775808S", time.Duration(math.MinInt64)},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ParseISO8601Duration(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseISO8601Duration_Failure(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "Empty duration",
			input:       "",
			expectedErr: errEmptyDuration,
		},
		{
			name:        "Empty duration with spaces",
			input:       "   ",
			expectedErr: errEmptyDuration,
		},
		{
			name:        "Missing P prefix",
			input:       "1Y",
			expectedErr: errInvalidPStart,
		},
		{
			name:        "Missing P prefix with time",
			input:       "T1H",
			expectedErr: errInvalidPStart,
		},
		{
			name:        "Empty after P",
			input:       "P",
			expectedErr: errEmptyAfterP,
		},
		{
			name:        "Empty after P with sign",
			input:       "-P",
			expectedErr: errEmptyAfterP,
		},
		{
			name:        "Empty after P with plus sign",
			input:       "+P",
			expectedErr: errEmptyAfterP,
		},
		{
			name:        "Empty time part after T",
			input:       "P1DT",
			expectedErr: errEmptyTime,
		},
		{
			name:        "Empty time part after T only",
			input:       "PT",
			expectedErr: errEmptyTime,
		},
		{
			name:        "Multiple decimal separators double dot",
			input:       "PT1.5.5S",
			expectedErr: errInvalidDec,
		},
		{
			name:        "Multiple decimal separators double comma",
			input:       "PT1,5,5S",
			expectedErr: errInvalidDec,
		},
		{
			name:        "Multiple decimal separators mixed",
			input:       "PT1.5,5S",
			expectedErr: errInvalidDec,
		},
		{
			name:        "Decimal fraction on non-least significant unit date",
			input:       "P1.5Y1M",
			expectedErr: errFraction,
		},
		{
			name:        "Decimal fraction on non-least significant unit month",
			input:       "P1Y1.5M1W",
			expectedErr: errFraction,
		},
		{
			name:        "Decimal fraction on non-least significant unit week",
			input:       "P1Y1M1.5W1D",
			expectedErr: errFraction,
		},
		{
			name:        "Decimal fraction on non-least significant unit day",
			input:       "P1.5DT1H",
			expectedErr: errFraction,
		},
		{
			name:        "Decimal fraction on non-least significant unit hour",
			input:       "PT1.5H30M",
			expectedErr: errFraction,
		},
		{
			name:        "Decimal fraction on non-least significant unit minute",
			input:       "PT1H1.5M30S",
			expectedErr: errFraction,
		},
		{
			name:        "Out of order units Y and M",
			input:       "P1M2Y",
			expectedErr: ErrFormat, // unrecognized trailing character
		},
		{
			name:        "Out of order units H and M",
			input:       "PT1M2H",
			expectedErr: ErrFormat,
		},
		{
			name:        "Out of order units date and time",
			input:       "PT1HP1D",
			expectedErr: ErrFormat,
		},
		{
			name:        "Unrecognized trailing character",
			input:       "P1YZ",
			expectedErr: ErrFormat,
		},
		{
			name:        "Missing T for time unit",
			input:       "P1D1H",
			expectedErr: ErrFormat,
		},
		{
			name:        "Internal spaces",
			input:       "P 1Y",
			expectedErr: ErrFormat,
		},
		{
			name:        "Internal spaces with T",
			input:       "P1Y T1H",
			expectedErr: ErrFormat,
		},
		{
			name:        "Integer overflow",
			input:       "P999999999999999999999999Y",
			expectedErr: errOverflow,
		},
		{
			name:        "Multiplication overflow in parseOptionalField",
			input:       "PT20000000000S",
			expectedErr: errOverflow,
		},
		{
			name:        "Positive overflow max duration + 1",
			input:       "PT9223372036.854775808S",
			expectedErr: errOverflow,
		},
		{
			name:        "Negative overflow min duration - 1",
			input:       "-PT9223372036.854775809S",
			expectedErr: errOverflow,
		},
		{
			name:        "Total sum overflow",
			input:       "P100000000Y",
			expectedErr: errOverflow,
		},
		{
			name:        "Negative total sum overflow",
			input:       "-PT1388888H5000000000S",
			expectedErr: errOverflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseISO8601Duration(tc.input)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.expectedErr), "expected error wrapped by %v, got %v", tc.expectedErr, err)
		})
	}
}

func TestParseISO8601Duration_PrecisionLimit(t *testing.T) {
	// Values beyond 18 fractional digits are truncated because scale caps at maxScale.
	// PT0.9223372036854775809S has 19 digits.
	// scale caps at 10^18, so we get 0.922337203685477580 * 1s, which is 922337203685477580ns = 922337203ms 685us 477ns.
	// Let's verify that it parses and is truncated exactly to 922337203685477580ns.
	d, err := ParseISO8601Duration("PT0.9223372036854775809S")
	require.NoError(t, err)
	assert.Equal(t, 922337203*time.Nanosecond, d)
}
