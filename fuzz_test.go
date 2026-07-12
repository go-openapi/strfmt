// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"database/sql"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

// Systematic fuzzing of every registered format parser.
//
// Each built-in format is reachable generically through the [Default] registry,
// which knows the format's name, concrete type and validator.
//
// A single harness ([fuzzFormat]) driven by the format name therefore
// exercises the whole parsing surface of every type:
//
//   - Validates(name, input)        — the registered validator
//   - Parse(name, input)            — UnmarshalText path
//   - Scan(input) / Scan([]byte)    — sql.Scanner path
//   - UnmarshalJSON("input")        — json.Unmarshaler path
//   - String() / MarshalText()      — marshaling of a parsed value
//   - re-Parse(String())            — round-trip against the parser's own output
//
// The only property asserted is that none of these panic.
// That contract is universally true, even for parsers that normalize their input or that intentionally store
// raw strings and validate separately (URI, Email, Password, colors, ...).
//
// We deliberately do not assert "Validates ⇔ Parse succeeds": several types store the raw string in UnmarshalText and
// validate only through the separate Validator, so such an assertion would produce false failures rather
// than real findings.
//
// To keep the CI fuzz matrix small (it schedules one job per Fuzz target), formats are grouped into parser-family
// buckets (see [fuzzBuckets]), each exposed as a single Fuzz target that selects a format within its bucket from
// a fuzzed selector byte.
//
// Every registered format must have an entry in [fuzzFormatCorpus] and belong to a bucket;
// [TestFuzzFormatCoverage] fails otherwise, so this generalization cannot silently fall behind the registry as
// new formats are added.

func FuzzUUIDs(f *testing.F)    { fuzzBucket(f, fuzzBuckets()["uuids"]) }
func FuzzNetwork(f *testing.F)  { fuzzBucket(f, fuzzBuckets()["network"]) }
func FuzzNumeric(f *testing.F)  { fuzzBucket(f, fuzzBuckets()["numeric"]) }
func FuzzTemporal(f *testing.F) { fuzzBucket(f, fuzzBuckets()["temporal"]) }
func FuzzMisc(f *testing.F)     { fuzzBucket(f, fuzzBuckets()["misc"]) }

// fuzzBucket fuzzes a family of formats through a single target.
//
// The fuzzer picks the format within the bucket from a selector byte, and the input string is fed
// to the shared [fuzzFormat] harness.
// Seeds carry the selector of the format they belong to so every format's valid branch is primed.
func fuzzBucket(f *testing.F, formats []string) {
	f.Helper()
	for i, name := range formats {
		sel := byte(i)
		for _, seed := range fuzzFormatCorpus[name] {
			f.Add(sel, seed)
		}
	}
	// structural seeds are format-agnostic: add them once
	// (the fuzzer mutates the selector to spread them across the bucket).
	for _, seed := range fuzzCommonSeeds {
		f.Add(byte(0), seed)
	}
	f.Fuzz(func(t *testing.T, sel byte, input string) {
		name := formats[int(sel)%len(formats)]
		fuzzFormat(t, name, input)
	})
}

// fuzzBuckets groups the registered formats by parser family.
//
// Each bucket is exposed as a single Fuzz target, so CI schedules one fuzz job per bucket rather than one per format.
// Grouping by family keeps the inputs a bucket explores related, so the coverage-guided engine is not diluted the way
// a single all-formats target would be.
//
// Every registered format must belong to exactly one bucket; TestFuzzFormatCoverage enforces this.
func fuzzBuckets() map[string][]string {
	return map[string][]string{
		"uuids":    {"uuid", "uuid3", "uuid4", "uuid5", "uuid7"},
		"network":  {"uri", "email", "hostname", "ipv4", "ipv6", "cidr", "mac"},
		"numeric":  {"isbn", "isbn10", "isbn13", "creditcard", "ssn"},
		"temporal": {"date", "datetime", "duration"},
		"misc":     {"bsonobjectid", "hexcolor", "rgbcolor", "password", "byte", "ulid"},
	}
}

