package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
)

func ConnectDB() (*sql.DB, error) {
	// Connection parametrs to yuor database, replace with real ones
	user := "postgres"
    password := "postgres"
	dbname := "news_feed_bot"
	host := "/var/run/postgresql"
	port := 5433


	psqlInfo := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d  sslmode=disable", user, password,dbname, host, port)

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
	queries := []string{
		`CREATE TABLE IF NOT EXISTS rss_sources (
			id SERIAL PRIMARY KEY,
			url TEXT UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS news (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			link TEXT UNIQUE NOT NULL,
			pub_date TIMESTAMP,
			source_url TEXT
		);`,
	}
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	} 
	return nil
}

func SaveNews(db *sql.DB, item rss.Item, sourceURL string) error {
	query := `
		INSERT INTO news (title, link, pub_date, source_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (link) DO NOTHING;
	`
	_, err := db.Exec(query, item.Title, item.Link, item.PubDate, sourceURL)
	return err
}

func GetLatestNews(db *sql.DB, limit int) ([]rss.Item, error) {
	query := `
		SELECT title, link, COALESCE(to_char(pub_date, 'YYYY-MM-DD HH24:MI:SS'), ''), source_url
		FROM news
		ORDER BY pub_date DESC NULLS LAST
		LIMIT $1
	`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []rss.Item
	for rows.Next() {
		var item rss.Item
		var published string
		err = rows.Scan(&item.Title, &item.Link, &published, new(string))
		if err != nil {
			return nil, err
		}
		item.PubDate = published
		items = append(items, item)
	}
	return items, nil
}

func GetAllSources(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT url FROM rss_sources")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		sources = append(sources, url)
	}
	return sources, nil
}

func AddSource(db *sql.DB, url string) error {
	_, err := db.Exec("INSERT INTO rss_sources(url) VALUES($1) ON CONFLICT DO NOTHING", url)
	return err
}

func RemoveSource(db *sql.DB, url string) error {
	_, err := db.Exec("DELETE FROM rss_sources WHERE url = $1", url)
	return err
}
