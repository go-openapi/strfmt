// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package strfmt

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestBSONObjectId_fullCycle(t *testing.T) {
	const str = "507f1f77bcf86cd799439011"

	id := NewObjectId("507f1f77bcf86cd799439011")
	bytes, err := id.MarshalText()
	require.NoError(t, err)

	require.True(t, IsBSONObjectID(str))

	var idCopy ObjectId

	err = idCopy.Scan(bytes)
	require.NoError(t, err)
	assert.Equal(t, id, idCopy)

	err = idCopy.UnmarshalText(bytes)
	require.NoError(t, err)
	assert.Equal(t, id, idCopy)

	require.Equal(t, str, idCopy.String())

	jsonBytes, err := id.MarshalJSON()
	require.NoError(t, err)

	err = idCopy.UnmarshalJSON(jsonBytes)
	require.NoError(t, err)
	assert.Equal(t, id, idCopy)

	bsonBytes, err := bson.Marshal(&id)
	require.NoError(t, err)

	err = bson.Unmarshal(bsonBytes, &idCopy)
	require.NoError(t, err)
	assert.Equal(t, id, idCopy)

}

func TestDeepCopyObjectId(t *testing.T) {
	id := NewObjectId("507f1f77bcf86cd799439011")
	in := &id

	out := new(ObjectId)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *ObjectId
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}
