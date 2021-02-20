package conv

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

const testUlid = string("01EYXZVGBHG26MFTG4JWR4K558")

func TestULIDValue(t *testing.T) {
	assert.Equal(t, strfmt.ULID{}, ULIDValue(nil))

	value := strfmt.ULID{}
	err := value.UnmarshalText([]byte(testUlid))
	assert.NoError(t, err)
	assert.Equal(t, value, ULIDValue(&value))

	ulidRef := ULID(value)
	assert.Equal(t, &value, ulidRef)
}
