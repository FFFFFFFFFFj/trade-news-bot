package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

func ConnectDB() (*sql.DB, error) {
	// Connection parametrs to yuor database, replace with real ones
	user := "postgres"
	password := "yourpassword"
	dbname := "yourdbname"
	host := "localhost"
	port := 5432


	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to PostgresSQL")
	return db, nil
}

func Migrate(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS rss_sources (
			id SERIAL PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			owner_telegram_id BIGINT NOT NULL
		);
	`
	_, err := db.Exec(query)
	return err
}

