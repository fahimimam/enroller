package config

import (
	"github.com/spf13/viper"
	"log"
	"sync"
)

type Enroller struct {
	Org           string
	TLSCertPath   string
	IngressDomain string
	Namespace     string
	RCAMSPPath    string
}

var enrollerOnce = sync.Once{}
var enrollerConfig *Enroller

// loadEnroller loads config from a path
func loadEnroller(fileName string) error {
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	viper.AutomaticEnv()

	enrollerConfig = &Enroller{
		Org:           viper.GetString("enroller.Org"),
		TLSCertPath:   viper.GetString("enroller.TLSCertPath"),
		IngressDomain: viper.GetString("enroller.IngressDomain"),
		Namespace:     viper.GetString("enroller.Namespace"),
		RCAMSPPath:    viper.GetString("enroller.RCAMSPPath"),
	}

	return nil
}

// GetEnroller returns postgres config
func GetEnroller(fileName string) *Enroller {
	enrollerOnce.Do(func() {
		err := loadEnroller(fileName)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
	})

	return enrollerConfig
}
