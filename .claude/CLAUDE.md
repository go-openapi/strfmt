# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go library that gives first-class Go types to the `format` keyword from
[JSON Schema](https://json-schema.org/understanding-json-schema/reference/string#format) and
[OpenAPI](https://spec.openapis.org/oas/v3.1.1.html#data-types).

In those specs, `format` is a string annotation that hints at richer semantics —
`"date-time"`, `"uuid"`, `"email"`, `"uri"`, `"duration"`, etc. This package turns each
format into a concrete Go type (`DateTime`, `UUID`, `Email`, …) that knows how to validate,
parse, and round-trip through JSON, SQL (`driver.Valuer`/`sql.Scanner`), text, and BSON
encodings.

A pluggable `Registry` lets consumers (code generators, validators, runtime
libraries) discover and work with formats generically.

The package is a foundational building block of the
[go-swagger](https://github.com/go-swagger/go-swagger) ecosystem, but is useful anywhere
OpenAPI/JSON Schema formats need to travel across serialization boundaries.

See [docs/MAINTAINERS.md](../docs/MAINTAINERS.md) for CI/CD, release process, and repo structure details.

### Package layout

This is a mono-repo with multiple Go modules tied together by `go.work`.

#### Root module (`github.com/go-openapi/strfmt`)

| File | Contents |
|------|----------|
| `doc.go` | Package-level godoc |
| `ifaces.go` | Core interfaces: `Format` (string + text marshaling) and `Registry` (format registration, validation, parsing) |
| `format.go` | `Default` registry, `NewFormats()`, `NewSeededFormats()`, `NameNormalizer`, registry implementation |
| `default.go` | Simple string-wrapper types: `Base64`, `URI`, `Email`, `Hostname`, `IPv4`, `IPv6`, `CIDR`, `MAC`, `UUID`/`UUID3`/`UUID4`/`UUID5`/`UUID7`, `ISBN`/`ISBN10`/`ISBN13`, `CreditCard`, `SSN`, `HexColor`, `RGBColor`, `Password`; validators (`IsEmail`, `IsURI`, …) |
| `date.go` | `Date` type (wraps `time.Time`, RFC3339 full-date format) |
| `time.go` | `DateTime` type (wraps `time.Time`, RFC3339 date-time with flexible parsing) |
| `duration.go` | `Duration` type (wraps `time.Duration`, ISO 8601 duration parsing) |
| `ulid.go` | `ULID` type (wraps `oklog/ulid`) |
| `bson.go` | `ObjectId` type (`[12]byte`), hex encoding, no mongo-driver dependency |
| `mongo.go` | `MarshalBSON`/`UnmarshalBSON` implementations for all types, uses `internal/bsonlite` codec |
| `errors.go` | `ErrFormat` sentinel error |
| `conv/` | Pointer-helper sub-package: `conv.Date(v)` → `*Date`, `conv.DateValue(p)` → `Date`, etc. for every type |

#### Internal packages (`internal/`)

| Package | Contents |
|---------|----------|
| `internal/bsonlite` | Minimal BSON codec (wire-compatible with mongo-driver v2.5.0), handles single-key documents with string/DateTime/ObjectID values |

#### Sub-modules

| Module | Contents |
|--------|----------|
| `enable/mongodb` | Blank-import package that swaps the built-in `bsonlite` codec with the real MongoDB driver codec |
| `internal/testintegration` | Integration tests against real databases (MongoDB, MariaDB, PostgreSQL) |

### Key API

- `Format` interface — `String()` + `encoding.TextMarshaler` / `encoding.TextUnmarshaler`
- `Registry` interface — `Add()`, `DelByName()`, `GetType()`, `ContainsName()`, `Validates()`, `Parse()`, `MapStructureHookFunc()`
- `Default` — the global `Registry` instance, pre-seeded with all built-in formats
- `NewFormats()` — creates a new registry seeded from `Default`
- `NewSeededFormats(seeds, normalizer)` — creates a registry with custom seeds and name normalizer
- Format types — `Date`, `DateTime`, `Duration`, `ULID`, `ObjectId`, `UUID`, `Email`, `URI`, `Hostname`, `Base64`, etc.
- Validators — `IsDate()`, `IsDateTime()`, `IsDuration()`, `IsUUID()`, `IsEmail()`, `IsHostname()`, etc.

### Dependencies

- `github.com/go-openapi/errors` — error types for validation failures
- `github.com/go-viper/mapstructure/v2` — decode hook for format-aware struct mapping
- `github.com/google/uuid` — UUID parsing and validation
- `github.com/oklog/ulid/v2` — ULID implementation
- `golang.org/x/net` — IDNA hostname validation
- `github.com/go-openapi/testify/v2` — test-only assertions (zero-dep testify fork)

### Notable design decisions

- **Self-registering types via `init()`** — each format type file has an `init()` function that
  registers itself in the `Default` registry. This means importing the package automatically
  makes all formats available.
- **BSON without mongo-driver** — the `internal/bsonlite` package provides a minimal BSON codec
  so the root module has no dependency on the MongoDB driver. Users who need the full driver
  import `enable/mongodb` as a blank import to swap in the real codec.
- **`ObjectId` is `[12]byte`, not a string** — unlike many Go BSON libraries, `ObjectId` is a
  fixed-size byte array for zero-allocation hex encoding and efficient comparison.
- **`DateTime` accepts flexible input** — the parser tries multiple RFC3339 variations (with/without
  fractional seconds, Z vs offset) to handle real-world JSON from various sources.
- **`Duration` parses ISO 8601** — supports both Go-style (`1h30m`) and ISO 8601 (`P1DT12H`)
  duration strings.

