// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"reflect"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type testableBSONFormat interface {
	testableFormat

	bson.Marshaler
	bson.Unmarshaler
}

func TestBSONDate(t *testing.T) {
	dateOriginal := Date(time.Date(2014, 10, 10, 0, 0, 0, 0, time.UTC))

	bsonData, err := bson.Marshal(&dateOriginal)
	require.NoError(t, err)

	var dateCopy Date
	err = bson.Unmarshal(bsonData, &dateCopy)
	require.NoError(t, err)
	assert.Equal(t, dateOriginal, dateCopy)
}

func TestBSONBase64(t *testing.T) {
	const b64 string = "This is a byte array with unprintable chars, but it also isn"
	b := []byte(b64)
	subj := Base64(b)

	bsonData, err := bson.Marshal(subj)
	require.NoError(t, err)

	var b64Copy Base64
	err = bson.Unmarshal(bsonData, &b64Copy)
	require.NoError(t, err)
	assert.Equal(t, subj, b64Copy)
}

func TestBSONDuration(t *testing.T) {
	dur := Duration(42)
	bsonData, err := bson.Marshal(&dur)
	require.NoError(t, err)

	var durCopy Duration
	err = bson.Unmarshal(bsonData, &durCopy)
	require.NoError(t, err)
	assert.Equal(t, dur, durCopy)
}

func TestBSONDateTime(t *testing.T) {
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)
		dt := DateTime(example.time)

		bsonData, err := bson.Marshal(&dt)
		require.NoError(t, err)

		var dtCopy DateTime
		err = bson.Unmarshal(bsonData, &dtCopy)
		require.NoError(t, err)
		// BSON DateTime type loses timezone information, so compare UTC()
		assert.Equal(t, time.Time(dt).UTC(), time.Time(dtCopy).UTC())

		// Check value marshaling explicitly
		m := bson.M{"data": dt}
		bsonData, err = bson.Marshal(&m)
		require.NoError(t, err)

		var mCopy bson.M
		err = bson.Unmarshal(bsonData, &mCopy)
		require.NoError(t, err)

		data, ok := m["data"].(DateTime)
		assert.True(t, ok)
		assert.Equal(t, time.Time(dt).UTC(), time.Time(data).UTC())
	}
}

func TestBSONULID(t *testing.T) {
	t.Parallel()
	t.Run("positive", func(t *testing.T) {
		t.Parallel()
		ulid, _ := ParseULID(testUlid)

		bsonData, err := bson.Marshal(&ulid)
		require.NoError(t, err)

		var ulidUnmarshaled ULID
		err = bson.Unmarshal(bsonData, &ulidUnmarshaled)
		require.NoError(t, err)
		assert.Equal(t, ulid, ulidUnmarshaled)

		// Check value marshaling explicitly
		m := bson.M{"data": ulid}
		bsonData, err = bson.Marshal(&m)
		require.NoError(t, err)

		var mUnmarshaled bson.M
		err = bson.Unmarshal(bsonData, &mUnmarshaled)
		require.NoError(t, err)

		data, ok := m["data"].(ULID)
		assert.True(t, ok)
		assert.Equal(t, ulid, data)
	})
	t.Run("negative", func(t *testing.T) {
		t.Parallel()
		uuid := UUID("00000000-0000-0000-0000-000000000000")
		bsonData, err := bson.Marshal(&uuid)
		require.NoError(t, err)

		var ulidUnmarshaled ULID
		err = bson.Unmarshal(bsonData, &ulidUnmarshaled)
		require.Error(t, err)
	})
}

