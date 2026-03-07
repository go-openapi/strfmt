// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build testintegration

// Package mariadb_test lives in its own sub-package so that tests which modify
// strfmt globals (e.g. MarshalFormat, NormalizeTimeForMarshal) cannot interfere
// with MongoDB integration tests running in a sibling package.
package mariadb_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	_ "github.com/go-sql-driver/mysql"
)

func mariadbDSN() string {
	if dsn := os.Getenv("MARIADB_DSN"); dsn != "" {
		return dsn
	}
	return "strfmt_test:strfmt_test@tcp(localhost:3306)/strfmt_integration_test?parseTime=true&loc=UTC"
}

func setupMariaDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("mysql", mariadbDSN())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, db.PingContext(ctx))

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

// createTable creates a test table and registers cleanup.
// cols is a list of "col_name TYPE" definitions.
func createTable(t *testing.T, db *sql.DB, cols ...string) string {
	t.Helper()

	ctx := context.Background()
	table := "test_" + strings.ReplaceAll(t.Name(), "/", "_")
	ddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id VARCHAR(64) PRIMARY KEY, %s)", table, strings.Join(cols, ", "))
	_, err := db.ExecContext(ctx, ddl)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table)
	})

	return table
}

// MariaDB time tests: these are the problematic types per issue #174.
// The go-sql-driver/mysql has hard-coded handling for time.Time but not for
// type redefinitions like strfmt.DateTime. The driver.Valuer interface returns
// an RFC3339 string with "Z" suffix that MySQL/MariaDB rejects for DATETIME columns.

func TestMariaDB_DateTime_AsString(t *testing.T) {
	// Workaround: store DateTime as VARCHAR, roundtrip via Value()/Scan() with string.
	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value VARCHAR(64)")

	original := strfmt.DateTime(time.Date(2024, 6, 15, 12, 30, 45, 123000000, time.UTC))

	// Insert using Value() — returns RFC3339 string, VARCHAR column accepts it.
	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "dt1", original)
	require.NoError(t, err)

	var got strfmt.DateTime
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "dt1").Scan(&got)
	require.NoError(t, err)

	assert.EqualT(t,
		time.Time(original).UTC().Truncate(time.Millisecond),
		time.Time(got).UTC().Truncate(time.Millisecond),
	)
}

func TestMariaDB_DateTime_NativeDatetime_DefaultFormat(t *testing.T) {
	// The go-sql-driver/mysql has hard-coded handling for time.Time specifically.
	// Since strfmt.DateTime is a type redefinition (type DateTime time.Time), the
	// driver doesn't intercept it and falls through to the driver.Valuer interface.
	// DateTime.Value() returns an RFC3339 string like "2024-06-15T12:30:45.123Z".
	// MySQL/MariaDB rejects this format for DATETIME columns because it doesn't
	// understand the "Z" timezone suffix.
	//
	// See: https://github.com/go-openapi/strfmt/issues/174
	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value DATETIME(3)")

	original := strfmt.DateTime(time.Date(2024, 6, 15, 12, 30, 45, 123000000, time.UTC))

	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "dt1", original)
	require.Error(t, err)
	t.Logf("confirmed: default MarshalFormat rejected by MariaDB: %v", err)
}

func TestMariaDB_DateTime_NativeDatetime_LocalTimeFormat(t *testing.T) {
	// Workaround for MySQL/MariaDB DateTime incompatibility (issue #174):
	// https://github.com/go-openapi/strfmt/issues/174
	//
	// Setting MarshalFormat to ISO8601LocalTime drops the "Z" timezone suffix,
	// producing "2024-06-15T12:30:45" which MySQL/MariaDB accepts for DATETIME
	// columns. NormalizeTimeForMarshal is set to UTC to ensure consistent
	// timezone normalization before the suffix is stripped.
	//
	// Trade-off: ISO8601LocalTime has second-only precision. For sub-second
	// precision, a custom format like "2006-01-02 15:04:05.000" could be used.
	//
	// NOTE: MarshalFormat is a package-level global — changing it affects all
	// strfmt.DateTime instances in the process. This is why the MariaDB tests
	// live in a separate sub-package from the MongoDB tests.
	savedFormat := strfmt.MarshalFormat
	savedNormalize := strfmt.NormalizeTimeForMarshal
	t.Cleanup(func() {
		strfmt.MarshalFormat = savedFormat
		strfmt.NormalizeTimeForMarshal = savedNormalize
	})

	strfmt.MarshalFormat = strfmt.ISO8601LocalTime
	strfmt.NormalizeTimeForMarshal = func(t time.Time) time.Time { return t.UTC() }

	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value DATETIME")

	original := strfmt.DateTime(time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC))

	// With ISO8601LocalTime, Value() returns "2024-06-15T12:30:45" — no "Z", accepted by MySQL.
	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "dt1", original)
	require.NoError(t, err)

	// With parseTime=true, the driver returns time.Time — Scan() handles this.
	var got strfmt.DateTime
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "dt1").Scan(&got)
	require.NoError(t, err)

	assert.EqualT(t,
		time.Time(original).UTC().Truncate(time.Second),
		time.Time(got).UTC().Truncate(time.Second),
	)
}