// fuzzFormat exercises the whole parsing surface of a registered format against
// a single input and asserts that none of it panics. It is the shared harness
// behind every per-format Fuzz target.
func fuzzFormat(t *testing.T, name, input string) {
	t.Helper()

	// 1. the registered validator
	require.NotPanicsf(t, func() { _ = Default.Validates(name, input) },
		"Validates(%q, %q)", name, input)

	// 2. the registry Parse path (UnmarshalText)
	var parsed any
	var perr error
	require.NotPanicsf(t, func() { parsed, perr = Default.Parse(name, input) },
		"Parse(%q, %q)", name, input)

	// 3. the sql.Scanner path, from both string and []byte sources
	require.NotPanicsf(t, func() {
		if sc, ok := newFormatValue(name).(sql.Scanner); ok {
			_ = sc.Scan(input)
		}
	}, "Scan(string) %q, %q", name, input)
	require.NotPanicsf(t, func() {
		if sc, ok := newFormatValue(name).(sql.Scanner); ok {
			_ = sc.Scan([]byte(input))
		}
	}, "Scan([]byte) %q, %q", name, input)

	// 4. the json.Unmarshaler path, with input as a JSON string
	require.NotPanicsf(t, func() {
		if ju, ok := newFormatValue(name).(json.Unmarshaler); ok {
			quoted, err := json.Marshal(input)
			if err != nil {
				return
			}
			_ = ju.UnmarshalJSON(quoted)
		}
	}, "UnmarshalJSON(%q) for %q", input, name)

	if perr != nil {
		return
	}

	// 5. round-trip of a successful parse: marshaling never panics and the
	// parser accepts its own String() output without panicking.
	require.NotPanicsf(t, func() {
		if s, ok := parsed.(fmt.Stringer); ok {
			_, _ = Default.Parse(name, s.String())
		}
	}, "re-Parse(String()) for %q, %q", name, input)
	require.NotPanicsf(t, func() {
		if m, ok := parsed.(encoding.TextMarshaler); ok {
			_, _ = m.MarshalText()
		}
	}, "MarshalText() for %q, %q", name, input)
}

// newFormatValue returns a fresh pointer to the concrete type registered under
// name (e.g. *UUID for "uuid"), or nil when the format is unknown.
func newFormatValue(name string) any {
	tpe, ok := Default.GetType(name)
	if !ok {
		return nil
	}
	return reflect.New(tpe).Interface()
}

// TestFuzzFormatCoverage guards the generalization.
//
// Every format registered in the default registry must have a fuzz corpus AND belong to a fuzz bucket (and
// therefore be fuzzed by one of the Fuzz targets below). Adding a new format without wiring it up here fails this test.
//
// The special "testformat" registered by format_test.go's init is a test artifact, not a real format, and is skipped.
func TestFuzzFormatCoverage(t *testing.T) {
	registry, ok := Default.(*defaultFormats)
	require.TrueT(t, ok)

	// each format appears in exactly one bucket
	bucketOf := make(map[string]string)
	for bucket, formats := range fuzzBuckets() {
		for _, name := range formats {
			if prev, dup := bucketOf[name]; dup {
				require.Failf(t, "duplicate bucket assignment",
					"format %q is in both buckets %q and %q", name, prev, bucket)
			}
			bucketOf[name] = bucket
		}
	}

	for _, kf := range registry.data {
		if kf.Name == "testformat" {
			continue // test-only format registered by format_test.go
		}
		_, hasCorpus := fuzzFormatCorpus[kf.Name]
		require.Truef(t, hasCorpus,
			"format %q is registered but has no fuzz corpus in fuzz_test.go", kf.Name)
		_, hasBucket := bucketOf[kf.Name]
		require.Truef(t, hasBucket,
			"format %q is registered but is not assigned to a fuzz bucket in fuzz_test.go", kf.Name)
	}

	// the reverse direction: no stale or mistyped corpus / bucket entries
	for name := range fuzzFormatCorpus {
		require.Truef(t, registry.ContainsName(name),
			"fuzz corpus references %q, which is not a registered format", name)
	}
	for name := range bucketOf {
		require.Truef(t, registry.ContainsName(name),
			"fuzz bucket references %q, which is not a registered format", name)
		_, hasCorpus := fuzzFormatCorpus[name]
		require.Truef(t, hasCorpus,
			"format %q is bucketed but has no fuzz corpus", name)
	}
}

// fuzzCommonSeeds are structural inputs applied to every fuzz target, independently of the format-specific corpus.
//
// They prime the corpus with the kind of cruft that breaks naive parsers.
//
//nolint:gochecknoglobals // shared, read-only seed corpus for the fuzz targets
var fuzzCommonSeeds = []string{
	"",
	" ",
	"               ",
	"\x00",
	"a\x00b",
	"\t\n\r",
	"\ufeff",   // BOM
	"\xff\xfe", // invalid UTF-8
	"-",
	"+",
	"0",
	"null",
	"[]",
	"字", //nolint:gosmopolitan // deliberately a non-ASCII / non-Latin seed
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // long-ish
}

