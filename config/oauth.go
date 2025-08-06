package config

import (
	"context"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"log"
	"sync"

	"github.com/spf13/viper"
)

// OAuth2 holds table configurations
type OAuth2 struct {
	Oauth2        oauth2.Config
	OidcProvider  *oidc.Provider
	TokenVerifier *oidc.IDTokenVerifier
}

var oauthOnce = sync.Once{}
var oauthConfig *OAuth2

// loadToken loads config from path
func loadOAuth(fileName string) error {
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	viper.AutomaticEnv()

	// Initialize OIDC provider
	ctx := context.Background()
	oidcProvider, err := oidc.NewProvider(ctx, viper.GetString("oauth.google_issuer"))
	if err != nil {
		log.Fatal("Failed to initialize OIDC provider:", err)
	}

	tokenVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: viper.GetString("oauth.client_id")})

	// Configure OAuth
	oauth2Config := oauth2.Config{
		ClientID:     viper.GetString("oauth.client_id"),
		ClientSecret: viper.GetString("oauth.client_secret"),
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  viper.GetString("oauth.redirect_url"),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	oauthConfig = &OAuth2{
		Oauth2:        oauth2Config,
		OidcProvider:  oidcProvider,
		TokenVerifier: tokenVerifier,
	}

	log.Println("table config ", tokenConfig)

	return nil
}

// GetOAuth returns token config
func GetOAuth(fileName string) *OAuth2 {
	oauthOnce.Do(func() {
		err := loadOAuth(fileName)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
	})

	return oauthConfig
}
