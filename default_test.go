// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
)

func TestFormatURI(t *testing.T) {
	uri := URI("http://somewhere.com")
	str := "http://somewhereelse.com"
	testStringFormat(t, &uri, "uri", str, []string{}, []string{"somewhere.com"})
}

func validEmails() []string {
	return []string{
		"blah@gmail.com",
		"test@d.verylongtoplevel",
		"email+tag@gmail.com",
		`" "@example.com`,
		`"Abc\@def"@example.com`,
		`"Fred Bloggs"@example.com`,
		`"Joe\\Blow"@example.com`,
		`"Abc@def"@example.com`,
		"customer/department=shipping@example.com",
		"$A12345@example.com",
		"!def!xyz%abc@example.com",
		"_somename@example.com",
		"!#$%&'*+-/=?^_`{}|~@example.com",
		"Miles.O'Brian@example.com",
		"postmaster@☁→❄→☃→☀→☺→☂→☹→✝.ws",
		"root@localhost",
		"john@com",
		"api@piston.ninja",
	}
}

func TestFormatEmail(t *testing.T) {
	email := Email("somebody@somewhere.com")
	str := string("somebodyelse@somewhere.com")

	testStringFormat(t, &email, "email", str, validEmails(), []string{"somebody@somewhere@com"})
}

func invalidHostnames() []string {
	veryLongStr := strings.Repeat("a", 256)
	longStr := strings.Repeat("a", 64)
	longAddrSegment := strings.Join([]string{"x", "y", longStr}, ".")

	return []string{
		"somewhere.com!",
		"user@email.domain",
		veryLongStr,
		longAddrSegment,
		// dashes
		"www.example-.org",
		"www.--example.org",
		"-www.example.org",
		"www-.example.org",
		"www.d-.org",
		"www.-d.org",
		"www-",
		"-www",
		"a.b.c.dot-",
		// other characters (not in symbols)
		"www.ex ample.org",
		"_www.example.org",
		"www.ex;ample.org",
		"www.example_underscored.org",
		// top-level domains too short
		"www.詹姆斯.x",
		"a.b.c.d",
		"www.詹姆斯.XN--1B4C3D", // invalid puny code
		"@www",
		"a.b.c.é;ö",
		// these code points are invalid
		"ex=ample.com",
		"ex$ample",
		"example^example",
		"<foo>",
		"ex_ample",
		"ex*ample",
		"ex\\ample",
		"www.\u0025",
		"www.\u007f",
		"www.\u0000",
		"www..com",
		"..example.com",
		"5512",                      // only digits is invalid
		"[fe80::b059:65f4:e877:c40", // invalid ip v6
		"fe80::b059:65f4:e877:c40]",
		"[192.168.250.1]",               // is an ip v4 not an ip v6
		"[",                             // invalid start of ipv6
		"[]",                            // empty ip v6
		"[1:2:3:4:5:6:7:8:9]",           // invalid ip v6
		"[:1]]",                         // invalid ip v6
		"[1::1::1]]",                    // invalid ip v6
		"[fe80::1%en0]",                 // ip v6 with zone
		"[fe80::b059:65f4:e877:c40%20]", // ip v6 with zone
		"[2001:0db8:85a3:0000:0000:8a2e:0370:7334].", // invalid in this context
		"", // empty host
		".",
		"..",
		"192.168.219.168.254",    // invalid ip v4
		"256.256.256.256",        // looks like an IP v4 but is not
		"192.168..168",           // invalid ip v4
		"192.168.0xg.168",        // invalid ip v4
		"0..0x300",               // out of range IP
		"1.2.3.09",               // leading 0, not an octal value
		"09.2.3.4",               // leading 0, not an octal value
		"0x100.2.3.4",            // out of range IP v4
		"192.0xffA80001",         // out of range IP v4
		"0x0a.2.0x0000000000f.3", // number part is too long
		"foo.2.3.4",              // expected an IP v4
		"foo.09",                 // expected an IP v4
		"foo.0x04",               // expected an IP v4
		"💩.123",                  //  expected an IP v4
		"0b1010.2.3.4",           // unsupported binary digits
		"0o07.2.3.4",             // unsupported alternated octal notation
		"localhost:81",
	}
}

