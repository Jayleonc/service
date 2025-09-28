package database

import (
	"fmt"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config represents Postgres configuration for GORM.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

var (
	mu      sync.RWMutex
	current *gorm.DB
)

// New returns a configured GORM database connection.
func New(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// Init constructs a database connection and records it as the global instance.
func Init(cfg Config) (*gorm.DB, error) {
	db, err := New(cfg)
	if err != nil {
		return nil, err
	}
	SetDefault(db)
	return db, nil
}

// SetDefault stores db as the global database connection.
func SetDefault(db *gorm.DB) {
	mu.Lock()
	defer mu.Unlock()
	current = db
}

// Default returns the global database connection, if one has been initialised.
func Default() *gorm.DB {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
