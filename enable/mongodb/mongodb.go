// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package mongodb replaces strfmt's built-in minimal BSON codec with one
// backed by the official MongoDB driver (go.mongodb.org/mongo-driver/v2).
//
// Usage: blank-import this package to enable the real driver codec:
//
//	import _ "github.com/go-openapi/strfmt/enable/mongodb"
package mongodb

import (
	"time"

	"github.com/go-openapi/strfmt/internal/bsonlite"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func init() { //nolint:gochecknoinits // blank-import registration pattern, by design
	bsonlite.Replace(driverCodec{})
}

// driverCodec implements bsonlite.Codec using the real MongoDB driver.
type driverCodec struct{}

func (driverCodec) MarshalDoc(value any) ([]byte, error) {
	switch v := value.(type) {
	case [bsonlite.ObjectIDSize]byte:
		return bson.Marshal(bson.M{"data": bson.ObjectID(v)})
	default:
		return bson.Marshal(bson.M{"data": v})
	}
}

func (driverCodec) UnmarshalDoc(data []byte) (any, error) {
	var m bson.M
	if err := bson.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	v := m["data"]

	switch val := v.(type) {
	case bson.DateTime:
		return val.Time(), nil
	case bson.ObjectID:
		return [bsonlite.ObjectIDSize]byte(val), nil
	case time.Time:
		return val, nil
	default:
		return v, nil
	}
}
