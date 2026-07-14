// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestParseISO8601Duration_Options exercises the explicit programmatic parse path: each option relaxes exactly one
// strict rule, and the strict default rejects what the matching option accepts.
func TestParseISO8601Duration_Options(t *testing.T) {
	cases := []struct {
		name string
		in   string
		opt  ISODurationOption
		want time.Duration
	}{
		{"fractions", "PT0.5S", WithISOFractions(), 500 * time.Millisecond},
		{"sign", "-P1D", WithISOSign(), -24 * time.Hour},
		{"space", " P1D ", WithISOSpace(), 24 * time.Hour},
		{"relaxed-anchoring", "P1Y2D", WithISORelaxedAnchoring(), isoYearDur + 2*24*time.Hour},
		{"week-combinable", "P1WT1H", WithISOWeekCombinable(), 7*24*time.Hour + time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// strict default rejects it
			_, err := ParseISO8601Duration(tc.in)
			require.Error(t, err, "strict default must reject %q", tc.in)

			// the matching option accepts it
			got, err := ParseISO8601Duration(tc.in, tc.opt)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	// WithISOLenient relaxes every knob at once.
	got, err := ParseISO8601Duration(" -P1Y2WT0.5S ", WithISOLenient())
	require.NoError(t, err)
	assert.Equal(t, -(isoYearDur + 2*7*24*time.Hour + 500*time.Millisecond), got)

	// ordering is never relaxed, even under full leniency.
	_, err = ParseISO8601Duration("P2D1Y", WithISOLenient())
	require.Error(t, err, "out-of-order components stay invalid")
}

func TestDurationISO8601_ValueSemantics(t *testing.T) {
	a := DurationISO8601(time.Hour)
	b := DurationISO8601(time.Hour)
	c := DurationISO8601(2 * time.Hour)
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))

	// DeepCopy yields an independent, equal value.
	cp := a.DeepCopy()
	require.NotNil(t, cp)
	assert.True(t, a.Equal(*cp))
	*cp = c
	assert.True(t, a.Equal(b), "mutating the copy must not touch the original")

	// DeepCopy of a nil pointer is nil.
	var nilp *DurationISO8601
	assert.Nil(t, nilp.DeepCopy())
}

// isoYearDur is the fixed-length ISO year as a time.Duration (365 days), for readable expectations above.
const isoYearDur = 365 * 24 * time.Hour
