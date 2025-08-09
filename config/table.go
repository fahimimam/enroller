package config

import (
	"github.com/spf13/viper"
	"log"
	"sync"
)

// Table holds table configurations
type Table struct {
	UserTable string `yaml:"user"`
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
		UserTable: viper.GetString("table.user"),
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