func validHostnames() []string {
	return []string{
		"somewhere.com",
		"Somewhere.Com",
		"888.com",
		"a.com",
		"a.b.com",
		"a.b.c.com",
		"a.b.c.d.com",
		"a.b.c.d.e.com",
		"1.com",
		"1.2.com",
		"1.2.3.com",
		"1.2.3.4.com",
		"99.domain.com",
		"99.99.domain.com",
		"1wwworg.example.com", // valid, per RFC1123
		"1000wwworg.example.com",
		"xn--bcher-kva.example.com", // puny encoded
		"xn-80ak6aa92e.co",
		"xn-80ak6aa92e.com",
		"xn--ls8h.la",
		"x.",       // valid trailing dot
		"foo.bar.", // valid trailing dot
		// extended symbol alphabet
		"☁→❄→☃→☀→☺→☂→☹→✝.ws",
		"💩.tv",
		"www.example.onion",
		"www.example.ôlà",
		"ôlà.ôlà",
		"ôlà.ôlà.ôlà",
		"localhost",
		"example",
		"x",
		"x-y",
		"a.b.c.dot",
		"www.example.org",
		"a.b.c.d.e.f.g.dot",
		"www.example-hyphenated.org",
		"foo.x04", // valid (last part not a number)
		"foo.0xz", // valid (last part not a number)
		// localized hostnames
		"www.詹姆斯.org",
		"example.إختبار",
		"www.élégigôö.org",
		"www.詹姆斯.london", // long top-level domain
		// localized top-level domains (valid unicode top-level domains)
		"www.च.चऒ",
		"www.कॉम",
		"www.詹姆斯.xn--11b4c3d", // valid puny code
		"1.1.1.1",             // is a valid IP v4 address
		"1.1.1.1.",            // is a valid IP v4 address, with trailing dot
		"1.1.1.06",            // valid IP, with last part octal
		"1.1.1.0xf",           // valid IP, with last part hex
		"1.1.1.0xz",           // valid hostname, not IP
		"1.0.1.1",             // is a valid IP v4 address
		"1.0x.1.1",            // is a valid IP v4 address
		"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]", // is a valid IP v6 address
		"192.168.219.a1",     // looks like an invalid ip v4, but is actually a valid domain
		"192.0x00A80001",     // mixed decimal / hex IP v4
		"0300.0250.0340.001", // octal IP v4
		"1.2.3.00",           // leading 0, valid octal value
	}
}

func TestFormatHostname(t *testing.T) {
	hostname := Hostname("somewhere.com")
	str := string("somewhere.com")

	testStringFormat(t, &hostname, "hostname", str, []string{}, invalidHostnames())
	testStringFormat(t, &hostname, "hostname", str, validHostnames(), []string{})
}

func TestFormatIPv4(t *testing.T) {
	ipv4 := IPv4("192.168.254.1")
	str := string("192.168.254.2")
	testStringFormat(t, &ipv4, "ipv4", str, []string{}, []string{"198.168.254.2.2"})
}

func TestFormatIPv6(t *testing.T) {
	ipv6 := IPv6("::1")
	str := string("::2")
	// TODO: test ipv6 zones
	testStringFormat(t, &ipv6, "ipv6", str, []string{}, []string{"127.0.0.1"})
}

func TestFormatCIDR(t *testing.T) {
	cidr := CIDR("192.168.254.1/24")
	str := string("192.168.254.2/24")
	testStringFormat(t, &cidr, "cidr", str, []string{"192.0.2.1/24", "2001:db8:a0b:12f0::1/32"}, []string{"198.168.254.2", "2001:db8:a0b:12f0::1"})
}

func TestFormatMAC(t *testing.T) {
	mac := MAC("01:02:03:04:05:06")
	str := string("06:05:04:03:02:01")
	testStringFormat(t, &mac, "mac", str, []string{}, []string{"01:02:03:04:05"})
}

func validUUID3s() []string {
	other3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhereelse.com"))

	return []string{
		other3.String(),
		strings.ReplaceAll(other3.String(), "-", ""),
	}
}