func TestMariaDB_Date(t *testing.T) {
	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value DATE")

	original := strfmt.Date(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))

	// Date.Value() returns "2024-06-15" which MySQL DATE columns accept.
	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "d1", original)
	require.NoError(t, err)

	var got strfmt.Date
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "d1").Scan(&got)
	require.NoError(t, err)

	assert.EqualT(t, original.String(), got.String())
}

func TestMariaDB_Duration(t *testing.T) {
	// Duration.Value() returns int64 (nanoseconds), so use BIGINT column.
	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value BIGINT")

	original := strfmt.Duration(42 * time.Second)

	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "dur1", original)
	require.NoError(t, err)

	var got strfmt.Duration
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "dur1").Scan(&got)
	require.NoError(t, err)

	assert.EqualT(t, original, got)
}

// stringRoundTrip is a helper for string-based strfmt types stored in VARCHAR columns.
func stringRoundTrip[T interface {
	~string
	fmt.Stringer
}](t *testing.T, db *sql.DB, original T, got *T,
) {
	t.Helper()

	ctx := context.Background()
	table := createTable(t, db, "value TEXT")

	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "v1", original)
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "v1").Scan(got)
	require.NoError(t, err)

	assert.EqualT(t, original.String(), (*got).String())
}

func TestMariaDB_URI(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.URI("https://example.com/path?q=1")
	var got strfmt.URI
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_Email(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.Email("user@example.com")
	var got strfmt.Email
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_Hostname(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.Hostname("example.com")
	var got strfmt.Hostname
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_IPv4(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.IPv4("192.168.1.1")
	var got strfmt.IPv4
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_IPv6(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.IPv6("::1")
	var got strfmt.IPv6
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_CIDR(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.CIDR("192.168.1.0/24")
	var got strfmt.CIDR
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_MAC(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.MAC("01:02:03:04:05:06")
	var got strfmt.MAC
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_UUID(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.UUID("a8098c1a-f86e-11da-bd1a-00112444be1e")
	var got strfmt.UUID
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_UUID3(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.UUID3("bcd02ab7-6beb-3467-84c0-3bdbea962817")
	var got strfmt.UUID3
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_UUID4(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.UUID4("025b0d74-00a2-4885-af46-084e7fbd0701")
	var got strfmt.UUID4
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_UUID5(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.UUID5("886313e1-3b8a-5372-9b90-0c9aee199e5d")
	var got strfmt.UUID5
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_UUID7(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.UUID7("01943ff8-3e9e-7be4-8921-de6a1e04d599")
	var got strfmt.UUID7
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_ISBN(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.ISBN("0321751043")
	var got strfmt.ISBN
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_ISBN10(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.ISBN10("0321751043")
	var got strfmt.ISBN10
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_ISBN13(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.ISBN13("978-0321751041")
	var got strfmt.ISBN13
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_CreditCard(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.CreditCard("4111-1111-1111-1111")
	var got strfmt.CreditCard
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_SSN(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.SSN("111-11-1111")
	var got strfmt.SSN
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_HexColor(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.HexColor("#FFFFFF")
	var got strfmt.HexColor
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_RGBColor(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.RGBColor("rgb(255,255,255)")
	var got strfmt.RGBColor
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_Password(t *testing.T) {
	db := setupMariaDB(t)
	original := strfmt.Password("super secret stuff here")
	var got strfmt.Password
	stringRoundTrip(t, db, original, &got)
}

func TestMariaDB_Base64(t *testing.T) {
	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value TEXT")

	payload := []byte("hello world with special chars: éàü")
	original := strfmt.Base64(payload)

	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "b64_1", original)
	require.NoError(t, err)

	var got strfmt.Base64
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "b64_1").Scan(&got)
	require.NoError(t, err)

	assert.EqualT(t, base64.StdEncoding.EncodeToString(original), base64.StdEncoding.EncodeToString(got))
}

func TestMariaDB_ULID(t *testing.T) {
	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value VARCHAR(64)")

	original, err := strfmt.ParseULID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "ulid1", original)
	require.NoError(t, err)

	var got strfmt.ULID
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "ulid1").Scan(&got)
	require.NoError(t, err)

	assert.EqualT(t, original.String(), got.String())
}

func TestMariaDB_ObjectId(t *testing.T) {
	db := setupMariaDB(t)
	ctx := context.Background()
	table := createTable(t, db, "value VARCHAR(64)")

	original := strfmt.NewObjectId("507f1f77bcf86cd799439011")

	_, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, value) VALUES (?, ?)", table), "oid1", original)
	require.NoError(t, err)

	var got strfmt.ObjectId
	err = db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE id = ?", table), "oid1").Scan(&got)
	require.NoError(t, err)

	assert.EqualT(t, original, got)
}
