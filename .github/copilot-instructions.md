# Copilot Instructions

## Project Overview

Go library that gives first-class Go types to the `format` keyword from
[JSON Schema](https://json-schema.org/understanding-json-schema/reference/string#format) and
[OpenAPI](https://spec.openapis.org/oas/v3.1.1.html#data-types).

In those specs, `format` is a string annotation that hints at richer semantics —
`"date-time"`, `"uuid"`, `"email"`, `"uri"`, `"duration"`, etc. This package turns each
format into a concrete Go type (`DateTime`, `UUID`, `Email`, …) that knows how to validate,
parse, and round-trip through JSON, SQL (`driver.Valuer`/`sql.Scanner`), text, and BSON
encodings. A pluggable `Registry` lets consumers (code generators, validators, runtime
libraries) discover and work with formats generically.

The package is a foundational building block of the
[go-swagger](https://github.com/go-swagger/go-swagger) ecosystem, but is useful anywhere
OpenAPI/JSON Schema formats need to travel across serialization boundaries.

This is a mono-repo (`go.work`) with three modules: root, `enable/mongodb`, `internal/testintegration`.

## Package Layout

| File | Contents |
|------|----------|
| `ifaces.go` | Core interfaces: `Format` (string + text marshaling) and `Registry` (format registration, validation, parsing) |
| `format.go` | `Default` registry, `NewFormats()`, `NewSeededFormats()`, `NameNormalizer` |
| `default.go` | Simple string-wrapper types: `URI`, `Email`, `Hostname`, `IPv4`, `IPv6`, `CIDR`, `MAC`, `UUID`/`UUID3-7`, `ISBN`, `CreditCard`, `SSN`, `HexColor`, `RGBColor`, `Password`, `Base64`; validators |
| `date.go` | `Date` type (wraps `time.Time`, RFC3339 full-date) |
| `time.go` | `DateTime` type (wraps `time.Time`, RFC3339 date-time with flexible parsing) |
| `duration.go` | `Duration` type (wraps `time.Duration`, ISO 8601 duration parsing) |
| `ulid.go` | `ULID` type (wraps `oklog/ulid`) |
| `bson.go` | `ObjectId` type (`[12]byte`), hex encoding, no mongo-driver dependency |
| `mongo.go` | `MarshalBSON`/`UnmarshalBSON` for all types via `internal/bsonlite` codec |
| `errors.go` | `ErrFormat` sentinel error |
| `conv/` | Pointer helpers: `conv.Date(v)` → `*Date`, `conv.DateValue(p)` → `Date`, etc. |

## Key API

- `Format` interface — `String()` + `encoding.TextMarshaler` / `encoding.TextUnmarshaler`
- `Registry` interface — `Add()`, `DelByName()`, `GetType()`, `ContainsName()`, `Validates()`, `Parse()`
- `Default` — global `Registry`, pre-seeded with all built-in formats
- Format types — `Date`, `DateTime`, `Duration`, `ULID`, `ObjectId`, `UUID`, `Email`, `URI`, `Hostname`, `Base64`, …
- Validators — `IsDate()`, `IsDateTime()`, `IsDuration()`, `IsUUID()`, `IsEmail()`, …

## Design Decisions

- Types self-register via `init()` — importing the package makes all formats available.
- BSON without mongo-driver — `internal/bsonlite` provides a minimal codec; `enable/mongodb` swaps in the real driver.
- `ObjectId` is `[12]byte`, not a string — zero-allocation hex encoding, efficient comparison.
- `DateTime` accepts flexible RFC3339 input (with/without fractional seconds, Z vs offset).
- `Duration` parses both Go-style (`1h30m`) and ISO 8601 (`P1DT12H`) strings.
- Formatted types carry a `swagger:strfmt [format]` annotation consumed by go-swagger (see rules).

## Dependencies

- `github.com/go-openapi/errors` — validation error types
- `github.com/go-viper/mapstructure/v2` — decode hook for format-aware struct mapping
- `github.com/google/uuid` — UUID parsing and validation
- `github.com/oklog/ulid/v2` — ULID implementation
- `golang.org/x/net` — IDNA hostname validation
- `github.com/go-openapi/testify/v2` — test-only assertions (zero-dep fork of `stretchr/testify`)

## Conventions

- All `.go` files must have SPDX license headers (Apache-2.0).
- Commits require DCO sign-off (`git commit -s`).
- Linting: `golangci-lint run` — config in `.golangci.yml` (posture: `default: all` with explicit disables).
- Every `//nolint` directive **must** have an inline comment explaining why.
- Tests: `go test work ./...` (mono-repo). CI runs on `{ubuntu, macos, windows} x {stable, oldstable}` with `-race`.
- Test framework: `github.com/go-openapi/testify/v2` (not `stretchr/testify`; `testifylint` does not work).

See `.github/copilot/` (symlinked to `.claude/rules/`) for detailed rules on Go conventions, linting, testing, and swagger comments.
