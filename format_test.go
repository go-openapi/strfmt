// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package strfmt

import (
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

type testFormat string

func (t testFormat) MarshalText() ([]byte, error) {
	return []byte(string(t)), nil
}

func (t *testFormat) UnmarshalText(b []byte) error {
	*t = testFormat(string(b))
	return nil
}

func (t testFormat) String() string {
	return string(t)
}

func isTestFormat(s string) bool {
	return strings.HasPrefix(s, "tf")
}

type tf2 string

func (t tf2) MarshalText() ([]byte, error) {
	return []byte(string(t)), nil
}

func (t *tf2) UnmarshalText(b []byte) error {
	*t = tf2(string(b))
	return nil
}

func istf2(s string) bool {
	return strings.HasPrefix(s, "af")
}

func (t tf2) String() string {
	return string(t)
}

type bf string

func (t bf) MarshalText() ([]byte, error) {
	return []byte(string(t)), nil
}

func (t *bf) UnmarshalText(b []byte) error {
	*t = bf(string(b))
	return nil
}

func (t bf) String() string {
	return string(t)
}

func isbf(s string) bool {
	return strings.HasPrefix(s, "bf")
}

func istf3(s string) bool {
	return strings.HasPrefix(s, "ff")
}

func init() {
	tf := testFormat("")
	Default.Add("test-format", &tf, isTestFormat)
}

func TestFormatRegistry(t *testing.T) {
	f2 := tf2("")
	f3 := bf("")
	registry := NewFormats()

	assert.True(t, registry.ContainsName("test-format"))
	assert.True(t, registry.ContainsName("testformat"))
	assert.False(t, registry.ContainsName("ttt"))

	assert.True(t, registry.Validates("testformat", "tfa"))
	assert.False(t, registry.Validates("testformat", "ffa"))

	assert.True(t, registry.Add("tf2", &f2, istf2))
	assert.True(t, registry.ContainsName("tf2"))
	assert.False(t, registry.ContainsName("tfw"))
	assert.True(t, registry.Validates("tf2", "afa"))

	assert.False(t, registry.Add("tf2", &f3, isbf))
	assert.True(t, registry.ContainsName("tf2"))
	assert.False(t, registry.ContainsName("tfw"))
	assert.True(t, registry.Validates("tf2", "bfa"))
	assert.False(t, registry.Validates("tf2", "afa"))

	assert.False(t, registry.Add("tf2", &f2, istf2))
	assert.True(t, registry.Add("tf3", &f2, istf3))
	assert.True(t, registry.ContainsName("tf3"))
	assert.True(t, registry.ContainsName("tf2"))
	assert.False(t, registry.ContainsName("tfw"))
	assert.True(t, registry.Validates("tf3", "ffa"))

	assert.True(t, registry.DelByName("tf3"))
	assert.True(t, registry.Add("tf3", &f2, istf3))

	assert.True(t, registry.DelByName("tf3"))
	assert.False(t, registry.DelByName("unknown"))
	assert.False(t, registry.Validates("unknown", ""))
}

type testStruct struct {
	D          Date       `json:"d,omitempty"`
	DT         DateTime   `json:"dt,omitempty"`
	Dur        Duration   `json:"dur,omitempty"`
	URI        URI        `json:"uri,omitempty"`
	Eml        Email      `json:"eml,omitempty"`
	UUID       UUID       `json:"uuid,omitempty"`
	UUID3      UUID3      `json:"uuid3,omitempty"`
	UUID4      UUID4      `json:"uuid4,omitempty"`
	UUID5      UUID5      `json:"uuid5,omitempty"`
	Hn         Hostname   `json:"hn,omitempty"`
	Ipv4       IPv4       `json:"ipv4,omitempty"`
	Ipv6       IPv6       `json:"ipv6,omitempty"`
	Mac        MAC        `json:"mac,omitempty"`
	Isbn       ISBN       `json:"isbn,omitempty"`
	Isbn10     ISBN10     `json:"isbn10,omitempty"`
	Isbn13     ISBN13     `json:"isbn13,omitempty"`
	Creditcard CreditCard `json:"creditcard,omitempty"`
	Ssn        SSN        `json:"ssn,omitempty"`
	Hexcolor   HexColor   `json:"hexcolor,omitempty"`
	Rgbcolor   RGBColor   `json:"rgbcolor,omitempty"`
	B64        Base64     `json:"b64,omitempty"`
	Pw         Password   `json:"pw,omitempty"`
}

func TestDecodeHook(t *testing.T) {
	registry := NewFormats()
	m := map[string]interface{}{
		"d":          "2014-12-15",
		"dt":         "2012-03-02T15:06:05.999999999Z",
		"dur":        "5s",
		"uri":        "http://www.dummy.com",
		"eml":        "dummy@dummy.com",
		"uuid":       "a8098c1a-f86e-11da-bd1a-00112444be1e",
		"uuid3":      "bcd02e22-68f0-3046-a512-327cca9def8f",
		"uuid4":      "025b0d74-00a2-4048-bf57-227c5111bb34",
		"uuid5":      "886313e1-3b8a-5372-9b90-0c9aee199e5d",
		"hn":         "somewhere.com",
		"ipv4":       "192.168.254.1",
		"ipv6":       "::1",
		"mac":        "01:02:03:04:05:06",
		"isbn":       "0321751043",
		"isbn10":     "0321751043",
		"isbn13":     "978-0321751041",
		"hexcolor":   "#FFFFFF",
		"rgbcolor":   "rgb(255,255,255)",
		"pw":         "super secret stuff here",
		"ssn":        "111-11-1111",
		"creditcard": "4111-1111-1111-1111",
		"b64":        "ZWxpemFiZXRocG9zZXk=",
	}

	date, _ := time.Parse(RFC3339FullDate, "2014-12-15")
	dur, _ := ParseDuration("5s")
	dt, _ := ParseDateTime("2012-03-02T15:06:05.999999999Z")

	exp := &testStruct{
		D:          Date(date),
		DT:         dt,
		Dur:        Duration(dur),
		URI:        URI("http://www.dummy.com"),
		Eml:        Email("dummy@dummy.com"),
		UUID:       UUID("a8098c1a-f86e-11da-bd1a-00112444be1e"),
		UUID3:      UUID3("bcd02e22-68f0-3046-a512-327cca9def8f"),
		UUID4:      UUID4("025b0d74-00a2-4048-bf57-227c5111bb34"),
		UUID5:      UUID5("886313e1-3b8a-5372-9b90-0c9aee199e5d"),
		Hn:         Hostname("somewhere.com"),
		Ipv4:       IPv4("192.168.254.1"),
		Ipv6:       IPv6("::1"),
		Mac:        MAC("01:02:03:04:05:06"),
		Isbn:       ISBN("0321751043"),
		Isbn10:     ISBN10("0321751043"),
		Isbn13:     ISBN13("978-0321751041"),
		Creditcard: CreditCard("4111-1111-1111-1111"),
		Ssn:        SSN("111-11-1111"),
		Hexcolor:   HexColor("#FFFFFF"),
		Rgbcolor:   RGBColor("rgb(255,255,255)"),
		B64:        Base64("ZWxpemFiZXRocG9zZXk="),
		Pw:         Password("super secret stuff here"),
	}

	test := new(testStruct)
	cfg := &mapstructure.DecoderConfig{
		DecodeHook: registry.MapStructureHookFunc(),
		// weakly typed will pass if this passes
		WeaklyTypedInput: false,
		Result:           test,
	}
	d, err := mapstructure.NewDecoder(cfg)
	assert.Nil(t, err)
	err = d.Decode(m)
	assert.Nil(t, err)
	assert.Equal(t, exp, test)
}

func TestMoreHostname(t *testing.T) {
	invalidHostnames := []string{
		"1wwworg.example.com",
		"www.example-.org",
		"www.--example.org",
		"www.example_dashed.org",
	}
	validHostnames := []string{
		"www.example.org",
		"a.b.c.d",
		"www.example-hyphenated.org",
		// TODO: localized hostnames
		//"www.詹姆斯.org",
		//"www.élégigôö.org",
	}

	for _, invHostname := range invalidHostnames {
		res := Default.Validates("hostname", invHostname)
		if assert.Falsef(t, res, "expected %q to be an invalid hostname", invHostname) {
			t.Logf("%q is an invalid hostname as expected", invHostname)
		}
	}
	for _, validHostname := range validHostnames {
		res := Default.Validates("hostname", validHostname)
		assert.True(t, res, "expected %q to be a valid hostname", validHostname)
	}
}

// TestMoreURI borrows from other URI validators to exercise strict RFC3986
// conforance (taken from .Net, perl, python, )
func TestMoreURI(t *testing.T) {
	invalidURIs := []string{
		// this test comes from the format test in JSONSchema-test suite
		"//foo.bar/?baz=qux#quux", // missing scheme and //

		// from https://docs.microsoft.com/en-gb/dotnet/api/system.uri.iswellformeduristring?view=netframework-4.7.2#System_Uri_IsWellFormedUriString_System_String_System_UriKind_
		"http://www.contoso.com/path???/file name", // The string is not correctly escaped.
		"c:\\directory\filename",                   // The string is an absolute Uri that represents an implicit file Uri.
		"http:\\host/path/file",                    // The string contains unescaped backslashes even if they will be treated as forward slashes
		"www.contoso.com/path/file",                // The string represents a hierarchical absolute Uri and does not contain "://"
		"2013.05.29_14:33:41",                      // relative URIs with a colon (':') in their first segment are not considered well-formed.

		// from https://metacpan.org/source/SONNEN/Data-Validate-URI-0.07/t/is_uri.t
		"",
		"foo",
		"foo@bar",
		"http://<foo>",      // illegal characters
		"://bob/",           // empty scheme
		"1http://bob",       // bad scheme
		"http:////foo.html", // bad path
		"http://example.w3.org/%illegal.html",
		"http://example.w3.org/%a",     // partial escape
		"http://example.w3.org/%a/foo", // partial escape
		"http://example.w3.org/%at",    // partial escape

		// from https://github.com/python-hyper/rfc3986/blob/master/tests/test_validators.py
		"https://user:passwd@[21DA:00D3:0000:2F3B:02AA:00FF:FE28:9C5A]:8080:8090/a?query=value#fragment", // multiple ports
		"https://user:passwd@[FF02::3::5]:8080/a?query=value#fragment",                                   // invalid IPv6
		"https://user:passwd@[FADF:01%en0]:8080/a?query=value#fragment",                                  // invalid IPv6
		"https://user:passwd@256.256.256.256:8080/a?query=value#fragment",                                // invalid IPv4

		// from github.com/scalatra/rl: URI parser in scala
		"http://www.exa mple.org",
	}
	validURIs := []string{
		"file://c:/directory/filename", // .Net: The string is an absolute URI that is missing a slash before the path/ Others: valid
		// from https://metacpan.org/source/SONNEN/Data-Validate-URI-0.07/t/is_uri.t
		"http://localhost/",
		"http://example.w3.org/path%20with%20spaces.html",
		"http://example.w3.org/%20",
		"ftp://ftp.is.co.za/rfc/rfc1808.txt",
		"ftp://ftp.is.co.za/../../../rfc/rfc1808.txt",
		"http://www.ietf.org/rfc/rfc2396.txt",
		"ldap://[2001:db8::7]/c=GB?objectClass?one",
		"mailto:John.Doe@example.com",
		"news:comp.infosystems.www.servers.unix",
		"tel:+1-816-555-1212",
		"telnet://192.0.2.16:80/",
		"urn:oasis:names:specification:docbook:dtd:xml:4.1.2",
		"http://www.richardsonnen.com/",

		// from https://github.com/python-hyper/rfc3986/blob/master/tests/test_validators.py
		"ssh://ssh@git.openstack.org:22/sigmavirus24",
		"https://git.openstack.org:443/sigmavirus24",
		"ssh://git.openstack.org:22/sigmavirus24?foo=bar#fragment",
		"git://github.com",
		"https://user:passwd@[21DA:00D3:0000:2F3B:02AA:00FF:FE28:9C5A]:8080/a?query=value#fragment",
		"https://user:passwd@[::1%25lo]:8080/a?query=value#fragment",
		"https://user:passwd@[FF02:30:0:0:0:0:0:5%25en1]:8080/a?query=value#fragment",
		"https://user:passwd@127.0.0.1:8080/a?query=value#fragment",
		"https://user:passwd@http-bin.org:8080/a?query=value#fragment",

		// from github.com/scalatra/rl: URI parser in scala
		"http://www.example.org:8080",
		"http://www.example.org/",
		"http://www.詹姆斯.org/",
		"http://www.example.org/hello/world.txt",
		"http://www.example.org/hello/world.txt/?id=5&part=three",
		"http://www.example.org/hello/world.txt/?id=5&part=three#there-you-go",
		"http://www.example.org/hello/world.txt/#here-we-are",
	}

	for _, invURI := range invalidURIs {
		res := Default.Validates("uri", invURI)
		assert.Falsef(t, res, "expected %q to be an invalid URI", invURI)
	}
	for _, validURI := range validURIs {
		res := Default.Validates("uri", validURI)
		assert.True(t, res, "expected %q to be a valid URI", validURI)
	}
}
