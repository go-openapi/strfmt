// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build testintegration

package mongodb_test

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func mongoURI() string {
	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		return uri
	}
	return "mongodb://localhost:27017"
}

func setup(t *testing.T) *mongo.Collection {
	t.Helper()

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI()))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, client.Ping(ctx, nil))

	db := client.Database("strfmt_integration_test")
	coll := db.Collection(t.Name())

	t.Cleanup(func() {
		_ = coll.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	return coll
}

// roundTrip inserts a document containing the value into MongoDB,
// reads it back, and returns the result document.
func roundTrip(t *testing.T, coll *mongo.Collection, doc bson.M) bson.M {
	t.Helper()
	ctx := context.Background()

	_, err := coll.InsertOne(ctx, doc)
	require.NoError(t, err)

	var result bson.M
	err = coll.FindOne(ctx, bson.M{"_id": doc["_id"]}).Decode(&result)
	require.NoError(t, err)

	return result
}

func TestDate(t *testing.T) {
	coll := setup(t)
	original := strfmt.Date(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))

	doc := bson.M{"_id": "date_test", "value": original}
	result := roundTrip(t, coll, doc)

	raw, ok := result["value"].(bson.D)
	require.TrueT(t, ok, "expected bson.D for value, got %T", result["value"])

	rawBytes, err := bson.Marshal(raw)
	require.NoError(t, err)

	var got strfmt.Date
	require.NoError(t, bson.Unmarshal(rawBytes, &got))

	assert.EqualT(t, original.String(), got.String())
}

func TestDateTime(t *testing.T) {
	coll := setup(t)
	original := strfmt.DateTime(time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC))

	doc := bson.M{"_id": "datetime_test", "value": original}
	result := roundTrip(t, coll, doc)

	// DateTime uses MarshalBSONValue, so MongoDB stores it as a native datetime.
	dt, ok := result["value"].(bson.DateTime)
	require.TrueT(t, ok, "expected bson.DateTime, got %T", result["value"])

	got := strfmt.DateTime(dt.Time())

	assert.EqualT(t, time.Time(original).UTC().UnixMilli(), time.Time(got).UTC().UnixMilli())
}

func TestDuration(t *testing.T) {
	coll := setup(t)
	original := strfmt.Duration(42 * time.Second)

	doc := bson.M{"_id": "duration_test", "value": original}
	result := roundTrip(t, coll, doc)

	raw, ok := result["value"].(bson.D)
	require.TrueT(t, ok, "expected bson.D for value, got %T", result["value"])

	rawBytes, err := bson.Marshal(raw)
	require.NoError(t, err)

	var got strfmt.Duration
	require.NoError(t, bson.Unmarshal(rawBytes, &got))

	assert.EqualT(t, original, got)
}

func TestBase64(t *testing.T) {
	coll := setup(t)
	payload := []byte("hello world with special chars: éàü")
	original := strfmt.Base64(payload)

	doc := bson.M{"_id": "base64_test", "value": original}
	result := roundTrip(t, coll, doc)

	raw, ok := result["value"].(bson.D)
	require.TrueT(t, ok, "expected bson.D for value, got %T", result["value"])

	rawBytes, err := bson.Marshal(raw)
	require.NoError(t, err)

	var got strfmt.Base64
	require.NoError(t, bson.Unmarshal(rawBytes, &got))

	assert.EqualT(t, base64.StdEncoding.EncodeToString(original), base64.StdEncoding.EncodeToString(got))
}

func TestULID(t *testing.T) {
	coll := setup(t)
	original, err := strfmt.ParseULID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	require.NoError(t, err)

	doc := bson.M{"_id": "ulid_test", "value": original}
	result := roundTrip(t, coll, doc)

	raw, ok := result["value"].(bson.D)
	require.TrueT(t, ok, "expected bson.D for value, got %T", result["value"])

	rawBytes, err := bson.Marshal(raw)
	require.NoError(t, err)

	var got strfmt.ULID
	require.NoError(t, bson.Unmarshal(rawBytes, &got))

	assert.EqualT(t, original, got)
}

func TestObjectId(t *testing.T) {
	coll := setup(t)
	original := strfmt.NewObjectId("507f1f77bcf86cd799439011")

	doc := bson.M{"_id": "objectid_test", "value": original}
	result := roundTrip(t, coll, doc)

	// ObjectId uses MarshalBSONValue, so MongoDB stores it as a native ObjectID.
	oid, ok := result["value"].(bson.ObjectID)
	require.TrueT(t, ok, "expected bson.ObjectID, got %T", result["value"])

	got := strfmt.ObjectId(oid)

	assert.EqualT(t, original, got)
}

