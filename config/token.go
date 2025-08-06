package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// Token holds table configurations
type Token struct {
	AccessTokenDuration  string `yaml:"access_token_duration"`
	RefreshTokenDuration string `yaml:"refresh_token_duration"`
	PrivateKeyPath       string `yaml:"private_key_path"`
	PublicKeyPath        string `yaml:"public_key_path"`
}

var tokenOnce = sync.Once{}
var tokenConfig *Token

// loadToken loads config from path
func loadToken(fileName string) error {
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	viper.AutomaticEnv()

	tokenConfig = &Token{
		AccessTokenDuration:  viper.GetString("token.access_token_duration"),
		RefreshTokenDuration: viper.GetString("token.RefreshTokenDuration"),
		PrivateKeyPath:       viper.GetString("token.private_key_path"),
		PublicKeyPath:        viper.GetString("token.public_key_path"),
	}

	log.Println("table config ", tokenConfig)

	return nil
}

// GetToken returns token config
func GetToken(fileName string) *Token {
	tokenOnce.Do(func() {
		err := loadToken(fileName)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
	})

	return tokenConfig
}
