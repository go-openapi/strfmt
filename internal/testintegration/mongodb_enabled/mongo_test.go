// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build testintegration

// Package mongodb_enabled_test runs MongoDB integration tests with the real
// MongoDB driver codec enabled via blank import.
package mongodb_enabled_test

import (
	"testing"

	_ "github.com/go-openapi/strfmt/enable/mongodb"
	"github.com/go-openapi/strfmt/internal/testintegration/mongotest"
)

func TestMongoDBDriverCodec(t *testing.T) {
	mongotest.RunAllTests(t)
}