// stringFormatRoundTrip is a helper for types that serialize as embedded BSON documents
// with a "data" string field (most strfmt string-based types).
func stringFormatRoundTrip(t *testing.T, coll *mongo.Collection, id string, input bson.Marshaler, output bson.Unmarshaler, _ string) {
	t.Helper()

	doc := bson.M{"_id": id, "value": input}
	result := roundTrip(t, coll, doc)

	raw, ok := result["value"].(bson.D)
	require.TrueT(t, ok, "expected bson.D for value, got %T", result["value"])

	rawBytes, err := bson.Marshal(raw)
	require.NoError(t, err)

	require.NoError(t, bson.Unmarshal(rawBytes, output))
}

func TestURI(t *testing.T) {
	coll := setup(t)
	original := strfmt.URI("https://example.com/path?q=1")
	var got strfmt.URI
	stringFormatRoundTrip(t, coll, "uri_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestEmail(t *testing.T) {
	coll := setup(t)
	original := strfmt.Email("user@example.com")
	var got strfmt.Email
	stringFormatRoundTrip(t, coll, "email_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestHostname(t *testing.T) {
	coll := setup(t)
	original := strfmt.Hostname("example.com")
	var got strfmt.Hostname
	stringFormatRoundTrip(t, coll, "hostname_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestIPv4(t *testing.T) {
	coll := setup(t)
	original := strfmt.IPv4("192.168.1.1")
	var got strfmt.IPv4
	stringFormatRoundTrip(t, coll, "ipv4_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestIPv6(t *testing.T) {
	coll := setup(t)
	original := strfmt.IPv6("::1")
	var got strfmt.IPv6
	stringFormatRoundTrip(t, coll, "ipv6_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestCIDR(t *testing.T) {
	coll := setup(t)
	original := strfmt.CIDR("192.168.1.0/24")
	var got strfmt.CIDR
	stringFormatRoundTrip(t, coll, "cidr_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestMAC(t *testing.T) {
	coll := setup(t)
	original := strfmt.MAC("01:02:03:04:05:06")
	var got strfmt.MAC
	stringFormatRoundTrip(t, coll, "mac_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestUUID(t *testing.T) {
	coll := setup(t)
	original := strfmt.UUID("a8098c1a-f86e-11da-bd1a-00112444be1e")
	var got strfmt.UUID
	stringFormatRoundTrip(t, coll, "uuid_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestUUID3(t *testing.T) {
	coll := setup(t)
	original := strfmt.UUID3("bcd02ab7-6beb-3467-84c0-3bdbea962817")
	var got strfmt.UUID3
	stringFormatRoundTrip(t, coll, "uuid3_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestUUID4(t *testing.T) {
	coll := setup(t)
	original := strfmt.UUID4("025b0d74-00a2-4885-af46-084e7fbd0701")
	var got strfmt.UUID4
	stringFormatRoundTrip(t, coll, "uuid4_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestUUID5(t *testing.T) {
	coll := setup(t)
	original := strfmt.UUID5("886313e1-3b8a-5372-9b90-0c9aee199e5d")
	var got strfmt.UUID5
	stringFormatRoundTrip(t, coll, "uuid5_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestUUID7(t *testing.T) {
	coll := setup(t)
	original := strfmt.UUID7("01943ff8-3e9e-7be4-8921-de6a1e04d599")
	var got strfmt.UUID7
	stringFormatRoundTrip(t, coll, "uuid7_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestISBN(t *testing.T) {
	coll := setup(t)
	original := strfmt.ISBN("0321751043")
	var got strfmt.ISBN
	stringFormatRoundTrip(t, coll, "isbn_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestISBN10(t *testing.T) {
	coll := setup(t)
	original := strfmt.ISBN10("0321751043")
	var got strfmt.ISBN10
	stringFormatRoundTrip(t, coll, "isbn10_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestISBN13(t *testing.T) {
	coll := setup(t)
	original := strfmt.ISBN13("978-0321751041")
	var got strfmt.ISBN13
	stringFormatRoundTrip(t, coll, "isbn13_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestCreditCard(t *testing.T) {
	coll := setup(t)
	original := strfmt.CreditCard("4111-1111-1111-1111")
	var got strfmt.CreditCard
	stringFormatRoundTrip(t, coll, "creditcard_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestSSN(t *testing.T) {
	coll := setup(t)
	original := strfmt.SSN("111-11-1111")
	var got strfmt.SSN
	stringFormatRoundTrip(t, coll, "ssn_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestHexColor(t *testing.T) {
	coll := setup(t)
	original := strfmt.HexColor("#FFFFFF")
	var got strfmt.HexColor
	stringFormatRoundTrip(t, coll, "hexcolor_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestRGBColor(t *testing.T) {
	coll := setup(t)
	original := strfmt.RGBColor("rgb(255,255,255)")
	var got strfmt.RGBColor
	stringFormatRoundTrip(t, coll, "rgbcolor_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}

func TestPassword(t *testing.T) {
	coll := setup(t)
	original := strfmt.Password("super secret stuff here")
	var got strfmt.Password
	stringFormatRoundTrip(t, coll, "password_test", original, &got, string(original))
	assert.EqualT(t, original, got)
}
