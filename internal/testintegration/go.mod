module github.com/go-openapi/strfmt/internal/testintegration

go 1.25.0

require (
	github.com/go-openapi/strfmt v0.26.1
	github.com/go-openapi/strfmt/enable/mongodb v0.25.0
	github.com/go-openapi/testify/v2 v2.4.2
	github.com/go-sql-driver/mysql v1.9.3
	github.com/jackc/pgx/v5 v5.9.1
	go.mongodb.org/mongo-driver/v2 v2.5.0
)

require (
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/go-openapi/errors v0.22.7 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace (
	github.com/go-openapi/strfmt => ../..
	github.com/go-openapi/strfmt/enable/mongodb => ../../enable/mongodb
)
