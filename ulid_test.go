package strfmt

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

const testUlid = string("01EYXZVGBHG26MFTG4JWR4K558")
const testUlidAlt = string("01EYXZW663G7PYHVSQ8WTMDA67")

var testUlidOverrideMtx sync.Mutex
var testUlidOverrideValMtx sync.Mutex

func TestFormatULID_Text(t *testing.T) {
	t.Parallel()

	t.Run("positive", func(t *testing.T) {
		t.Parallel()
		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)

		res, err := ulid.MarshalText()
		assert.NoError(t, err)
		assert.Equal(t, testUlid, string(res))

		ulid2, _ := ParseULID(testUlidAlt)
		assert.NoError(t, err)

		what := []byte(testUlid)
		err = ulid2.UnmarshalText(what)
		assert.NoError(t, err)
		assert.Equal(t, testUlid, ulid2.String())
	})
	t.Run("negative", func(t *testing.T) {
		t.Parallel()
		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)

		what := []byte("00000000-0000-0000-0000-000000000000")

		err = ulid.UnmarshalText(what)
		assert.Error(t, err)
	})
}

func TestFormatULID_BSON(t *testing.T) {
	t.Parallel()
	t.Run("positive", func(t *testing.T) {
		t.Parallel()
		ulid, _ := ParseULID(testUlid)

		bsonData, err := bson.Marshal(&ulid)
		assert.NoError(t, err)

		var ulidUnmarshaled ULID
		err = bson.Unmarshal(bsonData, &ulidUnmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, ulid, ulidUnmarshaled)

		// Check value marshaling explicitly
		m := bson.M{"data": ulid}
		bsonData, err = bson.Marshal(&m)
		assert.NoError(t, err)

		var mUnmarshaled bson.M
		err = bson.Unmarshal(bsonData, &mUnmarshaled)
		assert.NoError(t, err)

		data, ok := m["data"].(ULID)
		assert.Equal(t, true, ok)
		assert.Equal(t, ulid, data)
	})
	t.Run("negative", func(t *testing.T) {
		t.Parallel()
		uuid := UUID("00000000-0000-0000-0000-000000000000")
		bsonData, err := bson.Marshal(&uuid)
		assert.NoError(t, err)

		var ulidUnmarshaled ULID
		err = bson.Unmarshal(bsonData, &ulidUnmarshaled)
		assert.Error(t, err)
	})
}

func TestFormatULID_JSON(t *testing.T) {
	t.Parallel()
	t.Run("positive", func(t *testing.T) {
		t.Parallel()
		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)

		whatStr := fmt.Sprintf(`"%s"`, testUlidAlt)
		what := []byte(whatStr)
		err = ulid.UnmarshalJSON(what)
		assert.NoError(t, err)
		assert.Equal(t, testUlidAlt, ulid.String())

		data, err := ulid.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, whatStr, string(data))
	})
	t.Run("null", func(t *testing.T) {
		t.Parallel()
		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)

		err = ulid.UnmarshalJSON([]byte("null"))
		assert.Nil(t, err)
	})
	t.Run("negative", func(t *testing.T) {
		t.Parallel()
		// Check UnmarshalJSON failure with no lexed items
		ulid := NewULIDZero()
		err := ulid.UnmarshalJSON([]byte("zorg emperor"))
		assert.Error(t, err)

		// Check lexer failure
		err = ulid.UnmarshalJSON([]byte(`"zorg emperor"`))
		assert.Error(t, err)
	})
}