// fuzzFormatCorpus maps each registered format name to seed inputs.
// Seeds mix valid, borderline and known-tricky invalid values so the fuzzer
// starts from both sides of every validator.
//
//nolint:gochecknoglobals // shared, read-only seed corpus for the fuzz targets
var fuzzFormatCorpus = map[string][]string{
	"bsonobjectid": {
		"507f1f77bcf86cd799439011",
		"507F1F77BCF86CD799439011",
		"507f1f77bcf86cd79943901",   // too short
		"507f1f77bcf86cd7994390111", // too long
		"zzzzzzzzzzzzzzzzzzzzzzzz",  // non-hex
	},
	"date": {
		"2014-12-15",
		"0001-01-01",
		"2014-12-15T00:00:00Z", // date-time, not date
		"2014-13-15",           // invalid month
		"15-12-2014",           // wrong order
	},
	"datetime": {
		"2011-08-18T18:03:37.123Z",
		"2011-08-18T19:03:37.123+01:00",
		"2011-08-18T19:03:37.123+0100",
		"2014-12-15T08:00:00.000Z",
		"2014-12-15T08:00", // missing seconds/zone
		"2014-12-15",       // date only
	},
	"duration": {
		"1s",
		"300ms",
		"-1.5h",
		"2h45m",
		".5 week",
		"2 minutes 45 seconds",
		"300 ms",
		"   1s            ",
		"x",
		"1s1s1s               1s   1s            ",
	},
	"uri": {
		"http://foo.bar/baz?q=1#frag",
		"mailto:a@b.c",
		"urn:isbn:0451450523",
		"//no-scheme",
		"http://[::1]:8080/",
	},
	"email": {
		"somebody@somewhere.com",
		"a@b.c",
		"somebody@somewhere@com",
		"no-at-sign",
		"“weird”@example.com",
	},
	"hostname": {
		"www.example.com",
		"localhost",
		"www.xn--1b4c3d.london",
		"-leadinghyphen.com",
		"toolonglabel-" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com",
	},
	"ipv4": {
		"192.168.254.1",
		"0.0.0.0",
		"255.255.255.255",
		"198.168.254.2.2",
		"::1", // v6, not v4
	},
	"ipv6": {
		"::1",
		"2001:db8:a0b:12f0::1",
		"fe80::1%eth0",
		"127.0.0.1", // v4, not v6
		"::g",
	},
	"cidr": {
		"192.0.2.1/24",
		"2001:db8:a0b:12f0::1/32",
		"198.168.254.2", // missing prefix
		"192.0.2.1/33",  // prefix out of range
	},
	"mac": {
		"01:02:03:04:05:06",
		"01-02-03-04-05-06",
		"0102.0304.0506",
		"01:02:03:04:05", // too short
	},
	"uuid": {
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"6ba7b8109dad11d180b400c04fd430c8",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479",
		"not-a-uuid",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
	},
	"uuid3": {
		"6fa459ea-ee8a-3ca4-894e-db77e160355e",
		"6fa459eaee8a3ca4894edb77e160355e",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479", // v4, not v3
	},
	"uuid4": {
		"f47ac10b-58cc-4372-a567-0e02b2c3d479",
		"f47ac10b58cc4372a5670e02b2c3d479",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8", // v1, not v4
	},
	"uuid5": {
		"886313e1-3b8a-5372-9b90-0c9aee199e5d",
		"886313e13b8a53729b900c9aee199e5d",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479", // v4, not v5
	},
	"uuid7": {
		"01890a5d-ac96-774b-bcce-b302099a8057",
		"01890a5dac96774bbcceb302099a8057",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479", // v4, not v7
	},
	"isbn": {
		"0321751043",
		"978-0321751041",
		"836217463", // bad checksum
	},
	"isbn10": {
		"0321751043",
		"0-321-75104-3",
		"836217463", // bad checksum
	},
	"isbn13": {
		"978-0321751041",
		"9780321751041",
		"978-0321751042", // bad checksum
	},
	"creditcard": {
		"4111-1111-1111-1111",
		"4111111111111111",
		"9999-9999-9999-999", // bad
	},
	"ssn": {
		"111-11-1111",
		"078-05-1120",
		"111111111", // missing separators
	},
	"hexcolor": {
		"#FFFFFF",
		"#fff",
		"#000000",
		"#fffffffz", // invalid
		"FFFFFF",    // missing hash
	},
	"rgbcolor": {
		"rgb(255,255,255)",
		"rgb(0,0,0)",
		"rgb(300,0,0)", // out of range
		"rgb(1,2)",     // missing component
	},
	"password": {
		"super secret stuff here",
		"",
		"éàü",
	},
	"byte": {
		"aGVsbG8=",             // "hello", std/url compatible
		"VGhpcyBpcyBhIHRlc3Q=", // "This is a test"
		"ZWxpemFiZXRocG9zZXk",  // missing pad char
		"_-_-",                 // url-safe alphabet
		"++//",                 // std alphabet (not url-safe)
	},
	"ulid": {
		"01EYXZVGBHG26MFTG4JWR4K558",
		"00000000000000000000000000",
		"7ZZZZZZZZZZZZZZZZZZZZZZZZZ",
		"80000000000000000000000000", // overflows timestamp
		"0123456789ABCDEFGHILMNOPQR", // contains excluded letters
	},
}
