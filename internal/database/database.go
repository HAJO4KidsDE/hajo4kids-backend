package database

import (
	"fmt"
	"log"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {
	var err error
	var dsn string

	gormConfig := &gorm.Config{}

	// Enable logging in development
	if cfg.Server.Host == "0.0.0.0" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	switch cfg.Database.Driver {
	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Name,
			cfg.Database.SSLMode,
		)
		DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	case "mysql", "mariadb":
		// MySQL/MariaDB DSN format: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Name,
		)
		DB, err = gorm.Open(mysql.Open(dsn), gormConfig)
	case "sqlite":
		dsn = cfg.Database.Name
		DB, err = gorm.Open(sqlite.Open(dsn), gormConfig)
	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")
	return nil
}

func Migrate(models ...interface{}) error {
	return DB.AutoMigrate(models...)
}