func TestFormatULID_Scan(t *testing.T) {
	t.Parallel()
	t.Run("db.Scan", func(t *testing.T) {
		t.Parallel()
		testUlidOverrideMtx.Lock()
		defer testUlidOverrideMtx.Unlock()

		srcUlid := testUlidAlt

		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)

		err = ulid.Scan(srcUlid)
		assert.NoError(t, err)
		assert.Equal(t, srcUlid, ulid.String())

		ulid, _ = ParseULID(testUlid)
		err = ulid.Scan([]byte(srcUlid))
		assert.NoError(t, err)
		assert.Equal(t, srcUlid, ulid.String())
	})
	t.Run("db.Scan_Failed", func(t *testing.T) {
		t.Parallel()
		testUlidOverrideMtx.Lock()
		defer testUlidOverrideMtx.Unlock()

		ulid, err := ParseULID(testUlid)
		zero := NewULIDZero()
		assert.NoError(t, err)

		err = ulid.Scan(nil)
		assert.NoError(t, err)
		assert.Equal(t, zero, ulid)

		err = ulid.Scan("")
		assert.NoError(t, err)
		assert.Equal(t, zero, ulid)

		err = ulid.Scan(int64(0))
		assert.Error(t, err)

		err = ulid.Scan(float64(0))
		assert.Error(t, err)
	})
	t.Run("db.Value", func(t *testing.T) {
		t.Parallel()
		testUlidOverrideValMtx.Lock()
		defer testUlidOverrideValMtx.Unlock()

		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)

		val, err := ulid.Value()
		assert.NoError(t, err)

		assert.EqualValues(t, testUlid, val)
	})
	t.Run("override.Scan", func(t *testing.T) {
		t.Parallel()
		testUlidOverrideMtx.Lock()
		defer testUlidOverrideMtx.Unlock()

		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)
		ulid2, err := ParseULID(testUlidAlt)
		assert.NoError(t, err)

		ULIDScanOverrideFunc = func(raw interface{}) (ULID, error) {
			var u ULID = NewULIDZero()
			switch x := raw.(type) {
			case [16]byte:
				return u, u.ULID.UnmarshalBinary(x[:])
			case int: // just for linter
				return u, fmt.Errorf("cannot sql.Scan() strfmt.ULID from: %#v", raw)
			}
			return u, fmt.Errorf("cannot sql.Scan() strfmt.ULID from: %#v", raw)
		}

		// get underlying binary implementation which is actually [16]byte
		bytes := [16]byte(ulid.ULID)

		err = ulid2.Scan(bytes)
		assert.NoError(t, err)
		assert.Equal(t, ulid2, ulid)
		assert.Equal(t, ulid2.String(), ulid.String())

		// check other default cases became unreachable
		err = ulid2.Scan(testUlid)
		assert.Error(t, err)

		// return default Scan method
		ULIDScanOverrideFunc = ULIDScanDefaultFunc
		err = ulid2.Scan(testUlid)
		assert.NoError(t, err)
		assert.Equal(t, ulid2.String(), testUlid)
	})
	t.Run("override.Value", func(t *testing.T) {
		t.Parallel()
		testUlidOverrideValMtx.Lock()
		defer testUlidOverrideValMtx.Unlock()

		ulid, err := ParseULID(testUlid)
		assert.NoError(t, err)
		ulid2, err := ParseULID(testUlid)
		assert.NoError(t, err)

		ULIDValueOverrideFunc = func(u ULID) (driver.Value, error) {
			bytes := [16]byte(u.ULID)
			return driver.Value(bytes), nil
		}

		exp := [16]byte(ulid2.ULID)
		val, err := ulid.Value()
		assert.NoError(t, err)

		assert.EqualValues(t, exp, val)

		// return default Value method
		ULIDValueOverrideFunc = ULIDValueDefaultFunc

		val, err = ulid.Value()
		assert.NoError(t, err)

		assert.EqualValues(t, testUlid, val)
	})
}

func TestFormatULID_DeepCopy(t *testing.T) {
	ulid, err := ParseULID(testUlid)
	assert.NoError(t, err)
	in := &ulid

	out := new(ULID)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *ULID
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}

func TestFormatULID_GobEncoding(t *testing.T) {
	ulid, err := ParseULID(testUlid)
	assert.NoError(t, err)

	b := bytes.Buffer{}
	enc := gob.NewEncoder(&b)
	err = enc.Encode(ulid)
	assert.NoError(t, err)
	assert.NotEmpty(t, b.Bytes())

	var result ULID

	dec := gob.NewDecoder(&b)
	err = dec.Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, ulid, result)
	assert.Equal(t, ulid.String(), result.String())
}

func TestFormatULID_NewULID_and_Equal(t *testing.T) {
	t.Parallel()

	ulid1, err := NewULID()
	assert.NoError(t, err)

	ulid2, err := NewULID()
	assert.NoError(t, err)

	assert.True(t, ulid1.Equal(ulid1), "ULID instances should be equal")
	assert.False(t, ulid1.Equal(ulid2), "ULID instances should not be equal")

	ulidZero := NewULIDZero()
	ulidZero2 := NewULIDZero()
	assert.True(t, ulidZero.Equal(ulidZero2), "ULID instances should be equal")
}

func TestIsULID(t *testing.T) {
	t.Parallel()

	tcases := []struct {
		ulid   string
		expect bool
	}{
		{ulid: "01EYXZVGBHG26MFTG4JWR4K558", expect: true},
		{ulid: "01EYXZW663G7PYHVSQ8WTMDA67", expect: true},
		{ulid: "7ZZZZZZZZZ0000000000000000", expect: true},
		{ulid: "00000000000000000000000000", expect: true},
		{ulid: "7ZZZZZZZZZZZZZZZZZZZZZZZZZ", expect: true},
		{ulid: "not-a-ulid", expect: false},
		{ulid: "8000000000FJ2MMFJ3ATV3XB2C", expect: false},
		{ulid: "81EYY0NEYJZZZZZZZZZZZZZZZZ", expect: false},
		{ulid: "7ZZZZZZZZZ000000000000000U", expect: false},
		{ulid: "7ZZZZZZZZZ000000000000000L", expect: false},
		{ulid: "7ZZZZZZZZZ000000000000000O", expect: false},
		{ulid: "7ZZZZZZZZZ000000000000000I", expect: false},
	}
	for _, tcase := range tcases {
		tc := tcase
		t.Run(fmt.Sprintf("%s:%t", tc.ulid, tc.expect), func(t *testing.T) {
			t.Parallel()
			if tc.expect {
				assert.True(t, IsULID(tc.ulid))
			} else {
				assert.False(t, IsULID(tc.ulid))
			}
		})
	}

}
