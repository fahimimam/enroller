package utils

import (
	"strings"
)

var ValidRoles = map[string]bool{
	admin:          true,
	invoiceCreator: true,
	invoiceWriter:  true,
	invoiceReader:  true,
}

func IsRolesValid(roles []string) bool {
	for _, role := range roles {
		if !ValidRoles[strings.ToLower(strings.TrimSpace(role))] {
			return false
		}
	}
	return true
}
