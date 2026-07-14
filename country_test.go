// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"iter"
	"slices"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

func TestCountry(t *testing.T) {
	for tc := range countryCases() {
		t.Run(tc.Desc, func(t *testing.T) {
			if tc.Valid {
				require.TrueT(t, IsCountry(tc.Input))

				cur, err := ParseCountry(tc.Input)
				require.NoError(t, err)

				b, err := cur.MarshalText()
				require.NoError(t, err)
				require.NotEmpty(t, b)

				var other Country
				require.NoError(t, other.UnmarshalText(b))

				require.EqualT(t, cur.String(), other.String())

				return
			}

			require.FalseT(t, IsCountry(tc.Input))
		})
	}
}

type countryCase struct {
	Desc  string
	Input string
	Valid bool
}

func countryCases() iter.Seq[countryCase] {
	return slices.Values([]countryCase{
		{Desc: "USA", Input: "USA", Valid: true},
		{Desc: "FR-alpha-2", Input: "FR", Valid: true},
		{Desc: "GBR-alpha-3", Input: "GBR", Valid: true},
		{Desc: "unknown-alpha-2", Input: "FF", Valid: false},
		{Desc: "unknown-alpha-3", Input: "XYZ", Valid: false},
		{Desc: "invalid (empty)", Input: "", Valid: false},
		{Desc: "invalid (1)", Input: "X", Valid: false},
		{Desc: "invalid (4)", Input: "ABCD", Valid: false},
	})
}
