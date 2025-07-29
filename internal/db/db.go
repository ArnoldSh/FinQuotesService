package db

import (
	"database/sql"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

var (
	db     *sql.DB
	dbOnce sync.Once
)

func InitializeDb() *sql.DB {
	dbOnce.Do(func() {
		dsn := os.Getenv("DB_DSN")
		if dsn == "" {
			dsn = "postgres://user:pass@localhost:5432/quotes?sslmode=disable"
		}
		var err error
		for i := 0; i < 20; i++ {
			db, err = sql.Open("postgres", dsn)
			if err == nil {
				err = db.Ping()
				if err == nil {
					break
				}
			}
			log.Printf("Waiting for database... (%d/20)", i+1)
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			log.Fatalf("Failed to connect DB: %v", err)
		}
		initSchema(db, "./init.sql")
	})
	return db
}

func initSchema(db *sql.DB, path string) {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}
	log.Println(string(sqlBytes))
	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
}
