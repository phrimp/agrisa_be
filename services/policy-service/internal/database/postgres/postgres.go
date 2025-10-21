package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"policy-service/internal/config"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var DBStatus bool

func ConnectAndCreateDB(cfg config.PostgresConfig) (*sqlx.DB, error) {
	defaultConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=policy_service sslmode=disable",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	// Logging the connection string values (excluding password for security)
	log.Printf("Connecting to PostgreSQL with: host=%s, port=%s, user=%s, dbname=policy_service, password=%s",

		cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	defaultDB, err := sql.Open("postgres", defaultConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to default postgres db: %w", err)
	}
	defer defaultDB.Close()

	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	err = defaultDB.QueryRow(checkQuery, cfg.DBname).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !exists {
		createQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.DBname)
		_, err = defaultDB.Exec(createQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to create database %s: %w", cfg.DBname, err)
		}
		fmt.Printf("Database '%s' created successfully\n", cfg.DBname)
	} else {
		fmt.Printf("Database '%s' already exists\n", cfg.DBname)
	}

	targetConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBname)

	db, err := sqlx.Connect("postgres", targetConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping target database: %w", err)
	}

	// Execute schema.sql if database was newly created
	if !exists {
		if err := executeSchema(db); err != nil {
			log.Printf("Warning: Failed to execute schema.sql: %v", err)
			// Don't return error, just log warning to allow manual schema setup
		}
	}

	DBStatus = true
	return db, nil
}

// executeSchema reads and executes the schema.sql file
func executeSchema(db *sqlx.DB) error {
	// Find schema.sql file - check multiple possible locations
	schemaLocations := []string{
		"schema.sql",          // Current directory
		"./schema.sql",        // Relative to current directory
		"../../../schema.sql", // Relative to this file location
		"/app/schema.sql",     // Docker container location
		filepath.Join(os.Getenv("PWD"), "schema.sql"), // Working directory
	}

	var schemaPath string
	var schemaContent []byte
	var err error

	// Try to find schema.sql in possible locations
	for _, location := range schemaLocations {
		if _, err := os.Stat(location); err == nil {
			schemaPath = location
			break
		}
	}

	if schemaPath == "" {
		return fmt.Errorf("schema.sql not found in any expected locations: %v", schemaLocations)
	}

	// Read schema file
	schemaContent, err = os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema.sql from %s: %w", schemaPath, err)
	}

	log.Printf("Executing schema from: %s", schemaPath)

	// Split schema into individual statements (split by semicolon)
	statements := strings.Split(string(schemaContent), ";")

	successCount := 0
	for i, statement := range statements {
		// Clean up the statement
		statement = strings.TrimSpace(statement)
		if statement == "" || strings.HasPrefix(statement, "--") {
			continue // Skip empty lines and comments
		}

		// Execute each statement
		_, err := db.Exec(statement)
		if err != nil {
			log.Printf("Warning: Failed to execute statement %d: %v", i+1, err)
			log.Printf("Statement: %s", statement[:min(100, len(statement))])
			// Continue executing other statements even if one fails
		} else {
			successCount++
		}
	}

	log.Printf("Schema execution completed. Successfully executed %d statements", successCount)
	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func RetryConnectOnFailed(wait_amount time.Duration, db **sqlx.DB, cfg config.PostgresConfig) {
	if DBStatus {
		log.Printf("false database lost connnection alert! abort retry")
		return
	}

	// Check if *db is nil before using it
	if *db != nil {
		cur_db := *db
		err := cur_db.Ping()
		if err == nil {
			log.Printf("database connection is healthy, no retry needed")
			return
		}
		log.Printf("failed to ping target database: %s, retry db connection\n", err)
	} else {
		log.Printf("database connection is nil, attempting to reconnect...")
	}

	newDB, err := ConnectAndCreateDB(cfg)
	if err == nil {
		*db = newDB
		log.Printf("database retry connection successfully\n")
		return
	}
	log.Printf("failed to retry connect database: %s, next retry in %v\n", err, wait_amount)
	time.Sleep(wait_amount)

	RetryConnectOnFailed(wait_amount, db, cfg)
}