func TestFormatBSON(t *testing.T) {
	t.Run("with URI", func(t *testing.T) {
		t.Run("should bson.Marshal and bson.Unmarshal", func(t *testing.T) {
			uri := URI("http://somewhere.com")
			str := "http://somewhereelse.com"
			testBSONStringFormat(t, &uri, "uri", str, []string{}, []string{"somewhere.com"})
		})
	})

	t.Run("with Email", func(t *testing.T) {
		email := Email("somebody@somewhere.com")
		str := string("somebodyelse@somewhere.com")

		testBSONStringFormat(t, &email, "email", str, validEmails(), []string{"somebody@somewhere@com"})
	})

	t.Run("with Hostname", func(t *testing.T) {
		hostname := Hostname("somewhere.com")
		str := string("somewhere.com")

		testBSONStringFormat(t, &hostname, "hostname", str, []string{}, invalidHostnames())
		testBSONStringFormat(t, &hostname, "hostname", str, validHostnames(), []string{})
	})

	t.Run("with IPv4", func(t *testing.T) {
		ipv4 := IPv4("192.168.254.1")
		str := string("192.168.254.2")
		testBSONStringFormat(t, &ipv4, "ipv4", str, []string{}, []string{"198.168.254.2.2"})
	})

	t.Run("with IPv6", func(t *testing.T) {
		ipv6 := IPv6("::1")
		str := string("::2")
		testBSONStringFormat(t, &ipv6, "ipv6", str, []string{}, []string{"127.0.0.1"})
	})

	t.Run("with CIDR", func(t *testing.T) {
		cidr := CIDR("192.168.254.1/24")
		str := string("192.168.254.2/24")
		testBSONStringFormat(t, &cidr, "cidr", str, []string{"192.0.2.1/24", "2001:db8:a0b:12f0::1/32"}, []string{"198.168.254.2", "2001:db8:a0b:12f0::1"})
	})

	t.Run("with MAC", func(t *testing.T) {
		mac := MAC("01:02:03:04:05:06")
		str := string("06:05:04:03:02:01")
		testBSONStringFormat(t, &mac, "mac", str, []string{}, []string{"01:02:03:04:05"})
	})

	t.Run("with UUID3", func(t *testing.T) {
		first3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhere.com"))
		uuid3 := UUID3(first3.String())
		str := first3.String()
		testBSONStringFormat(t, &uuid3, "uuid3", str,
			validUUID3s(),
			invalidUUID3s(),
		)
	})

	t.Run("with UUID4", func(t *testing.T) {
		first4 := uuid.Must(uuid.NewRandom())
		other4 := uuid.Must(uuid.NewRandom())
		uuid4 := UUID4(first4.String())
		str := other4.String()
		testBSONStringFormat(t, &uuid4, "uuid4", str,
			validUUID4s(),
			invalidUUID4s(),
		)
	})

	t.Run("with UUID5", func(t *testing.T) {
		first5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhere.com"))
		other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))
		uuid5 := UUID5(first5.String())
		str := other5.String()
		testBSONStringFormat(t, &uuid5, "uuid5", str,
			validUUID5s(),
			invalidUUID5s(),
		)
	})

	t.Run("with UUID7", func(t *testing.T) {
		first7 := uuid.Must(uuid.NewV7())
		str := first7.String()
		uuid7 := UUID7(str)
		testBSONStringFormat(t, &uuid7, "uuid7", str,
			validUUID7s(),
			invalidUUID7s(),
		)
	})

	t.Run("with UUID", func(t *testing.T) {
		first5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhere.com"))
		other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))
		uuid := UUID(first5.String())
		str := other5.String()
		testBSONStringFormat(t, &uuid, "uuid", str,
			validUUIDs(),
			invalidUUIDs(),
		)
	})

	t.Run("with ISBN", func(t *testing.T) {
		isbn := ISBN("0321751043")
		str := string("0321751043")
		testBSONStringFormat(t, &isbn, "isbn", str, []string{}, []string{"836217463"}) // bad checksum
	})

	t.Run("with ISBN10", func(t *testing.T) {
		isbn10 := ISBN10("0321751043")
		str := string("0321751043")
		testBSONStringFormat(t, &isbn10, "isbn10", str, []string{}, []string{"836217463"}) // bad checksum
	})

	t.Run("with ISBN13", func(t *testing.T) {
		isbn13 := ISBN13("978-0321751041")
		str := string("978-0321751041")
		testBSONStringFormat(t, &isbn13, "isbn13", str, []string{}, []string{"978-0321751042"}) // bad checksum
	})

	t.Run("with HexColor", func(t *testing.T) {
		hexColor := HexColor("#FFFFFF")
		str := string("#000000")
		testBSONStringFormat(t, &hexColor, "hexcolor", str, []string{}, []string{"#fffffffz"})
	})

	t.Run("with RGBColor", func(t *testing.T) {
		rgbColor := RGBColor("rgb(255,255,255)")
		str := string("rgb(0,0,0)")
		testBSONStringFormat(t, &rgbColor, "rgbcolor", str, []string{}, []string{"rgb(300,0,0)"})
	})

	t.Run("with SSN", func(t *testing.T) {
		ssn := SSN("111-11-1111")
		str := string("999 99 9999")
		testBSONStringFormat(t, &ssn, "ssn", str, []string{}, []string{"999 99 999"})
	})

	t.Run("with CreditCard", func(t *testing.T) {
		creditCard := CreditCard("4111-1111-1111-1111")
		str := string("4012-8888-8888-1881")
		testBSONStringFormat(t, &creditCard, "creditcard", str, []string{}, []string{"9999-9999-9999-999"})
	})

	t.Run("with Password", func(t *testing.T) {
		password := Password("super secret stuff here")
		testBSONStringFormat(t, &password, "password", "super secret!!!", []string{"even more secret"}, []string{})
	})
}

func testBSONStringFormat(t *testing.T, what testableBSONFormat, format, with string, _, _ []string) {
	t.Helper()
	b := []byte(with)
	err := what.UnmarshalText(b)
	require.NoError(t, err)

	// bson encoding interface
	bsonData, err := bson.Marshal(what)
	require.NoError(t, err)

	resetValue(t, format, what)

	err = bson.Unmarshal(bsonData, what)
	require.NoError(t, err)
	val := reflect.Indirect(reflect.ValueOf(what))
	strVal := val.String()
	assert.Equal(t, with, strVal, "[%s]bson.Unmarshal: expected %v and %v to be equal (reset value) ", format, what, with)
}
