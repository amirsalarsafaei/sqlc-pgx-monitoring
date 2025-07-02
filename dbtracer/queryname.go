package dbtracer

import (
	"regexp"
)

// sqlc queries are declared after a sql command in the form of -- name: TheQueryName :type
var queryNameRegex = regexp.MustCompile(`^(?:--|/\*)\s+name:\s+(?P<name>\w+) :(?P<command>\w+)`)

type queryMetadata struct{
	name string
	command string
}

func queryMetadataFromSQL(sql string) (*queryMetadata) {
	if !queryNameRegex.MatchString(sql) {
		return nil
	}

	return &queryMetadata{name: queryNameRegex.FindStringSubmatch(sql)[1], command: queryNameRegex.FindStringSubmatch(sql)[2]}
}
