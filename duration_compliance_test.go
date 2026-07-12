// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"encoding/json"
	"iter"
	"os"
	"slices"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestDurationISO8601_Compliance asserts the strict parser matches the official
// JSON Schema Test Suite for the "duration" format (RFC 3339 Appendix A).
func TestDurationISO8601_Compliance(t *testing.T) {
	for tc := range durationTestCases(t) {
		t.Run(tc.Desc, func(t *testing.T) {
			got := IsDurationISO8601(tc.Input)
			assert.Equalf(t, tc.Valid, got,
				"input %q: strict validity mismatch (%s)", tc.Input, tc.Desc)
		})
	}
}

// TestDurationISO8601_Lenient checks that DurationLenient accepts exactly the
// cases the strict grammar rejects for leniency reasons.
func TestDurationISO8601_Lenient(t *testing.T) {
	lenient := DurationLenient{}.isoDurationConfig()

	for tc := range lenientISO8601DurationCases() {
		if tc.Valid {
			t.Run("lenient/"+tc.Desc, func(t *testing.T) {
				assert.Falsef(t, IsDurationISO8601(tc.Input), "strict should reject %q", tc.Input)
				_, err := parseISO8601Duration(tc.Input, lenient)
				assert.NoErrorf(t, err, "lenient should accept %q", tc.Input)
			})

			continue
		}

		// Ordering is still enforced under leniency.
		t.Run("lenient-rejects/"+tc.Desc, func(t *testing.T) {
			_, err := parseISO8601Duration(tc.Input, lenient)
			assert.Errorf(t, err, "lenient should still reject out-of-order %q", tc.Input)
		})
	}
}

// jsonSchemaSuiteGroup mirrors the JSON Schema Test Suite file layout.
type jsonSchemaSuiteGroup struct {
	Description string `json:"description"`
	Tests       []struct {
		Description string          `json:"description"`
		Data        json.RawMessage `json:"data"`
		Valid       bool            `json:"valid"`
	} `json:"tests"`
}

type durationTestCase struct {
	Desc  string
	Input string
	Valid bool
}

func durationTestCases(t *testing.T) iter.Seq[durationTestCase] {
	t.Helper()

	return func(yield func(durationTestCase) bool) {
		raw, err := os.ReadFile("testdata/jsonschema-suite/duration.json")
		require.NoError(t, err)

		var groups []jsonSchemaSuiteGroup
		require.NoError(t, json.Unmarshal(raw, &groups))

		for _, g := range groups {
			for _, tc := range g.Tests {
				// The parser only ever sees string data; the format keyword ignores
				// non-string JSON values. Skip explicit JSON null (decodes to "").
				if string(tc.Data) == "null" {
					continue
				}
				var s string
				if json.Unmarshal(tc.Data, &s) != nil {
					continue
				}

				tc := durationTestCase{tc.Description, s, tc.Valid}
				if !yield(tc) {
					return
				}
			}
		}
	}
}

func lenientISO8601DurationCases() iter.Seq[durationTestCase] {
	return slices.Values([]durationTestCase{
		{Input: "PT0.5S", Desc: "fraction", Valid: true},
		{Input: "P1,5D", Desc: "comma fraction", Valid: true},
		{Input: "-P1D", Desc: "sign", Valid: true},
		{Input: "+P1D", Desc: "sign", Valid: true},
		{Input: " P1D", Desc: "whitespace", Valid: true},
		{Input: "P1D ", Desc: "whitespace", Valid: true},
		{Input: "P1Y2D", Desc: "relaxed anchoring (gap)", Valid: true},
		{Input: "PT1H2S", Desc: "relaxed anchoring (gap)", Valid: true},
		{Input: "P1Y2W", Desc: "week combinable", Valid: true},
		{Input: "P2D1Y", Desc: "invalid under lenient", Valid: false},
		{Input: "PT1M2H", Desc: "invalid under lenient (2)", Valid: false},
	})
}
