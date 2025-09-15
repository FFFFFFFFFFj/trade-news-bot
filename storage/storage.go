package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
)

func ConnectDB() (*sql.DB, error) {
	// Connection parametrs to your database, replace with real ones
	user := "postgres"
	password := "postgres"
	dbname := "news_feed_bot"
	host := "/var/run/postgresql"
	port := 5433
	psqlInfo := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable", user, password, dbname, host, port)
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
			url TEXT UNIQUE NOT NULL,
			owner_telegram_id BIGINT
		);`,
		`CREATE TABLE IF NOT EXISTS news (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			link TEXT UNIQUE NOT NULL,
			pub_date TIMESTAMP,
			source_url TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS user_read_news (
			user_id BIGINT NOT NULL,
			news_id INT NOT NULL,
			read_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, news_id),
			FOREIGN KEY (news_id) REFERENCES news(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS user_subscriptions (
			user_id BIGINT NOT NULL,
			source_id INT NOT NULL,
			PRIMARY KEY (user_id, source_id),
			FOREIGN KEY (source_id) REFERENCES rss_sources(id) ON DELETE CASCADE
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

// Получить все новости (без фильтрации на прочитанные)
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

// Получить непрочитанные новости для пользователя
func GetUnreadNews(db *sql.DB, userID int64, limit int) ([]rss.Item, error) {
	query := `
		SELECT title, link, COALESCE(to_char(pub_date, 'YYYY-MM-DD HH24:MI:SS'), '')
		FROM news
		WHERE id NOT IN (
			SELECT news_id FROM user_read_news WHERE user_id = $1
		)
		ORDER BY pub_date DESC
		LIMIT $2
	`
	rows, err := db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []rss.Item
	for rows.Next() {
		var item rss.Item
		var published string
		if err := rows.Scan(&item.Title, &item.Link, &published); err != nil {
			return nil, err
		}
		item.PubDate = published
		items = append(items, item)
	}
	return items, nil
}

// Отметить новость как прочитанную пользователем
func MarkNewsAsRead(db *sql.DB, userID int64, newsLink string) error {
	var newsID int
	err := db.QueryRow("SELECT id FROM news WHERE link = $1", newsLink).Scan(&newsID)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO user_read_news(user_id, news_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, newsID)
	return err
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

func AddSource(db *sql.DB, url string, ownerTelegramID int64) error {
	_, err := db.Exec("INSERT INTO rss_sources(url, owner_telegram_id) VALUES($1, $2) ON CONFLICT DO NOTHING", url, ownerTelegramID)
	return err
}

func RemoveSource(db *sql.DB, url string) error {
	_, err := db.Exec("DELETE FROM rss_sources WHERE url = $1", url)
	return err
}
