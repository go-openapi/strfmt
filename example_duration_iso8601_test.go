// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/strfmt/conv"
)

// The strict parser accepts the RFC 3339 Appendix A grammar (the JSON Schema "duration" format)
// and returns an ordinary [time.Duration]. Calendar units are collapsed to fixed lengths
// (year = 365 days, month = 30 days).
func ExampleParseISO8601Duration() {
	d, err := strfmt.ParseISO8601Duration("P1DT12H")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(d)

	// The strict grammar rejects fractions, signs and whitespace; every rejection
	// wraps strfmt.ErrFormat.
	_, err = strfmt.ParseISO8601Duration("PT0.5S")
	fmt.Println(errors.Is(err, strfmt.ErrFormat))

	// Output:
	// 36h0m0s
	// true
}

// Options relax individual rules on the explicit ParseISO8601Duration path only — never the
// registry or struct-field decode path, which stays strict and unambiguous.
func ExampleParseISO8601Duration_options() {
	// A decimal fraction on the least significant component is opt-in.
	frac, _ := strfmt.ParseISO8601Duration("PT0.5S", strfmt.WithISOFractions())
	fmt.Println(frac)

	// A leading sign is opt-in.
	neg, _ := strfmt.ParseISO8601Duration("-PT1H", strfmt.WithISOSign())
	fmt.Println(neg)

	// WithISOLenient turns on every relaxation at once (fraction, sign, whitespace,
	// non-contiguous components, combinable "W"). Ordering is still enforced.
	lenient, _ := strfmt.ParseISO8601Duration("  P1Y2D  ", strfmt.WithISOLenient())
	fmt.Println(lenient)

	// Output:
	// 500ms
	// -1h0m0s
	// 8808h0m0s
}

// DurationISO8601 is the strict alias; it round-trips through JSON as a canonical ISO 8601 string.
func ExampleDurationISO8601_json() {
	type Payload struct {
		Timeout strfmt.DurationISO8601 `json:"timeout"`
	}

	var p Payload
	if err := json.Unmarshal([]byte(`{"timeout":"PT1H30M"}`), &p); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(time.Duration(p.Timeout))

	out, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(out))

	// Output:
	// 1h30m0s
	// {"timeout":"PT1H30M"}
}

// String is a lossless display form: it always renders the true value, even one a strict policy
// could not emit (a sign, or sub-second precision).
func ExampleDurationISO8601_String() {
	fmt.Println(strfmt.DurationISO8601(90 * time.Minute).String())
	fmt.Println(strfmt.DurationISO8601(time.Second + 500*time.Millisecond).String())
	fmt.Println(strfmt.DurationISO8601(-(time.Hour + 30*time.Minute)).String())

	// Output:
	// PT1H30M
	// PT1.5S
	// -PT1H30M
}

// Selecting the DurationLenient policy as the type parameter applies the relaxed grammar on the
// struct-decode path too — useful when consuming feeds that emit fractions or signs.
func ExampleISODuration_lenient() {
	type Config struct {
		Interval strfmt.ISODuration[strfmt.DurationLenient] `json:"interval"`
	}

	var c Config
	if err := json.Unmarshal([]byte(`{"interval":"PT0.25S"}`), &c); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(time.Duration(c.Interval))

	// Output:
	// 250ms
}

// The format is registered in the Default registry under the "duration-iso8601" handle, distinct
// from the human-readable "duration" format.
func ExampleDurationISO8601_registry() {
	// A plain week duration is valid; strict "W" is exclusive, so combining it is rejected.
	fmt.Println(strfmt.Default.Validates("duration-iso8601", "P2W"))
	fmt.Println(strfmt.Default.Validates("duration-iso8601", "P2W3D")) // invalid
	fmt.Println()
	fmt.Println(strfmt.Default.Validates("duration", "2 weeks")) // human duration is the default for duration
	fmt.Println(strfmt.Default.Validates("duration", "P2W"))     // iso8601 is NOT the default for duration
	fmt.Println()
	fmt.Println(strfmt.JSONSchema2020Registry.Validates("duration", "P2W"))           // iso8601 is the duration for [JSONSchema2020Registry]
	fmt.Println(strfmt.JSONSchema2020Registry.Validates("duration", "2 weeks"))       // human duration is NOT the default duration for JSONSchema 2020
	fmt.Println(strfmt.JSONSchema2020Registry.Validates("duration-human", "2 weeks")) // human duration remain supported as an explicit format

	// Output:
	// true
	// false
	//
	// true
	// false
	//
	// true
	// false
	// true
}

// The conv sub-package provides the usual value/pointer helpers for the strict alias.
func ExampleDurationISO8601Value() {
	p := conv.DurationISO8601(strfmt.DurationISO8601(time.Hour))
	fmt.Println(time.Duration(conv.DurationISO8601Value(p)))

	// A nil pointer decodes to the zero duration.
	fmt.Println(time.Duration(conv.DurationISO8601Value(nil)))

	// Output:
	// 1h0m0s
	// 0s
}