func invalidUUID3s() []string {
	other3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhereelse.com"))
	other4 := uuid.Must(uuid.NewRandom())
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))

	return []string{
		"not-a-uuid",
		other4.String(),
		other5.String(),
		strings.ReplaceAll(other4.String(), "-", ""),
		strings.ReplaceAll(other5.String(), "-", ""),
		strings.Replace(other3.String(), "-", "", 2),
		strings.Replace(other4.String(), "-", "", 2),
		strings.Replace(other5.String(), "-", "", 2),
	}
}

func TestFormatUUID3(t *testing.T) {
	first3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhere.com"))
	uuid3 := UUID3(first3.String())
	str := first3.String()
	testStringFormat(t, &uuid3, "uuid3", str,
		validUUID3s(),
		invalidUUID3s(),
	)

	// special case for zero UUID
	var uuidZero UUID3
	err := uuidZero.UnmarshalJSON([]byte(jsonNull))
	require.NoError(t, err)
	assert.Equal(t, UUID3(""), uuidZero)
}

func validUUID4s() []string {
	other4 := uuid.Must(uuid.NewRandom())

	return []string{
		other4.String(),
		strings.ReplaceAll(other4.String(), "-", ""),
	}
}

func invalidUUID4s() []string {
	other3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhere.com"))
	other4 := uuid.Must(uuid.NewRandom())
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))

	return []string{
		"not-a-uuid",
		other3.String(),
		other5.String(),
		strings.ReplaceAll(other3.String(), "-", ""),
		strings.ReplaceAll(other5.String(), "-", ""),
		strings.Replace(other3.String(), "-", "", 2),
		strings.Replace(other4.String(), "-", "", 2),
		strings.Replace(other5.String(), "-", "", 2),
	}
}
func TestFormatUUID4(t *testing.T) {
	first4 := uuid.Must(uuid.NewRandom())
	other4 := uuid.Must(uuid.NewRandom())
	uuid4 := UUID4(first4.String())
	str := other4.String()
	testStringFormat(t, &uuid4, "uuid4", str,
		validUUID4s(),
		invalidUUID4s(),
	)

	// special case for zero UUID
	var uuidZero UUID4
	err := uuidZero.UnmarshalJSON([]byte(jsonNull))
	require.NoError(t, err)
	assert.Equal(t, UUID4(""), uuidZero)
}

func validUUID5s() []string {
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))

	return []string{
		other5.String(),
		strings.ReplaceAll(other5.String(), "-", ""),
	}
}

func invalidUUID5s() []string {
	other3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhere.com"))
	other4 := uuid.Must(uuid.NewRandom())
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))

	return []string{
		"not-a-uuid",
		other3.String(),
		other4.String(),
		strings.ReplaceAll(other3.String(), "-", ""),
		strings.ReplaceAll(other4.String(), "-", ""),
		strings.Replace(other3.String(), "-", "", 2),
		strings.Replace(other4.String(), "-", "", 2),
		strings.Replace(other5.String(), "-", "", 2),
	}
}

func TestFormatUUID5(t *testing.T) {
	first5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhere.com"))
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))
	uuid5 := UUID5(first5.String())
	str := other5.String()
	testStringFormat(t, &uuid5, "uuid5", str,
		validUUID5s(),
		invalidUUID5s(),
	)

	// special case for zero UUID
	var uuidZero UUID5
	err := uuidZero.UnmarshalJSON([]byte(jsonNull))
	require.NoError(t, err)
	assert.Equal(t, UUID5(""), uuidZero)
}

func validUUID7s() []string {
	other7 := uuid.Must(uuid.NewV7())

	return []string{
		other7.String(),
		strings.ReplaceAll(other7.String(), "-", ""),
	}
}

func invalidUUID7s() []string {
	other3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhere.com"))
	other4 := uuid.Must(uuid.NewRandom())
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))

	return []string{
		"not-a-uuid",
		other3.String(),
		other4.String(),
		strings.ReplaceAll(other3.String(), "-", ""),
		strings.ReplaceAll(other4.String(), "-", ""),
		strings.Replace(other3.String(), "-", "", 2),
		strings.Replace(other4.String(), "-", "", 2),
		strings.Replace(other5.String(), "-", "", 2),
	}
}

