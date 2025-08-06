package config

import (
	"github.com/spf13/viper"
	"log"
	"sync"
)

// Table holds table configurations
type Table struct {
	UserTable string `yaml:"user"`

	UserCollectionNameProfile string `yaml:"profile"`
	VerificationCollection    string `yaml:"verification"`
	OtpCollection             string `yaml:"otp"`
}

var tableOnce = sync.Once{}
var tableConfig *Table

// loadTable loads config from path
func loadTable(fileName string) error {
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	viper.AutomaticEnv()

	tableConfig = &Table{
		UserTable:                 viper.GetString("table.user"),
		UserCollectionNameProfile: viper.GetString("table.profile"),
		VerificationCollection:    viper.GetString("table.verification"),
		OtpCollection:             viper.GetString("table.otp"),
	}

	log.Println("table config ", tableConfig)

	return nil
}

// GetTable returns table config
func GetTable(fileName string) *Table {
	tableOnce.Do(func() {
		err := loadTable(fileName)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
	})

	return tableConfig
}
