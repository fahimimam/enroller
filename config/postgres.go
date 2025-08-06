package config

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Postgres holds postgres config
type Postgres struct {
	DBHost     string        `yaml:"db_host"`
	DBPort     string        `yaml:"db_port"`
	DBName     string        `yaml:"db_name"`
	DBUser     string        `yaml:"db_user"`
	DBPassword string        `yaml:"db_password"`
	DBSSLMode  string        `yaml:"db_ssl_mode"`
	DBTimeZone string        `yaml:"db_time_zone"`
	SchemaName string        `yaml:"schema_name"`
	DBTimeOut  time.Duration `yaml:"time_out"`
	Level      string        `yaml:"level"`
}

var postgresOnce = sync.Once{}
var postgresConfig *Postgres

// loadPostgres loads config from a path
func loadPostgres(fileName string) error {
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	viper.AutomaticEnv()

	postgresConfig = &Postgres{
		DBHost:     viper.GetString("postgres.host"),
		DBPort:     viper.GetString("postgres.port"),
		DBName:     viper.GetString("postgres.name"),
		DBUser:     viper.GetString("postgres.user"),
		DBPassword: viper.GetString("postgres.password"),
		DBSSLMode:  viper.GetString("postgres.ssl_mode"),
		DBTimeZone: viper.GetString("postgres.time_zone"),
		SchemaName: viper.GetString("postgres.schema_name"),
		DBTimeOut:  viper.GetDuration("postgres.time_out") * time.Second,
		Level:      viper.GetString("postgres.level"),
	}

	return nil
}

// GetPostgres returns postgres config
func GetPostgres(fileName string) *Postgres {
	postgresOnce.Do(func() {
		err := loadPostgres(fileName)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
	})

	return postgresConfig
}
