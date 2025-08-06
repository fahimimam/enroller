package config

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Application holds application configurations
type Application struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	GracefulTimeout time.Duration `yaml:"graceful_timeout"`
	Env             string        `yaml:"env"`
}

var appOnce = sync.Once{}
var appConfig *Application

// loadApp loads config from path
func loadApp(fileName string) error {
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	viper.AutomaticEnv()

	appConfig = &Application{
		Host:            viper.GetString("app.host"),
		GracefulTimeout: viper.GetDuration("app.graceful_timeout"),
		Port:            viper.GetInt("app.port"),
		Env:             viper.GetString("app.env"),
	}

	log.Println("app config ", appConfig)
	return nil
}

// GetApp returns application config
func GetApp(fileName string) *Application {
	appOnce.Do(func() {
		err := loadApp(fileName)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
	})

	return appConfig
}
