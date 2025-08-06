package config

import (
	"github.com/spf13/viper"
	"log"
	"sync"
)

type Vault struct {
	KvPath     string
	VaultAddr  string
	VaultToken string
}

var vaultOnce = sync.Once{}
var vaultConfig *Vault

// loadEnroller loads config from a path
func loadVault(fileName string) error {
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	viper.AutomaticEnv()

	vaultConfig = &Vault{
		KvPath:     viper.GetString("vault.kv_path"),
		VaultAddr:  viper.GetString("vault.vault_addr"),
		VaultToken: viper.GetString("vault.vault_token"),
	}

	return nil
}

// GetVault returns postgres config
func GetVault(fileName string) *Vault {
	vaultOnce.Do(func() {
		err := loadEnroller(fileName)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
	})

	return vaultConfig
}
