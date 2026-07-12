// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package conv

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"

	"github.com/go-openapi/strfmt"
)

func TestDurationISO8601Value(t *testing.T) {
	assert.EqualT(t, strfmt.DurationISO8601(0), DurationISO8601Value(nil))
	duration := strfmt.DurationISO8601(42)
	assert.EqualT(t, duration, DurationISO8601Value(&duration))
	assert.EqualT(t, duration, *DurationISO8601(duration))
}
