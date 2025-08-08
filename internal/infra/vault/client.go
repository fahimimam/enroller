package vault

import (
	"github.com/hashicorp/vault/api"
	"github.com/triapex/auth/config"
)

type Vault struct {
	client *api.Client
}

func NewVault(cfg *config.Vault) (*Vault, error) {
	vaultDefaultConfig := api.DefaultConfig()
	vaultDefaultConfig.Address = cfg.VaultAddr
	client, err := api.NewClient(vaultDefaultConfig)
	if err != nil {
		return nil, err
	}
	client.SetToken(cfg.VaultToken)

	return &Vault{
		client: client,
	}, nil
}
func (v *Vault) LogicalWrite(path string, data map[string]interface{}) (*api.Secret, error) {
	return v.client.Logical().Write(path, data)
}

func (v *Vault) LogicalDelete(path string) (*api.Secret, error) {
	return v.client.Logical().Delete(path)
}

func (v *Vault) LogicalList(path string) (*api.Secret, error) {
	return v.client.Logical().List(path)
}

func (v *Vault) LogicalRead(path string) (*api.Secret, error) {
	return v.client.Logical().Read(path)
}

//func (v *Vault) LogicalDelete(username string) error {
//	// Delete all versions and metadata
//	path := fmt.Sprintf("%s/metadata/users/%s", v.kvPath, username)
//	_, err := v.client.Logical().Delete(path)
//	if err != nil {
//		return fmt.Errorf("vault deletion failed: %v", err)
//	}
//
//	// Destroy all versions
//	destroyPath := fmt.Sprintf("%s/destroy/users/%s", v.kvPath, username)
//	_, err = v.client.Logical().Write(destroyPath, map[string]interface{}{
//		"versions": []int{1},
//	})
//	if err != nil {
//		return fmt.Errorf("vault deletion failed: %v", err)
//	}
//	return nil
//}
