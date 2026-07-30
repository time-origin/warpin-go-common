package database

import (
	"fmt"

	"gorm.io/gorm"
)

// GormConnector adapts a PostgreSQL Config to retry/health-check containers.
type GormConnector struct {
	Config *Config
}

func (c *GormConnector) Name() string {
	return "PostgreSQL"
}

func (c *GormConnector) Connect() (*gorm.DB, error) {
	if c.Config == nil {
		return nil, fmt.Errorf("database config is not set")
	}
	return NewPostgresConnection(*c.Config)
}

func (c *GormConnector) CheckHealth(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is not set")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying database connection: %w", err)
	}
	return sqlDB.Ping()
}
