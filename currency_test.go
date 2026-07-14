// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"iter"
	"slices"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

func TestCurrency(t *testing.T) {
	for tc := range currencyCases() {
		t.Run(tc.Desc, func(t *testing.T) {
			if tc.Valid {
				require.TrueT(t, IsCurrency(tc.Input))

				cur, err := ParseCurrency(tc.Input)
				require.NoError(t, err)

				b, err := cur.MarshalText()
				require.NoError(t, err)
				require.NotEmpty(t, b)

				var other Currency
				require.NoError(t, other.UnmarshalText(b))

				require.EqualT(t, cur.String(), other.String())

				return
			}

			require.FalseT(t, IsCurrency(tc.Input))
		})
	}
}

type currencyCase struct {
	Desc  string
	Input string
	Valid bool
}

func currencyCases() iter.Seq[currencyCase] {
	return slices.Values([]currencyCase{
		{Desc: "EUR", Input: "EUR", Valid: true},
		{Desc: "USD", Input: "USD", Valid: true},
		{Desc: "GBP", Input: "GBP", Valid: true},
		{Desc: "FF", Input: "FF", Valid: false},
	})
}