func TestFormatUUID7(t *testing.T) {
	first7 := uuid.Must(uuid.NewV7())
	str := first7.String()
	uuid7 := UUID7(str)
	testStringFormat(t, &uuid7, "uuid7", str,
		validUUID7s(),
		invalidUUID7s(),
	)

	// special case for zero UUID
	var uuidZero UUID7
	err := uuidZero.UnmarshalJSON([]byte(jsonNull))
	require.NoError(t, err)
	assert.Equal(t, UUID7(""), uuidZero)
}

func validUUIDs() []string {
	other3 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))
	other4 := uuid.Must(uuid.NewRandom())
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))
	other6 := uuid.Must(uuid.NewV6())
	other7 := uuid.Must(uuid.NewV7())
	microsoft := "0" + other4.String() + "f"

	return []string{
		other3.String(),
		other4.String(),
		other5.String(),
		strings.ReplaceAll(other3.String(), "-", ""),
		strings.ReplaceAll(other4.String(), "-", ""),
		strings.ReplaceAll(other5.String(), "-", ""),
		other6.String(),
		other7.String(),
		microsoft,
	}
}

func invalidUUIDs() []string {
	other3 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))
	other4 := uuid.Must(uuid.NewRandom())
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))

	return []string{
		"not-a-uuid",
		strings.Replace(other3.String(), "-", "", 2),
		strings.Replace(other4.String(), "-", "", 2),
		strings.Replace(other5.String(), "-", "", 2),
	}
}
func TestFormatUUID(t *testing.T) {
	first5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhere.com"))
	other5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhereelse.com"))
	uuid := UUID(first5.String())
	str := other5.String()
	testStringFormat(t, &uuid, "uuid", str,
		validUUIDs(),
		invalidUUIDs(),
	)

	// special case for zero UUID
	var uuidZero UUID
	err := uuidZero.UnmarshalJSON([]byte(jsonNull))
	require.NoError(t, err)
	assert.Equal(t, UUID(""), uuidZero)
}

func TestFormatISBN(t *testing.T) {
	isbn := ISBN("0321751043")
	str := string("0321751043")
	testStringFormat(t, &isbn, "isbn", str, []string{}, []string{"836217463"}) // bad checksum
}

func TestFormatISBN10(t *testing.T) {
	isbn10 := ISBN10("0321751043")
	str := string("0321751043")
	testStringFormat(t, &isbn10, "isbn10", str, []string{}, []string{"836217463"}) // bad checksum
}

func TestFormatISBN13(t *testing.T) {
	isbn13 := ISBN13("978-0321751041")
	str := string("978-0321751041")
	testStringFormat(t, &isbn13, "isbn13", str, []string{}, []string{"978-0321751042"}) // bad checksum
}

func TestFormatHexColor(t *testing.T) {
	hexColor := HexColor("#FFFFFF")
	str := string("#000000")
	testStringFormat(t, &hexColor, "hexcolor", str, []string{}, []string{"#fffffffz"})
}

func TestFormatRGBColor(t *testing.T) {
	rgbColor := RGBColor("rgb(255,255,255)")
	str := string("rgb(0,0,0)")
	testStringFormat(t, &rgbColor, "rgbcolor", str, []string{}, []string{"rgb(300,0,0)"})
}

func TestFormatSSN(t *testing.T) {
	ssn := SSN("111-11-1111")
	str := string("999 99 9999")
	testStringFormat(t, &ssn, "ssn", str, []string{}, []string{"999 99 999"})
}

func TestFormatCreditCard(t *testing.T) {
	creditCard := CreditCard("4111-1111-1111-1111")
	str := string("4012-8888-8888-1881")
	testStringFormat(t, &creditCard, "creditcard", str, []string{}, []string{"9999-9999-9999-999"})
}

func TestFormatPassword(t *testing.T) {
	password := Password("super secret stuff here")
	testStringFormat(t, &password, "password", "super secret!!!", []string{"even more secret"}, []string{})
}

