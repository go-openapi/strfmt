// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"reflect"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestDurationISO8601_Registration(t *testing.T) {
	// The strict ISO 8601 duration is registered under a distinct handle; "duration" stays the human-readable type.
	assert.True(t, Default.ContainsName("duration-iso8601"))

	tpe, ok := Default.GetType("duration-iso8601")
	require.True(t, ok)
	assert.Equal(t, reflect.TypeFor[DurationISO8601](), tpe)

	// Validation goes through the strict validator.
	assert.True(t, Default.Validates("duration-iso8601", "P1DT2H"))
	assert.False(t, Default.Validates("duration-iso8601", "PT0.5S"), "strict rejects fractions")
	assert.False(t, Default.Validates("duration-iso8601", "1h30m"), "the ISO handle is not the human-readable parser")

	// Parse goes through UnmarshalText and yields a typed value.
	parsed, err := Default.Parse("duration-iso8601", "P1DT2H")
	require.NoError(t, err)
	d, ok := parsed.(*DurationISO8601)
	require.True(t, ok)
	assert.Equal(t, 26*time.Hour, time.Duration(*d))

	// The human-readable "duration" handle is untouched (non-breaking).
	dtype, ok := Default.GetType("duration")
	require.True(t, ok)
	assert.Equal(t, reflect.TypeFor[Duration](), dtype)
}

func TestDurationISO8601_SQL(t *testing.T) {
	d := DurationISO8601(26 * time.Hour)
	v, err := d.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(26*time.Hour), v)

	var back DurationISO8601
	require.NoError(t, back.Scan(int64(26*time.Hour)))
	assert.Equal(t, d, back)

	require.NoError(t, back.Scan(float64(time.Second)))
	assert.Equal(t, DurationISO8601(time.Second), back)

	require.NoError(t, back.Scan(nil))
	assert.Equal(t, DurationISO8601(0), back)

	require.Error(t, back.Scan("P1D"), "string is not a valid SQL source")
}

func TestDurationISO8601_BSON(t *testing.T) {
	// BSON is a storage boundary: it must round-trip losslessly regardless of policy — including values a strict policy
	// cannot emit on the text/JSON path (a sign, or sub-second precision).
	cases := []time.Duration{
		0,
		26 * time.Hour,
		time.Second + 500*time.Millisecond,
		-(time.Hour + 30*time.Minute),
	}
	for _, want := range cases {
		d := DurationISO8601(want)
		data, err := d.MarshalBSON()
		require.NoError(t, err, "storage marshal must never fail on a representable duration")

		var back DurationISO8601
		require.NoError(t, back.UnmarshalBSON(data))
		assert.Equal(t, want, time.Duration(back))
	}
}
