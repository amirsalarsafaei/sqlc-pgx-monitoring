package dbtracer

import "regexp"

// sqlc queries are declared after a sql command in the form of -- name: TheQueryName :type
var queryNameRegex = regexp.MustCompile(`^(?:--|/\*)\s+name:\s+(?P<name>\w+)`)

func queryNameFromSQL(sql string) string {
	if !queryNameRegex.MatchString(sql) {
		return sql
	}
	return queryNameRegex.FindStringSubmatch(sql)[1]
}
