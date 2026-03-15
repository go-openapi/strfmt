---
paths:
  - "**/*.go"
---

# Comments for strfmt types

Formatted types (all types exported by this library, e.g. `DateTime`, `UUID`, ...) come
with a special annotation `swagger:strfmt [format]`.

These comments are significant and consumed by the `go-swagger`
to generate OpenAPI specifications from go source.

The comment must reside exactly on a single line, preferably as the last block of comment lines for this type.

The `swagger:strfmt` string must remain as-is.

The `[format]` string must be the name of the registered format (e.g. `date`, `date-time`, `uuid` etc.).

The comment line _may_ be terminated with a `.` to mark the end of the comment sentence (common comment linting rule).

```go
// [MyFormattedType] ...
//
// swagger:strfmt [format].
type [MyFormattedType] ...
```
