// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package conv

import "github.com/go-openapi/strfmt"

// DurationISO8601 returns a pointer to of the [strfmt.DurationISO8601] value passed in.
func DurationISO8601(v strfmt.DurationISO8601) *strfmt.DurationISO8601 {
	return &v
}

// DurationISO8601Value returns the value of the [strfmt.DurationISO8601] pointer passed in or
// the default value if the pointer is nil.
func DurationISO8601Value(v *strfmt.DurationISO8601) strfmt.DurationISO8601 {
	if v == nil {
		return strfmt.DurationISO8601(0)
	}

	return *v
}
