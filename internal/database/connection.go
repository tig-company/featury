package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns a default database configuration
func DefaultConfig() *Config {
	return &Config{
		Host:            "localhost",
		Port:            5432,
		User:            "featury",
		Password:        "featury",
		Database:        "featury",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 15,
	}
}

// DSN builds a PostgreSQL connection string from the config
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// Connect establishes a connection to the PostgreSQL database
func Connect(config *Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", config.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// CreateDatabase creates the database if it doesn't exist
func CreateDatabase(config *Config) error {
	// Connect to postgres database to create the target database
	tempConfig := *config
	tempConfig.Database = "postgres"
	
	db, err := sql.Open("postgres", tempConfig.DSN())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer db.Close()

	// Check if database exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", config.Database).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !exists {
		// Create the database
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", config.Database))
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", config.Database, err)
		}
		fmt.Printf("Database %s created successfully\n", config.Database)
	} else {
		fmt.Printf("Database %s already exists\n", config.Database)
	}

	return nil
}

// DropDatabase drops the database if it exists
func DropDatabase(config *Config) error {
	// Connect to postgres database to drop the target database
	tempConfig := *config
	tempConfig.Database = "postgres"
	
	db, err := sql.Open("postgres", tempConfig.DSN())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer db.Close()

	// Terminate connections to the database
	_, err = db.Exec(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`, config.Database)
	if err != nil {
		return fmt.Errorf("failed to terminate database connections: %w", err)
	}

	// Drop the database
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", config.Database))
	if err != nil {
		return fmt.Errorf("failed to drop database %s: %w", config.Database, err)
	}

	fmt.Printf("Database %s dropped successfully\n", config.Database)
	return nil
}

// ConnectWithURL connects to database using a database URL string
func ConnectWithURL(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set sensible defaults for connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 15)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// ParseDatabaseURL parses a database URL into a Config struct
func ParseDatabaseURL(databaseURL string) (*Config, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config := DefaultConfig()

	config.Host = u.Hostname()
	if port := u.Port(); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid port in database URL: %w", err)
		}
		config.Port = p
	}

	if u.User != nil {
		config.User = u.User.Username()
		if password, ok := u.User.Password(); ok {
			config.Password = password
		}
	}

	if len(u.Path) > 1 {
		config.Database = u.Path[1:] // Remove leading '/'
	}

	// Parse query parameters
	query := u.Query()
	if sslMode := query.Get("sslmode"); sslMode != "" {
		config.SSLMode = sslMode
	}

	return config, nil
}

// Migrate runs database migrations using the default migrations directory
func Migrate(databaseURL string) error {
	db, err := ConnectWithURL(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database for migrations: %w", err)
	}
	defer db.Close()

	return MigrateUp(db, "./internal/database/migrations")
}