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
	_ "github.com/lib/pq"
)

var DB_Status bool

// executeSchemaFile reads and executes SQL statements from schema.sql file
func executeSchemaFile(db *sqlx.DB) error {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Look for schema.sql in the current directory and parent directories
	schemaPath := findSchemaFile(wd)
	if schemaPath == "" {
		return fmt.Errorf("schema.sql file not found")
	}

	log.Printf("Found schema.sql at: %s", schemaPath)

	// Read the schema file
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	// Split the content by semicolons to execute statements individually
	statements := strings.Split(string(schemaContent), ";")

	// Execute each statement
	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		_, err := db.Exec(statement)
		if err != nil {
			// Check if table already exists error
			if strings.Contains(err.Error(), "already exists") {
				log.Printf("Table/Index already exists, skipping statement %d", i+1)
				continue
			}
			return fmt.Errorf("failed to execute statement %d: %w\nStatement: %s", i+1, err, statement)
		}
	}

	log.Printf("Schema executed successfully")
	return nil
}

// findSchemaFile searches for schema.sql in current directory and parent directories
func findSchemaFile(startDir string) string {
	currentDir := startDir
	for {
		schemaPath := filepath.Join(currentDir, "schema.sql")
		if _, err := os.Stat(schemaPath); err == nil {
			return schemaPath
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root directory
			break
		}
		currentDir = parentDir
	}
	return ""
}

func ConnectAndCreateDB(cfg config.PostgresConfig) (*sqlx.DB, error) {
	defaultConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
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

	// Execute schema.sql file to create tables and indexes
	if !exists {
		if err := executeSchemaFile(db); err != nil {
			return nil, fmt.Errorf("failed to execute schema.sql: %w", err)
		}
	}

	DB_Status = true
	sqlx.BindDriver("postgres", sqlx.DOLLAR)
	sqlx.NameMapper = func(s string) string { return s }
	return db, nil
}

func RetryConnectOnFailed(wait_amount time.Duration, db **sqlx.DB, cfg config.PostgresConfig) {
	if DB_Status {
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
