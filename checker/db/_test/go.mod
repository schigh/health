module github.com/schigh/health/checker/db/_test

go 1.22.0

replace github.com/schigh/health => ../../../

require (
	github.com/go-sql-driver/mysql v1.7.1
	github.com/jmoiron/sqlx v1.3.5
	github.com/schigh/health v0.0.0-00010101000000-000000000000
)

require google.golang.org/protobuf v1.32.0 // indirect
