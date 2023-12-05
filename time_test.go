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
	"bytes"
	"encoding/gob"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	p, _ = time.Parse(time.RFC3339Nano, "2011-08-18T19:03:37.000000000+01:00")

	testCases = []struct {
		in     []byte    // externally sourced data -- to be unmarshalled
		time   time.Time // its representation in time.Time
		str    string    // its marshalled representation
		utcStr string    // the marshaled representation as utc
	}{
		{[]byte("2014-12-15"), time.Date(2014, 12, 15, 0, 0, 0, 0, time.UTC), "2014-12-15T00:00:00.000Z", "2014-12-15T00:00:00.000Z"},
		{[]byte("2014-12-15 08:00:00"), time.Date(2014, 12, 15, 8, 0, 0, 0, time.UTC), "2014-12-15T08:00:00.000Z", "2014-12-15T08:00:00.000Z"},
		{[]byte("2014-12-15T08:00:00"), time.Date(2014, 12, 15, 8, 0, 0, 0, time.UTC), "2014-12-15T08:00:00.000Z", "2014-12-15T08:00:00.000Z"},
		{[]byte("2014-12-15T08:00"), time.Date(2014, 12, 15, 8, 0, 0, 0, time.UTC), "2014-12-15T08:00:00.000Z", "2014-12-15T08:00:00.000Z"},
		{[]byte("2014-12-15T08:00Z"), time.Date(2014, 12, 15, 8, 0, 0, 0, time.UTC), "2014-12-15T08:00:00.000Z", "2014-12-15T08:00:00.000Z"},
		{[]byte("2018-01-28T23:54Z"), time.Date(2018, 01, 28, 23, 54, 0, 0, time.UTC), "2018-01-28T23:54:00.000Z", "2018-01-28T23:54:00.000Z"},
		{[]byte("2014-12-15T08:00:00.000Z"), time.Date(2014, 12, 15, 8, 0, 0, 0, time.UTC), "2014-12-15T08:00:00.000Z", "2014-12-15T08:00:00.000Z"},
		{[]byte("2011-08-18T19:03:37.123000000+01:00"), time.Date(2011, 8, 18, 19, 3, 37, 123*1e6, p.Location()), "2011-08-18T19:03:37.123+01:00", "2011-08-18T18:03:37.123Z"},
		{[]byte("2011-08-18T19:03:37.123000+0100"), time.Date(2011, 8, 18, 19, 3, 37, 123*1e6, p.Location()), "2011-08-18T19:03:37.123+01:00", "2011-08-18T18:03:37.123Z"},
		{[]byte("2011-08-18T19:03:37.123+0100"), time.Date(2011, 8, 18, 19, 3, 37, 123*1e6, p.Location()), "2011-08-18T19:03:37.123+01:00", "2011-08-18T18:03:37.123Z"},
		{[]byte("2014-12-15T19:30:20Z"), time.Date(2014, 12, 15, 19, 30, 20, 0, time.UTC), "2014-12-15T19:30:20.000Z", "2014-12-15T19:30:20.000Z"},
		{[]byte("0001-01-01T00:00:00Z"), time.Time{}.UTC(), "0001-01-01T00:00:00.000Z", "0001-01-01T00:00:00.000Z"},
		{[]byte(""), time.Unix(0, 0).UTC(), "1970-01-01T00:00:00.000Z", "1970-01-01T00:00:00.000Z"},
		{[]byte(nil), time.Unix(0, 0).UTC(), "1970-01-01T00:00:00.000Z", "1970-01-01T00:00:00.000Z"},
	}
)

func TestNewDateTime(t *testing.T) {
	assert.EqualValues(t, time.Unix(0, 0).UTC(), NewDateTime())
}

func TestIsZero(t *testing.T) {
	var empty DateTime
	assert.True(t, empty.IsZero())
	var nilDt *DateTime
	assert.True(t, nilDt.IsZero())
	small := DateTime(time.Unix(100, 5))
	assert.False(t, small.IsZero())

	// time.Unix(0,0) does not produce a true zero value struct,
	// so this is expected to fail.
	dt := NewDateTime()
	assert.False(t, dt.IsZero())
}

func TestIsUnixZero(t *testing.T) {
	dt := NewDateTime()
	assert.True(t, dt.IsUnixZero())
	assert.NotEqual(t, dt.IsZero(), dt.IsUnixZero())
	// Test configuring UnixZero
	estLocation := time.FixedZone("EST", int((-5 * time.Hour).Seconds()))
	estUnixZero := time.Unix(0, 0).In(estLocation)
	UnixZero = estUnixZero
	t.Cleanup(func() { UnixZero = time.Unix(0, 0).UTC() })
	dtz := DateTime(estUnixZero)
	assert.True(t, dtz.IsUnixZero())
}

func TestParseDateTime_errorCases(t *testing.T) {
	_, err := ParseDateTime("yada")
	require.Error(t, err)
}

// TestParseDateTime tests the full cycle:
// parsing -> marshalling -> unmarshalling / scanning
func TestParseDateTime_fullCycle(t *testing.T) {
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)

		parsed, err := ParseDateTime(example.str)
		require.NoError(t, err)
		assert.EqualValues(t, example.time, parsed)

		mt, err := parsed.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, []byte(example.str), mt)

		if example.str != "" {
			v := IsDateTime(example.str)
			assert.True(t, v)
		} else {
			t.Logf("IsDateTime() skipped for empty testcases")
		}

		pp := NewDateTime()
		err = pp.UnmarshalText(mt)
		require.NoError(t, err)
		assert.EqualValues(t, example.time, pp)

		pp = NewDateTime()
		err = pp.Scan(mt)
		require.NoError(t, err)
		assert.Equal(t, DateTime(example.time), pp)
	}
}

