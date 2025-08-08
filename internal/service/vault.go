package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/triapex/auth/config"
	"github.com/triapex/auth/internal/repo"
	"github.com/triapex/auth/logger"
	"github.com/triapex/auth/utils"
	"os"
	"path/filepath"
)

// VaultService interface
type VaultService interface {
	GetUserMsp(ctx context.Context, username string) (interface{}, error)
	RemoveUserMSP(ctx context.Context, username string) error
	StoreUserMSP(ctx context.Context, userName, mspPath string) error
}

// Vault ...
type Vault struct {
	vaultRepo   repo.VaultRepo
	enrollerCfg *config.Enroller
	vaultConfig *config.Vault
	log         logger.StructLogger
}

// NewVault ...
func NewVault(
	vaultRepo repo.VaultRepo,
	lgr logger.StructLogger,
	enrollerCfg *config.Enroller,
	vaultConfig *config.Vault) VaultService {
	return &Vault{
		log:         lgr,
		vaultRepo:   vaultRepo,
		enrollerCfg: enrollerCfg,
		vaultConfig: vaultConfig,
	}
}

// SetLogger ...
func (v *Vault) SetLogger(l logger.StructLogger) {
	v.log = l
}

func (v *Vault) GetUserMsp(ctx context.Context, username string) (interface{}, error) {
	tid := utils.GetTracingID(ctx)
	v.log.Println("EnrollUser", tid, "Request for Enroll User from service")

	mspData := make(map[string][]byte)

	// Get IssuerRevocationPublicKey
	if content, err := v.vaultRepo.GetVaultContent(username, "IssuerRevocationPublicKey"); err == nil {
		mspData["IssuerRevocationPublicKey"] = content
	}

	// Get signcerts with error propagation
	if err := v.vaultRepo.GetDirectoryContent(username, "signcerts", mspData); err != nil {
		return nil, fmt.Errorf("signcerts error: %v", err)
	}

	// Get keystore with error propagation
	if err := v.vaultRepo.GetDirectoryContent(username, "keystore", mspData); err != nil {
		return nil, fmt.Errorf("keystore error: %v", err)
	}

	// Get cacerts with error propagation
	if err := v.vaultRepo.GetDirectoryContent(username, "cacerts", mspData); err != nil {
		return nil, fmt.Errorf("cacerts error: %v", err)
	}

	if len(mspData) == 0 {
		return nil, errors.New("no MSP data found")
	}
	return mspData, nil
}

func (v *Vault) RemoveUserMSP(ctx context.Context, username string) error {
	path := fmt.Sprintf("%s/metadata/users/%s", v.vaultConfig.KvPath, username)
	err := v.vaultRepo.DeleteContent(path)
	if err != nil {
		return err
	}
	return nil
}

func (v *Vault) StoreUserMSP(ctx context.Context, username, mspPath string) error {
	// Store IssuerRevocationPublicKey if exists
	revocationKeyPath := filepath.Join(mspPath, "IssuerRevocationPublicKey")
	if content, err := os.ReadFile(revocationKeyPath); err == nil {
		vaultPath := fmt.Sprintf("%s/data/users/%s/IssuerRevocationPublicKey", v.vaultConfig.KvPath, username)
		mapContent := map[string]interface{}{
			"data": map[string]interface{}{
				"content": base64.StdEncoding.EncodeToString(content),
			},
		}
		if err := v.vaultRepo.WriteContent(vaultPath, mapContent); err != nil {
			return fmt.Errorf("failed to store revocation key: %v", err)
		}

	}
	// Store signcerts
	signcertsDir := filepath.Join(mspPath, "signcerts")
	if err := v.vaultRepo.StoreDirectory(username, signcertsDir); err != nil {
		return fmt.Errorf("failed to store signcerts: %v", err)
	}

	// Store keystore
	keystoreDir := filepath.Join(mspPath, "keystore")
	if err := v.vaultRepo.StoreDirectory(username, keystoreDir); err != nil {
		return fmt.Errorf("failed to store keystore: %v", err)
	}

	// Store cacerts
	cacertsDir := filepath.Join(mspPath, "cacerts")
	if err := v.vaultRepo.StoreDirectory(username, cacertsDir); err != nil {
		return fmt.Errorf("failed to store cacerts: %v", err)
	}

	return nil
}
