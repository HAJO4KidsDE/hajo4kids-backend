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

	// Enable verbose logging to debug migration issues
	gormConfig.Logger = logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			LogLevel: logger.Info,
		},
	)

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
	log.Println("Running database migrations...")
	if err := DB.AutoMigrate(models...); err != nil {
		log.Printf("Migration failed: %v", err)
		return err
	}
	log.Println("Database migrations completed successfully")
	
	// Debug: List created tables
	var tables []string
	if DB.Raw("SHOW TABLES").Scan(&tables); len(tables) > 0 {
		log.Printf("Created %d tables: %v", len(tables), tables)
	} else {
		log.Println("WARNING: No tables found after migration!")
	}
	
	return nil
}