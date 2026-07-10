// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	hoursPerDay  = 24
	daysPerWeek  = 7
	daysPerMonth = 30
	daysPerYear  = 365

	base10   = 10
	maxScale = 1000000000000000000 // 1e18

	maxUint64Div10 = math.MaxUint64 / 10
	maxUint64Mod10 = '0' + byte(math.MaxUint64%10)
)

var (
	errEmptyDuration = fmt.Errorf("%w: empty duration", ErrFormat)
	errInvalidPStart = fmt.Errorf("%w: invalid ISO 8601 duration: must start with P", ErrFormat)
	errEmptyAfterP   = fmt.Errorf("%w: invalid ISO 8601 duration: empty after P", ErrFormat)
	errOverflow      = fmt.Errorf("%w: numerical overflow", ErrFormat)
	errFraction      = fmt.Errorf("%w: decimal fraction is only allowed on the least significant unit", ErrFormat)
	errEmptyTime     = fmt.Errorf("%w: empty time part after T", ErrFormat)
	errInvalidDec    = fmt.Errorf("%w: invalid decimal number", ErrFormat)
)

type durationParser struct {
	s           string
	total       uint64
	hasFraction bool
	neg         bool
}

// ParseISO8601Duration parses an ISO 8601 duration string.
func ParseISO8601Duration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errEmptyDuration
	}

	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		s = s[1:]
	}

	if len(s) == 0 || s[0] != 'P' {
		return 0, errInvalidPStart
	}
	s = s[1:] // Consume 'P'
	if s == "" {
		return 0, errEmptyAfterP
	}

	p := durationParser{
		s:   s,
		neg: neg,
	}

	if err := p.parseDatePart(); err != nil {
		return 0, err
	}

	if err := p.parseTimePart(); err != nil {
		return 0, err
	}

	// If there is anything left in the string, it indicates invalid junk or out-of-order components
	if len(p.s) > 0 {
		return 0, fmt.Errorf("%w: unrecognized trailing character %q", ErrFormat, p.s[0])
	}

	// Final boundary check for the positive duration limits
	const maxDurationVal = uint64(1 << 63)
	if p.total > maxDurationVal-1 {
		if neg && p.total == maxDurationVal {
			return time.Duration(math.MinInt64), nil
		}
		return 0, errOverflow
	}

	dur := time.Duration(p.total)
	if neg {
		dur = -dur
	}
	return dur, nil
}

func (p *durationParser) parseDatePart() error {
	// 1. Years
	if err := p.parseField(daysPerYear*hoursPerDay*time.Hour, 'Y'); err != nil {
		return err
	}
	// 2. Months (before T)
	if err := p.parseField(daysPerMonth*hoursPerDay*time.Hour, 'M'); err != nil {
		return err
	}
	// 3. Weeks
	if err := p.parseField(daysPerWeek*hoursPerDay*time.Hour, 'W'); err != nil {
		return err
	}
	// 4. Days
	if err := p.parseField(hoursPerDay*time.Hour, 'D'); err != nil {
		return err
	}
	return nil
}

func (p *durationParser) parseTimePart() error {
	if len(p.s) == 0 {
		return nil
	}
	if p.s[0] != 'T' {
		return fmt.Errorf("%w: unrecognized trailing character %q", ErrFormat, p.s[0])
	}
	p.s = p.s[1:] // Consume 'T'
	if len(p.s) == 0 {
		return errEmptyTime
	}

	// 1. Hours
	if err := p.parseField(time.Hour, 'H'); err != nil {
		return err
	}
	// 2. Minutes (after T)
	if err := p.parseField(time.Minute, 'M'); err != nil {
		return err
	}
	// 3. Seconds
	if err := p.parseField(time.Second, 'S'); err != nil {
		return err
	}
	return nil
}

func (p *durationParser) parseField(unit time.Duration, suffix byte) error {
	if err := p.parseAndAccumulate(unit, suffix); err != nil {
		return err
	}
	if p.hasFraction && len(p.s) > 0 {
		return errFraction
	}
	return nil
}

func (p *durationParser) parseAndAccumulate(unit time.Duration, suffix byte) error {
	val, nextS, hasFraction, err := parseOptionalField(p.s, unit, suffix)
	if err != nil {
		return err
	}
	p.s = nextS
	p.hasFraction = hasFraction

	if val == 0 {
		return nil
	}

	const maxVal = uint64(1 << 63)
	if val > maxVal || p.total > maxVal-val {
		// Special case: time.MinDuration (-1 << 63) is allowed if negative
		if p.neg && p.total+val == maxVal {
			p.total += val
			return nil
		}
		return errOverflow
	}

	p.total += val
	return nil
}

// parseOptionalField parses the number and trailing suffix unit in a single pass.
// It returns whether a decimal fraction was parsed (hasFraction).
func parseOptionalField(s string, unit time.Duration, suffix byte) (uint64, string, bool, error) {
	i := 0
	var v uint64
	var f uint64
	var scale uint64 = 1
	hasDot := false

loop:
	for i < len(s) {
		c := s[i]
		switch {
		case '0' <= c && c <= '9':
			if !hasDot {
				// Integer part overflow check - strict 10^18 boundary check
				if v > maxUint64Div10 || (v == maxUint64Div10 && c > maxUint64Mod10) {
					return 0, "", false, errOverflow
				}
				v = v*base10 + uint64(c-'0')
			} else if scale < maxScale {
				f = f*base10 + uint64(c-'0')
				scale *= base10
			}
			i++
		case c == '.' || c == ',': // Support both dot and comma as decimal separator
			if hasDot {
				return 0, "", false, errInvalidDec
			}
			hasDot = true
			i++
		default:
			break loop
		}
	}

	if i == 0 {
		return 0, s, false, nil // No digits found: skip field
	}
	if i >= len(s) || s[i] != suffix {
		return 0, s, false, nil // Suffix doesn't match: skip field
	}

	//nolint:gosec // unit is a positive time.Duration, conversion to uint64 is safe
	u := uint64(unit)

	// Multiplication overflow check to prevent silent wrap-around
	if v > 0 && u > math.MaxUint64/v {
		return 0, "", false, errOverflow
	}

	// Calculate total value in nanoseconds
	val := v * u
	if f > 0 {
		// Use float64 multiplication for fractional calculation to prevent scale overflow
		val += uint64(float64(f) * (float64(u) / float64(scale)))
	}
	return val, s[i+1:], hasDot, nil // Consume digits and suffix
}
