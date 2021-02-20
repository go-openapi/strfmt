package conv

import "github.com/go-openapi/strfmt"

// ULID returns a pointer to of the ULID value passed in.
func ULID(v strfmt.ULID) *strfmt.ULID {
	return &v
}

// ULIDValue returns the value of the ULID pointer passed in or
// the default value if the pointer is nil.
func ULIDValue(v *strfmt.ULID) strfmt.ULID {
	if v == nil {
		return strfmt.ULID{}
	}

	return *v
}