func TestFormatBase64(t *testing.T) {
	const b64 string = "This is a byte array with unprintable chars, but it also isn"
	str := base64.URLEncoding.EncodeToString([]byte(b64))
	b := []byte(b64)
	expected := Base64(b)
	bj := []byte("\"" + str + "\"")

	var subj Base64
	err := subj.UnmarshalText([]byte(str))
	require.NoError(t, err)
	assert.Equal(t, expected, subj)

	b, err = subj.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte(str), b)

	var subj2 Base64
	err = subj2.UnmarshalJSON(bj)
	require.NoError(t, err)
	assert.Equal(t, expected, subj2)

	b, err = subj2.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, bj, b)

	testValid(t, "byte", str)
	testInvalid(t, "byte", "ZWxpemFiZXRocG9zZXk") // missing pad char

	// Valuer interface
	sqlvalue, err := subj2.Value()
	require.NoError(t, err)
	sqlvalueAsString, ok := sqlvalue.(string)
	if assert.Truef(t, ok, "[%s]Value: expected driver value to be a string", "byte") {
		assert.Equal(t, str, sqlvalueAsString, "[%s]Value: expected %v and %v to be equal", "byte", sqlvalue, str)
	}
	// Scanner interface
	var subj3 Base64
	err = subj3.Scan([]byte(str))
	require.NoError(t, err)
	assert.Equal(t, str, subj3.String())

	var subj4 Base64
	err = subj4.Scan(str)
	require.NoError(t, err)
	assert.Equal(t, str, subj4.String())

	err = subj4.Scan(123)
	require.Error(t, err)
}

type testableFormat interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
	json.Marshaler
	json.Unmarshaler
	fmt.Stringer
	sql.Scanner
	driver.Valuer
}

func testStringFormat(t *testing.T, what testableFormat, format, with string, validSamples, invalidSamples []string) {
	t.Helper()

	// text encoding interface
	b := []byte(with)
	err := what.UnmarshalText(b)
	require.NoError(t, err)

	val := reflect.Indirect(reflect.ValueOf(what))
	strVal := val.String()
	assert.Equalf(t, with, strVal, "[%s]UnmarshalText: expected %v and %v to be value equal", format, strVal, with)

	b, err = what.MarshalText()
	require.NoError(t, err)
	assert.Equalf(t, []byte(with), b, "[%s]MarshalText: expected %v and %v to be value equal as []byte", format, string(b), with)

	// Stringer
	strVal = what.String()
	assert.Equalf(t, []byte(with), b, "[%s]String: expected %v and %v to be equal", format, strVal, with)

	// JSON encoding interface
	bj := []byte("\"" + with + "\"")
	err = what.UnmarshalJSON(bj)
	require.NoError(t, err)
	val = reflect.Indirect(reflect.ValueOf(what))
	strVal = val.String()
	assert.Equal(t, with, strVal, "[%s]UnmarshalJSON: expected %v and %v to be value equal", format, strVal, with)

	b, err = what.MarshalJSON()
	require.NoError(t, err)
	assert.Equalf(t, bj, b, "[%s]MarshalJSON: expected %v and %v to be value equal as []byte", format, string(b), with)

	// Scanner interface
	resetValue(t, format, what)
	err = what.Scan(with)
	require.NoError(t, err)
	val = reflect.Indirect(reflect.ValueOf(what))
	strVal = val.String()
	assert.Equal(t, with, strVal, "[%s]Scan: expected %v and %v to be value equal", format, strVal, with)

	err = what.Scan([]byte(with))
	require.NoError(t, err)
	val = reflect.Indirect(reflect.ValueOf(what))
	strVal = val.String()
	assert.Equal(t, with, strVal, "[%s]Scan: expected %v and %v to be value equal", format, strVal, with)

	err = what.Scan(123)
	require.Error(t, err)

	// Valuer interface
	sqlvalue, err := what.Value()
	require.NoError(t, err)
	sqlvalueAsString, ok := sqlvalue.(string)
	if assert.Truef(t, ok, "[%s]Value: expected driver value to be a string", format) {
		assert.Equal(t, with, sqlvalueAsString, "[%s]Value: expected %v and %v to be equal", format, sqlvalue, with)
	}

	// validation with Registry
	for _, valid := range append(validSamples, with) {
		testValid(t, format, valid)
	}

	for _, invalid := range invalidSamples {
		testInvalid(t, format, invalid)
	}
}

