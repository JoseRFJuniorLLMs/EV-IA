package postgres

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewConnection initializes a new PostgreSQL connection using GORM
func NewConnection(url string, log *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Adjust log level as needed
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set connection pool settings
	// These could be configurable
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Info("Successfully connected to PostgreSQL")
	return db, nil
}

// RunMigrations - migrations are managed via SQL files in migrations/
// AutoMigrate is disabled to prevent conflicts with existing schema
func RunMigrations(db *gorm.DB) error {
	// Tables already exist from SQL migrations (001_initial_schema.sql, 002_v2g_tables.sql)
	// Skip GORM AutoMigrate to avoid constraint conflicts
	return nil
}

// Helper to close connection if needed (though *gorm.DB doesn't have Close directly, sql.DB does)
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
