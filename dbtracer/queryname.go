package dbtracer

import (
	"regexp"
)

var queryNameRegex = regexp.MustCompile(`^(?:--|/\*)\s+name:\s+(?P<name>\w+) :(?P<command>\w+)`)

type queryMetadata struct {
	name    string
	command string
}

func queryMetadataFromSQL(sql string) *queryMetadata {
	match := queryNameRegex.FindStringSubmatch(sql)
	if match == nil {
		return nil
	}

	return &queryMetadata{name: match[1], command: match[2]}
}
