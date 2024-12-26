package dbtracer

import (
	"testing"
)

func TestQueryNameFromSQL(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		wantQueryName string
		wantQueryType string
	}{
		{
			name:          "valid sqlc query with double dash",
			sql:           "-- name: GetUser :one",
			wantQueryName: "GetUser",
			wantQueryType: "one",
		},
		{
			name:          "valid sqlc query with block comment",
			sql:           "/* name: CreateUser :exec */",
			wantQueryName: "CreateUser",
			wantQueryType: "exec",
		},
		{
			name:          "invalid sql - no name declaration",
			sql:           "SELECT * FROM users",
			wantQueryName: "unknown",
			wantQueryType: "unknown",
		},
		{
			name:          "invalid sql - malformed name declaration",
			sql:           "-- name: Invalid Query Format",
			wantQueryName: "unknown",
			wantQueryType: "unknown",
		},
		{
			name:          "empty sql",
			sql:           "",
			wantQueryName: "unknown",
			wantQueryType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotType := queryNameFromSQL(tt.sql)
			if gotName != tt.wantQueryName {
				t.Errorf("queryNameFromSQL() gotName = %v, want %v", gotName, tt.wantQueryName)
			}
			if gotType != tt.wantQueryType {
				t.Errorf("queryNameFromSQL() gotType = %v, want %v", gotType, tt.wantQueryType)
			}
		})
	}
}
