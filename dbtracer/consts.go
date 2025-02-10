package dbtracer

import "go.opentelemetry.io/otel/attribute"

const (
	SQLCQueryNameKey      = attribute.Key("sqlc.query.name")
	SQLCQueryTypeKey      = attribute.Key("sqlc.query.type")
	PGXOperationTypeKey   = attribute.Key("pgx.operation.type")
	PGXPrepareStmtNameKey = attribute.Key("pgx.prepare_stmt.name")
	PGXStatusKey          = attribute.Key("pgx.status")

	DBStatusCodeKey = attribute.Key("db.response.status_code")
)
