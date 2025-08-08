package repo

import (
	"encoding/base64"
	"fmt"
	"github.com/triapex/auth/config"
	"github.com/triapex/auth/internal/infra"
	"os"
	"path/filepath"
	"strings"
)

// VaultRepo returns brand repo
type VaultRepo interface {
	StoreDirectory(username, dirPath string) error
	GetDirectoryContent(username, dir string, mspData map[string][]byte) error
	GetVaultContent(username, path string) ([]byte, error)
	WriteContent(path string, data map[string]interface{}) error
	DeleteContent(path string) error
}

type Vault struct {
	vault       infra.Vault
	vaultConfig *config.Vault
}

// NewVault returns new brand repo
func NewVault(vaultConfig *config.Vault, vault infra.Vault) VaultRepo {
	return &Vault{
		vaultConfig: vaultConfig,
		vault:       vault,
	}
}

func (v *Vault) StoreDirectory(username, dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %v", path, err)
		}

		relPath, _ := filepath.Rel(dirPath, path)
		vaultPath := fmt.Sprintf("%s/data/users/%s/%s/%s", v.vaultConfig.KvPath, username, filepath.Base(dirPath), relPath)

		fmt.Println(vaultPath)

		err = v.vault.LogicalWrite(vaultPath, map[string]interface{}{
			"data": map[string]interface{}{
				"content": base64.StdEncoding.EncodeToString(content),
			},
		})

		return err
	})
}

func (v *Vault) GetDirectoryContent(username, dirPath string, mspData map[string][]byte) error {
	// List directory contents using metadata path
	listPath := fmt.Sprintf("%s/metadata/users/%s/%s", v.vaultConfig.KvPath, username, dirPath)
	fmt.Println(listPath)
	secret, err := v.vault.LogicalList(listPath)
	if err != nil {
		return fmt.Errorf("failed to list directory: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return fmt.Errorf("directory not found")
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid directory structure")
	}

	for _, key := range keys {
		fileName := key.(string)
		// Skip directory markers
		if strings.HasSuffix(fileName, "/") {
			continue
		}

		content, err := v.GetVaultContent(username, fmt.Sprintf("%s/%s", dirPath, fileName))
		if err != nil {
			return fmt.Errorf("failed to get %s: %w", fileName, err)
		}
		mspData[fmt.Sprintf("%s/%s", dirPath, fileName)] = content
	}
	return nil
}

func (v *Vault) GetVaultContent(username, path string) ([]byte, error) {
	vaultPath := fmt.Sprintf("%s/data/users/%s/%s", v.vaultConfig.KvPath, username, path)
	secret, err := v.vault.LogicalRead(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("vault read failed: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found")
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format")
	}

	content, ok := data["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content missing")
	}

	return []byte(content), nil
}

func (v *Vault) WriteContent(path string, data map[string]interface{}) error {
	if err := v.vault.LogicalWrite(path, data); err != nil {
		return fmt.Errorf("failed to store revocation key: %v", err)
	}
	return nil
}

func (v *Vault) DeleteContent(path string) error {
	if err := v.vault.LogicalDelete(path); err != nil {
		return fmt.Errorf("failed to store revocation key: %v", err)
	}
	return nil
}
