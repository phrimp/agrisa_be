package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"policy-service/internal/config"
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
	DBStatus = true

	return db, nil
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
