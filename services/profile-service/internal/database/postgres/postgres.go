package postgres

import (
	"fmt"
	"log"
	"profile-service/internal/config"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var DB_Status bool

func ConnectAndCreateDB(cfg config.PostgresConfig) (*sqlx.DB, error) {
	targetConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=profile_service sslmode=disable",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	db, err := sqlx.Connect("postgres", targetConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target database: %w", err)
	}

	// list databases for debugging
	err = ListDatabases(db)
	if err != nil {
		log.Printf("failed to list databases: %s", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping target database: %w", err)
	}
	DB_Status = true

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

func ListDatabases(db *sqlx.DB) error {
	var databases []string
	query := `SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname`

	err := db.Select(&databases, query)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	log.Println("Current databases:")
	for _, dbName := range databases {
		fmt.Println(dbName)
	}

	return nil
}
