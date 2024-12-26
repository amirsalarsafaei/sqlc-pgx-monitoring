package dbtracer

import (
	"regexp"
)

// sqlc queries are declared after a sql command in the form of -- name: TheQueryName :type
var queryNameRegex = regexp.MustCompile(`^(?:--|/\*)\s+name:\s+(?P<name>\w+) :(?P<type>\w+)`)

func queryNameFromSQL(sql string) (string, string) {
	if !queryNameRegex.MatchString(sql) {
		return "unknown", "unknown"
	}
	return queryNameRegex.FindStringSubmatch(sql)[1], queryNameRegex.FindStringSubmatch(sql)[2]
}
