package database

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config represents Postgres configuration for GORM.
type Config struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	Params   map[string]string
}

var (
	mu      sync.RWMutex
	current *gorm.DB
)

// New returns a configured GORM database connection.
func New(cfg Config) (*gorm.DB, error) {
	driver := strings.TrimSpace(cfg.Driver)
	if driver == "" {
		driver = "postgres"
	}

	switch strings.ToLower(driver) {
	case "postgres", "postgresql":
		dsn := buildPostgresDSN(cfg)
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("database: unsupported driver %q", cfg.Driver)
	}
}

func buildPostgresDSN(cfg Config) string {
	parts := []string{}
	if cfg.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", cfg.Host))
	}
	if cfg.Port != 0 {
		parts = append(parts, fmt.Sprintf("port=%d", cfg.Port))
	}
	if cfg.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", cfg.User))
	}
	if cfg.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", cfg.Password))
	}
	if cfg.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", cfg.Database))
	}
	if cfg.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", cfg.SSLMode))
	}

	if len(cfg.Params) > 0 {
		keys := make([]string, 0, len(cfg.Params))
		for k := range cfg.Params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := cfg.Params[k]
			if v == "" {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return strings.Join(parts, " ")
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
