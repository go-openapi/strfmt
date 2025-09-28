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
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ sql.Scanner = &Date{}
var _ driver.Valuer = Date{}

func TestDate(t *testing.T) {
	pp := Date{}
	err := pp.UnmarshalText([]byte{})
	require.NoError(t, err)
	err = pp.UnmarshalText([]byte("yada"))
	require.Error(t, err)

	orig := "2014-12-15"
	bj := []byte("\"" + orig + "\"")
	err = pp.UnmarshalText([]byte(orig))
	require.NoError(t, err)

	txt, err := pp.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, orig, string(txt))

	err = pp.UnmarshalJSON(bj)
	require.NoError(t, err)
	assert.Equal(t, orig, pp.String())

	err = pp.UnmarshalJSON([]byte(`"1972/01/01"`))
	require.Error(t, err)

	b, err := pp.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, bj, b)

	var dateZero Date
	err = dateZero.UnmarshalJSON([]byte(jsonNull))
	require.NoError(t, err)
	assert.Equal(t, Date{}, dateZero)
}

func TestDate_Scan(t *testing.T) {
	ref := time.Now().Truncate(24 * time.Hour).UTC()
	date, str := Date(ref), ref.Format(RFC3339FullDate)

	values := []interface{}{str, []byte(str), ref}
	for _, value := range values {
		result := Date{}
		_ = (&result).Scan(value)
		assert.Equal(t, date, result, "value: %#v", value)
	}

	dd := Date{}
	err := dd.Scan(nil)
	require.NoError(t, err)
	assert.Equal(t, Date{}, dd)

	err = dd.Scan(19700101)
	require.Error(t, err)
}

func TestDate_Value(t *testing.T) {
	ref := time.Now().Truncate(24 * time.Hour).UTC()
	date := Date(ref)
	dbv, err := date.Value()
	require.NoError(t, err)
	assert.EqualValues(t, dbv, ref.Format("2006-01-02"))
}

func TestDate_IsDate(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"2017-12-22", true},
		{"2017-1-1", false},
		{"17-13-22", false},
		{"2017-02-29", false}, // not a valid date : 2017 is not a leap year
		{"1900-02-29", false}, // not a valid date : 1900 is not a leap year
		{"2100-02-29", false}, // not a valid date : 2100 is not a leap year
		{"2000-02-29", true},  // a valid date : 2000 is a leap year
		{"2400-02-29", true},  // a valid date : 2000 is a leap year
		{"2017-13-22", false},
		{"2017-12-32", false},
		{"20171-12-32", false},
		{"YYYY-MM-DD", false},
		{"20-17-2017", false},
		{"2017-12-22T01:02:03Z", false},
	}
	for _, test := range tests {
		assert.Equal(t, test.valid, IsDate(test.value), "value [%s] should be valid: [%t]", test.value, test.valid)
	}
}

func TestDeepCopyDate(t *testing.T) {
	ref := time.Now().Truncate(24 * time.Hour).UTC()
	date := Date(ref)
	in := &date

	out := new(Date)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *Date
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestGobEncodingDate(t *testing.T) {
	now := time.Now()

	b := bytes.Buffer{}
	enc := gob.NewEncoder(&b)
	err := enc.Encode(Date(now))
	require.NoError(t, err)
	assert.NotEmpty(t, b.Bytes())

	var result Date

	dec := gob.NewDecoder(&b)
	err = dec.Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, now.Year(), time.Time(result).Year())
	assert.Equal(t, now.Month(), time.Time(result).Month())
	assert.Equal(t, now.Day(), time.Time(result).Day())
}

func TestDate_Equal(t *testing.T) {
	t.Parallel()

	d1 := Date(time.Date(2020, 10, 11, 12, 13, 14, 15, time.UTC))
	d2 := Date(time.Date(2020, 10, 11, 12, 13, 14, 15, time.UTC))
	d3 := Date(time.Date(2020, 11, 12, 13, 14, 15, 16, time.UTC))

	//nolint:gocritic
	assert.True(t, d1.Equal(d1), "Same Date should Equal itself")
	assert.True(t, d1.Equal(d2), "Date instances should be equal")
	assert.False(t, d1.Equal(d3), "Date instances should not be equal")
}