func TestDateTime_IsDateTime_errorCases(t *testing.T) {
	v := IsDateTime("zor")
	assert.False(t, v)

	v = IsDateTime("zorg")
	assert.False(t, v)

	v = IsDateTime("zorgTx")
	assert.False(t, v)

	v = IsDateTime("1972-12-31Tx")
	assert.False(t, v)

	v = IsDateTime("1972-12-31T24:40:00.000Z")
	assert.False(t, v)

	v = IsDateTime("1972-12-31T23:63:00.000Z")
	assert.False(t, v)

	v = IsDateTime("1972-12-31T23:59:60.000Z")
	assert.False(t, v)

}
func TestDateTime_UnmarshalText_errorCases(t *testing.T) {
	pp := NewDateTime()
	err := pp.UnmarshalText([]byte("yada"))
	require.Error(t, err)
	err = pp.UnmarshalJSON([]byte("yada"))
	require.Error(t, err)
}

func TestDateTime_UnmarshalText(t *testing.T) {
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)
		pp := NewDateTime()
		err := pp.UnmarshalText(example.in)
		require.NoError(t, err)
		assert.EqualValues(t, example.time, pp)

		// Other way around
		val, erv := pp.Value()
		require.NoError(t, erv)
		assert.EqualValues(t, example.str, val)

	}
}
func TestDateTime_UnmarshalJSON(t *testing.T) {
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)
		pp := NewDateTime()
		err := pp.UnmarshalJSON(esc(example.in))
		require.NoError(t, err)
		assert.EqualValues(t, example.time, pp)
	}

	// Check UnmarshalJSON failure with no lexed items
	pp := NewDateTime()
	err := pp.UnmarshalJSON([]byte("zorg emperor"))
	require.Error(t, err)

	// Check lexer failure
	err = pp.UnmarshalJSON([]byte(`"zorg emperor"`))
	require.Error(t, err)

	// Check null case
	err = pp.UnmarshalJSON([]byte("null"))
	require.NoError(t, err)
}

func esc(v []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte('"')
	buf.Write(v)
	buf.WriteByte('"')
	return buf.Bytes()
}

func TestDateTime_MarshalText(t *testing.T) {
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)
		dt := DateTime(example.time)
		mt, err := dt.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, []byte(example.str), mt)
	}
}
func TestDateTime_MarshalJSON(t *testing.T) {
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)
		dt := DateTime(example.time)
		bb, err := dt.MarshalJSON()
		require.NoError(t, err)
		assert.EqualValues(t, esc([]byte(example.str)), bb)
	}
}
func TestDateTime_MarshalJSON_Override(t *testing.T) {
	oldNormalizeMarshal := NormalizeTimeForMarshal
	defer func() {
		NormalizeTimeForMarshal = oldNormalizeMarshal
	}()

	NormalizeTimeForMarshal = func(t time.Time) time.Time {
		return t.UTC()
	}
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)
		dt := DateTime(example.time.UTC())
		bb, err := dt.MarshalJSON()
		require.NoError(t, err)
		assert.EqualValues(t, esc([]byte(example.utcStr)), bb)
	}
}

func TestDateTime_Scan(t *testing.T) {
	for caseNum, example := range testCases {
		t.Logf("Case #%d", caseNum)

		pp := NewDateTime()
		err := pp.Scan(example.in)
		require.NoError(t, err)
		assert.Equal(t, DateTime(example.time), pp)

		pp = NewDateTime()
		err = pp.Scan(string(example.in))
		require.NoError(t, err)
		assert.Equal(t, DateTime(example.time), pp)

		pp = NewDateTime()
		err = pp.Scan(example.time)
		require.NoError(t, err)
		assert.Equal(t, DateTime(example.time), pp)
	}
}

func TestDateTime_Scan_Failed(t *testing.T) {
	pp := NewDateTime()
	zero := NewDateTime()

	err := pp.Scan(nil)
	require.NoError(t, err)
	// Zero values differ...
	// assert.Equal(t, zero, pp)
	assert.Equal(t, DateTime{}, pp)

	err = pp.Scan("")
	require.NoError(t, err)
	assert.Equal(t, zero, pp)

	err = pp.Scan(int64(0))
	require.Error(t, err)

	err = pp.Scan(float64(0))
	require.Error(t, err)
}

func TestDateTime_BSON(t *testing.T) {
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

func TestDeepCopyDateTime(t *testing.T) {
	p, err := ParseDateTime("2011-08-18T19:03:37.000000000+01:00")
	require.NoError(t, err)
	in := &p

	out := new(DateTime)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *DateTime
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestGobEncodingDateTime(t *testing.T) {
	now := time.Now()

	b := bytes.Buffer{}
	enc := gob.NewEncoder(&b)
	err := enc.Encode(DateTime(now))
	require.NoError(t, err)
	assert.NotEmpty(t, b.Bytes())

	var result DateTime

	dec := gob.NewDecoder(&b)
	err = dec.Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, now.Year(), time.Time(result).Year())
	assert.Equal(t, now.Month(), time.Time(result).Month())
	assert.Equal(t, now.Day(), time.Time(result).Day())
	assert.Equal(t, now.Hour(), time.Time(result).Hour())
	assert.Equal(t, now.Minute(), time.Time(result).Minute())
	assert.Equal(t, now.Second(), time.Time(result).Second())
}

func TestDateTime_Equal(t *testing.T) {
	t.Parallel()

	dt1 := DateTime(time.Now())
	dt2 := DateTime(time.Time(dt1).Add(time.Second))

	//nolint:gocritic
	assert.True(t, dt1.Equal(dt1), "DateTime instances should be equal")
	assert.False(t, dt1.Equal(dt2), "DateTime instances should not be equal")
}
