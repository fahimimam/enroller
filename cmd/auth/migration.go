package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/triapex/auth/config"
	"github.com/triapex/auth/internal/infra/postgres"
	"github.com/triapex/auth/internal/infra/redis"
	"github.com/triapex/auth/internal/repo"
	"github.com/triapex/auth/internal/service"
	"github.com/triapex/auth/logger"
	"github.com/triapex/auth/model"
	"log"
	"strconv"
	"time"
)

var userSVC service.UserService
var db *postgres.Postgres
var migrationConfig *config.MigrationDetails

var migrationRoot = &cobra.Command{
	Use:   "migration",
	Short: "Run database migrations",
	Long:  `Migration is a tool to generate and modify databse tables`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfgPostgres := config.GetPostgres(cfgPath)
		cfgDBTable := config.GetTable(cfgPath)
		cfgRedis := config.GetRedis(cfgPath)
		cfgToken := config.GetToken(cfgPath)
		cfgOAuth := config.GetOAuth(cfgPath)
		migrationConfig = config.GetMigration(cfgPath)
		ctx := context.Background()
		lgr := logger.DefaultOutStructLogger
		var err error
		// connect postgres db
		db, err = postgres.NewConnection(ctx, cfgPostgres)
		if err != nil {
			return err
		}
		//defer db.Close(ctx)
		// connect redis db

		kv, err := redis.New(cfgRedis.URL, cfgRedis.RedisTimeOut, "auth")
		if err != nil {
			return err
		}
		defer kv.Close()

		userRepo := repo.NewUser(cfgDBTable, db)
		privateKey, err := config.GetPrivateKey(cfgToken.PrivateKeyPath)
		if err != nil {
			return err
		}
		publicKey, err := config.GetPublicKey(cfgToken.PublicKeyPath)
		if err != nil {
			return err
		}

		accessTokenDuration, err := strconv.ParseInt(cfgToken.AccessTokenDuration, 10, 64)
		if err != nil || accessTokenDuration == 0 {
			accessTokenDuration = DefaultAccessTokenDuration
		}

		refreshTokenDuration, err := strconv.ParseInt(cfgToken.RefreshTokenDuration, 10, 64)
		if err != nil || refreshTokenDuration == 0 {
			refreshTokenDuration = DefaultRefreshTokenDuration
		}
		userSVC = service.NewUser(userRepo, privateKey, publicKey, time.Duration(accessTokenDuration), time.Duration(refreshTokenDuration), lgr, cfgOAuth)
		return nil
	},
}

func init() {
	migrationRoot.PersistentFlags().StringVarP(&cfgPath, "config", "c", "config.yaml", "config file path")
}

var migrationUp = &cobra.Command{
	Use:   "up",
	Short: "Populate tables in database",
	Long:  `Populate tables in database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("Populating database...")
		//ctx := context.Background()
		defer db.Close(context.Background())
		if err := db.DB.AutoMigrate(model.Models...); err != nil {
			log.Println("Failed to migrate database. Error: ", err.Error())
			return err
		}

		log.Println("Database populated successfully!")
		return nil
	},
}

var migrationDown = &cobra.Command{
	Use:   "down",
	Short: "Drop tables from database",
	Long:  `Drop tables from database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		log.Println("Dropping database table...")
		i := 1
		for _, table := range model.Models {
			fmt.Printf("Iteration: %v\n", i)
			i++
			err = db.Migrator().DropTable(table)
		}
		log.Println("Database dopped successfully!")
		return err
	},
}

func init() {
	migrationRoot.AddCommand(
		migrationUp,
		migrationDown,
	)
}
