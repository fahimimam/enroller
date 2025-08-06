package infra

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/hashicorp/vault/api"
	"github.com/triapex/auth/config"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Vault interface {
	StoreUserMSP(username, mspPath string) error
	StoreDirectory(username, dirPath string) error
	GetUserMSP(username string) (map[string][]byte, error)
	GetDirectoryContent(username, dir string, mspData map[string][]byte) error
	GetVaultContent(username, path string) ([]byte, error)
	LogicalWrite(path string, data []byte) error
	LogicalDelete(path string) error
}

type VaultStore struct {
	client *api.Client
	kvPath string
}

func NewVaultStore(cfg *config.Vault) (*VaultStore, error) {
	vaultDefaultConfig := api.DefaultConfig()
	vaultDefaultConfig.Address = cfg.VaultAddr
	client, err := api.NewClient(vaultDefaultConfig)
	if err != nil {
		return nil, err
	}
	client.SetToken(cfg.VaultToken)

	return &VaultStore{
		client: client,
		kvPath: cfg.KvPath,
	}, nil
}

func (v *VaultStore) StoreUserMSP(username, mspPath string) error {
	// Store IssuerRevocationPublicKey if exists
	revocationKeyPath := filepath.Join(mspPath, "IssuerRevocationPublicKey")
	if content, err := ioutil.ReadFile(revocationKeyPath); err == nil {
		vaultPath := fmt.Sprintf("%s/data/users/%s/IssuerRevocationPublicKey", v.kvPath, username)
		if _, err := v.client.Logical().Write(vaultPath, map[string]interface{}{
			"data": map[string]interface{}{
				"content": base64.StdEncoding.EncodeToString(content),
			},
		}); err != nil {
			return fmt.Errorf("failed to store revocation key: %v", err)
		}
	}
	// Store sign certs
	signcertsDir := filepath.Join(mspPath, "signcerts")
	if err := v.StoreDirectory(username, signcertsDir); err != nil {
		return fmt.Errorf("failed to store signcerts: %v", err)
	}

	// Store keystore
	keystoreDir := filepath.Join(mspPath, "keystore")
	if err := v.StoreDirectory(username, keystoreDir); err != nil {
		return fmt.Errorf("failed to store keystore: %v", err)
	}

	// Store cacerts
	cacertsDir := filepath.Join(mspPath, "cacerts")
	if err := v.StoreDirectory(username, cacertsDir); err != nil {
		return fmt.Errorf("failed to store cacerts: %v", err)
	}

	return nil
}

func (v *VaultStore) StoreDirectory(username, dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %v", path, err)
		}

		relPath, _ := filepath.Rel(dirPath, path)
		vaultPath := fmt.Sprintf("%s/data/users/%s/%s/%s", v.kvPath, username, filepath.Base(dirPath), relPath)

		fmt.Println(vaultPath)

		_, err = v.client.Logical().Write(vaultPath, map[string]interface{}{
			"data": map[string]interface{}{
				"content": base64.StdEncoding.EncodeToString(content),
			},
		})

		return err
	})
}

func (v *VaultStore) GetUserMSP(username string) (map[string][]byte, error) {
	mspData := make(map[string][]byte)

	// Get IssuerRevocationPublicKey
	if content, err := v.GetVaultContent(username, "IssuerRevocationPublicKey"); err == nil {
		mspData["IssuerRevocationPublicKey"] = content
	}

	// Get signcerts with error propagation
	if err := v.GetDirectoryContent(username, "signcerts", mspData); err != nil {
		return nil, fmt.Errorf("signcerts error: %v", err)
	}

	// Get keystore with error propagation
	if err := v.GetDirectoryContent(username, "keystore", mspData); err != nil {
		return nil, fmt.Errorf("keystore error: %v", err)
	}

	// Get cacerts with error propagation
	if err := v.GetDirectoryContent(username, "cacerts", mspData); err != nil {
		return nil, fmt.Errorf("cacerts error: %v", err)
	}

	if len(mspData) == 0 {
		return nil, errors.New("no MSP data found")
	}

	return mspData, nil
}

func (v *VaultStore) GetDirectoryContent(username, dirPath string, mspData map[string][]byte) error {
	// List directory contents using metadata path
	listPath := fmt.Sprintf("%s/metadata/users/%s/%s", v.kvPath, username, dirPath)
	fmt.Println(listPath)
	secret, err := v.client.Logical().List(listPath)
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

func (v *VaultStore) GetVaultContent(username, path string) ([]byte, error) {
	vaultPath := fmt.Sprintf("%s/data/users/%s/%s", v.kvPath, username, path)
	secret, err := v.client.Logical().Read(vaultPath)
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

func (v *VaultStore) LogicalWrite(path string, data []byte) error {
	if _, err := v.client.Logical().Write(path, map[string]interface{}{
		"data": map[string]interface{}{
			"content": base64.StdEncoding.EncodeToString(data),
		},
	}); err != nil {
		return fmt.Errorf("failed to write to vault: %v", err)
	}
	return nil
}

func (v *VaultStore) LogicalDelete(username string) error {
	// Delete all versions and metadata
	path := fmt.Sprintf("%s/metadata/users/%s", v.kvPath, username)
	_, err := v.client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("vault deletion failed: %v", err)
	}

	// Destroy all versions
	destroyPath := fmt.Sprintf("%s/destroy/users/%s", v.kvPath, username)
	_, err = v.client.Logical().Write(destroyPath, map[string]interface{}{
		"versions": []int{1},
	})
}
