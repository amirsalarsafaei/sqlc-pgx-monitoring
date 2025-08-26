package dbtracer

import "go.opentelemetry.io/otel/attribute"

const (
	SQLCQueryNameKey      = attribute.Key("sqlc.query.name")
	SQLCQueryCommandKey   = attribute.Key("sqlc.query.command")
	PGXOperationTypeKey   = attribute.Key("pgx.operation.type")
	PGXPrepareStmtNameKey = attribute.Key("pgx.prepare_stmt.name")
	PGXStatusKey          = attribute.Key("pgx.status")

	DBStatusCodeKey = attribute.Key("db.response.status_code")

	PGXPoolConnOperationKey = attribute.Key("pgx.pool.operation")
)

var defaultBucketBoundaries = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}
