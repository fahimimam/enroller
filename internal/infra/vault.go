package infra

import "github.com/hashicorp/vault/api"

type Vault interface {
	LogicalWrite(path string, data map[string]interface{}) error
	LogicalDelete(path string) error
	LogicalList(path string) (*api.Secret, error)
	LogicalRead(path string) (*api.Secret, error)
}
