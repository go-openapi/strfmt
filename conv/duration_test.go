// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package conv

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"

	"github.com/go-openapi/strfmt"
)

func TestDurationValue(t *testing.T) {
	assert.Equal(t, strfmt.Duration(0), DurationValue(nil))
	duration := strfmt.Duration(42)
	assert.Equal(t, duration, DurationValue(&duration))
}