func resetValue(t *testing.T, format string, what encoding.TextUnmarshaler) {
	t.Helper()

	err := what.UnmarshalText([]byte("reset value"))
	require.NoError(t, err)
	val := reflect.Indirect(reflect.ValueOf(what))
	strVal := val.String()
	assert.Equalf(t, "reset value", strVal, "[%s]UnmarshalText: expected %v and %v to be equal (reset value) ", format, strVal, "reset value")
}

func testValid(t *testing.T, name, value string) {
	t.Helper()

	ok := Default.Validates(name, value)
	if !ok {
		t.Errorf("expected %q of type %s to be valid", value, name)
	}
}

func testInvalid(t *testing.T, name, value string) {
	t.Helper()

	ok := Default.Validates(name, value)
	if ok {
		t.Errorf("expected %q of type %s to be invalid", value, name)
	}
}

func TestDeepCopyBase64(t *testing.T) {
	b64 := Base64("ZWxpemFiZXRocG9zZXk=")
	in := &b64

	out := new(Base64)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *Base64
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyURI(t *testing.T) {
	uri := URI("http://somewhere.com")
	in := &uri

	out := new(URI)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *URI
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyEmail(t *testing.T) {
	email := Email("somebody@somewhere.com")
	in := &email

	out := new(Email)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *Email
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyHostname(t *testing.T) {
	hostname := Hostname("somewhere.com")
	in := &hostname

	out := new(Hostname)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *Hostname
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyIPv4(t *testing.T) {
	ipv4 := IPv4("192.168.254.1")
	in := &ipv4

	out := new(IPv4)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *IPv4
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyIPv6(t *testing.T) {
	ipv6 := IPv6("::1")
	in := &ipv6

	out := new(IPv6)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *IPv6
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyCIDR(t *testing.T) {
	cidr := CIDR("192.0.2.1/24")
	in := &cidr

	out := new(CIDR)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *CIDR
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyMAC(t *testing.T) {
	mac := MAC("01:02:03:04:05:06")
	in := &mac

	out := new(MAC)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *MAC
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyUUID(t *testing.T) {
	first5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhere.com"))
	uuid := UUID(first5.String())
	in := &uuid

	out := new(UUID)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *UUID
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyUUID3(t *testing.T) {
	first3 := uuid.NewMD5(uuid.NameSpaceURL, []byte("somewhere.com"))
	uuid3 := UUID3(first3.String())
	in := &uuid3

	out := new(UUID3)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *UUID3
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyUUID4(t *testing.T) {
	first4 := uuid.Must(uuid.NewRandom())
	uuid4 := UUID4(first4.String())
	in := &uuid4

	out := new(UUID4)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *UUID4
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyUUID5(t *testing.T) {
	first5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("somewhere.com"))
	uuid5 := UUID5(first5.String())
	in := &uuid5

	out := new(UUID5)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *UUID5
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyUUID7(t *testing.T) {
	first7 := uuid.Must(uuid.NewV7())
	uuid7 := UUID7(first7.String())
	in := &uuid7

	out := new(UUID7)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *UUID7
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyISBN(t *testing.T) {
	isbn := ISBN("0321751043")
	in := &isbn

	out := new(ISBN)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *ISBN
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyISBN10(t *testing.T) {
	isbn10 := ISBN10("0321751043")
	in := &isbn10

	out := new(ISBN10)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *ISBN10
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyISBN13(t *testing.T) {
	isbn13 := ISBN13("978-0321751041")
	in := &isbn13

	out := new(ISBN13)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *ISBN13
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyCreditCard(t *testing.T) {
	creditCard := CreditCard("4111-1111-1111-1111")
	in := &creditCard

	out := new(CreditCard)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *CreditCard
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopySSN(t *testing.T) {
	ssn := SSN("111-11-1111")
	in := &ssn

	out := new(SSN)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *SSN
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyHexColor(t *testing.T) {
	hexColor := HexColor("#FFFFFF")
	in := &hexColor

	out := new(HexColor)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *HexColor
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyRGBColor(t *testing.T) {
	rgbColor := RGBColor("rgb(255,255,255)")
	in := &rgbColor

	out := new(RGBColor)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *RGBColor
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestDeepCopyPassword(t *testing.T) {
	password := Password("super secret stuff here")
	in := &password

	out := new(Password)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *Password
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func BenchmarkIsUUID(b *testing.B) {
	const sampleSize = 100
	rxUUID := regexp.MustCompile(UUIDPattern)
	rxUUID3 := regexp.MustCompile(UUID3Pattern)
	rxUUID4 := regexp.MustCompile(UUID4Pattern)
	rxUUID5 := regexp.MustCompile(UUID5Pattern)

	uuids := make([]string, 0, sampleSize)
	uuid3s := make([]string, 0, sampleSize)
	uuid4s := make([]string, 0, sampleSize)
	uuid5s := make([]string, 0, sampleSize)

	for range sampleSize {
		seed := []byte(uuid.Must(uuid.NewRandom()).String())
		uuids = append(uuids, uuid.Must(uuid.NewRandom()).String())
		uuid3s = append(uuid3s, uuid.NewMD5(uuid.NameSpaceURL, seed).String())
		uuid4s = append(uuid4s, uuid.Must(uuid.NewRandom()).String())
		uuid5s = append(uuid5s, uuid.NewSHA1(uuid.NameSpaceURL, seed).String())
	}

	b.Run("IsUUID - google.uuid", benchmarkIs(uuids, IsUUID))
	b.Run("IsUUID - regexp", benchmarkIs(uuids, func(id string) bool { return rxUUID.MatchString(id) }))

	b.Run("IsUUIDv3 - google.uuid", benchmarkIs(uuid3s, IsUUID3))
	b.Run("IsUUIDv3 - regexp", benchmarkIs(uuid3s, func(id string) bool { return rxUUID3.MatchString(id) }))

	b.Run("IsUUIDv4 - google.uuid", benchmarkIs(uuid4s, IsUUID4))
	b.Run("IsUUIDv4 - regexp", benchmarkIs(uuid4s, func(id string) bool { return rxUUID4.MatchString(id) }))

	b.Run("IsUUIDv5 - google.uuid", benchmarkIs(uuid5s, IsUUID5))
	b.Run("IsUUIDv5 - regexp", benchmarkIs(uuid5s, func(id string) bool { return rxUUID5.MatchString(id) }))
}

func benchmarkIs(input []string, fn func(string) bool) func(*testing.B) {
	return func(b *testing.B) {
		var isTrue bool
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			isTrue = fn(input[b.N%len(input)])
		}
		fmt.Fprintln(io.Discard, isTrue)
	}
}

func BenchmarkIsHostname(b *testing.B) {
	hostnames := []string{
		"somewhere.com",
		"888.com",
		"a.com",
		"a.b.com",
		"a.b.c.com",
		"a.b.c.d.com",
		"a.b.c.d.e.com",
		"1.com",
		"1.2.com",
		"1.2.3.com",
		"1.2.3.4.com",
		"99.domain.com",
		"99.99.domain.com",
		"1wwworg.example.com",
		"1000wwworg.example.com",
		"xn--bcher-kva.example.com",
		"xn-80ak6aa92e.co",
		"xn-80ak6aa92e.com",
		"xn--ls8h.la",
		"☁→❄→☃→☀→☺→☂→☹→✝.ws",
		"www.example.onion",
		"www.example.ôlà",
		"ôlà.ôlà",
		"ôlà.ôlà.ôlà",
		"ex$ample",
		"localhost",
		"example",
		"x",
		"x-y",
		"a.b.c.dot",
		"www.example.org",
		"a.b.c.d.e.f.g.dot",
		"ex=ample.com",
		"<foo>",
		"www.example-hyphenated.org",
		"www.詹姆斯.org",
		"www.élégigôö.org",
		"www.詹姆斯.london",
	}
	rxHostname := regexp.MustCompile(HostnamePattern)

	b.Run("IsHostname - regexp", benchmarkIs(hostnames, func(str string) bool {
		// regexp-based version of IsHostname
		if !rxHostname.MatchString(str) {
			return false
		}

		const maxHostnameLength = 255
		if len(str) > maxHostnameLength {
			return false
		}

		const maxNodeLength = 63
		parts := strings.Split(str, ".")
		valid := true
		for _, p := range parts {
			if len(p) > maxNodeLength {
				valid = false
			}
		}
		return valid

	}))
	b.Run("IsHostname - idna", benchmarkIs(hostnames, IsHostname))
}
