// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0

package exampletable

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type ExampleTable struct {
	ID  pgtype.Int8 `db:"id" json:"id"`
	Foo pgtype.Text `db:"foo" json:"foo"`
}