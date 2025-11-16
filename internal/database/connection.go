package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/lypolix/avito_test/internal/config"

	_ "github.com/lib/pq"
)

func ConnectWithRetry(dbConfig config.DatabaseConfig) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < dbConfig.MaxRetries; i++ {
		db, err = sql.Open("postgres", dbConfig.ConnectionString())
		if err != nil {
			log.Printf("open db error: %v", err)
			time.Sleep(dbConfig.RetryInterval)
			continue
		}

		err = db.Ping()
		if err != nil {
			log.Printf("ping db error: %v", err)
			time.Sleep(dbConfig.RetryInterval)
			continue
		}

		db.SetMaxOpenConns(dbConfig.MaxOpenConns)
		db.SetMaxIdleConns(dbConfig.MaxIdleConns)
		db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

		log.Printf("db connected")
		return db, nil
	}

	return nil, fmt.Errorf("db connect failed: %w", err)
}

func HealthCheck(db *sql.DB) error {
	return db.Ping()
}
