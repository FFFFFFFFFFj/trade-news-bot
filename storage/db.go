package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Подключение к PostgreSQL
func ConnectDB() (*sql.DB, error) {
	user := "postgres"
	password := "postgres"
	dbname := "news_feed_bot"
	host := "/var/run/postgresql"
	port := 5433

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		user, password, dbname, host, port)
	return sql.Open("postgres", connStr)
}

// Миграции БД
func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (id BIGINT PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS sources (url TEXT PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
			source_url TEXT REFERENCES sources(url) ON DELETE CASCADE,
			PRIMARY KEY (user_id, source_url)
		);`,
		`CREATE TABLE IF NOT EXISTS news (
			link TEXT PRIMARY KEY,
			title TEXT,
			pub_date TIMESTAMP,
			source_url TEXT REFERENCES sources(url) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS user_read_news (
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
			news_id TEXT REFERENCES news(link) ON DELETE CASCADE,
			PRIMARY KEY (user_id, news_id)
		);`,
		`CREATE TABLE IF NOT EXISTS user_autopost (
			user_id BIGINT PRIMARY KEY,
			times TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
    		key TEXT PRIMARY KEY,
    		value TEXT
		);`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
