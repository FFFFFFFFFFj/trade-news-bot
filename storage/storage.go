package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
)

type NewsItem struct {
	Title   string
	Link    string
	PubDate time.Time
}

// 🔹 Подключение к PostgreSQL
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

// 🔹 Миграции БД
func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY
		);`,
		`CREATE TABLE IF NOT EXISTS sources (
			url TEXT PRIMARY KEY
		);`,
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
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// 🔹 Добавление источника
func AddSource(db *sql.DB, url string) error {
	_, err := db.Exec(`INSERT INTO sources (url) VALUES ($1) ON CONFLICT DO NOTHING`, url)
	return err
}

// 🔹 Удаление источника
func RemoveSource(db *sql.DB, url string) error {
	_, err := db.Exec(`DELETE FROM sources WHERE url=$1`, url)
	return err
}

// 🔹 Получение всех источников
func GetAllSources(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT url FROM sources`)
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

// 🔹 Подписка
func Subscribe(db *sql.DB, userID int64, url string) error {
	_, err := db.Exec(`INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING`, userID)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO subscriptions (user_id, source_url) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, url)
	return err
}

// 🔹 Отписка
func Unsubscribe(db *sql.DB, userID int64, url string) error {
	_, err := db.Exec(`DELETE FROM subscriptions WHERE user_id=$1 AND source_url=$2`, userID, url)
	return err
}

// 🔹 Подписки пользователя
func GetUserSources(db *sql.DB, userID int64) ([]string, error) {
	rows, err := db.Query(`SELECT source_url FROM subscriptions WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

// 🔹 Кол-во подписок пользователя
func GetUserSubscriptionCount(db *sql.DB, userID int64) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE user_id=$1`, userID).Scan(&count)
	return count, err
}

// 🔹 Кол-во активных пользователей
func GetActiveUsersCount(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM subscriptions`).Scan(&count)
	return count, err
}

// 🔹 Непрочитанные новости
func GetUnreadNews(db *sql.DB, userID int64, limit int) ([]NewsItem, error) {
	rows, err := db.Query(`
		SELECT n.title, n.link, n.pub_date
		FROM news n
		JOIN subscriptions s ON s.source_url = n.source_url
		WHERE s.user_id=$1
		AND NOT EXISTS (
			SELECT 1 FROM user_read_news ur WHERE ur.user_id=$1 AND ur.news_id=n.link
		)
		ORDER BY n.pub_date DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []NewsItem
	for rows.Next() {
		var item NewsItem
		if err := rows.Scan(&item.Title, &item.Link, &item.PubDate); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// 🔹 Пометить новость как прочитанную
func MarkNewsAsRead(db *sql.DB, userID int64, link string) error {
	_, err := db.Exec(`INSERT INTO user_read_news (user_id, news_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, link)
	return err
}

// 🔹 Загрузка новостей из RSS
func FetchAndStoreNews(db *sql.DB) (map[int64][]NewsItem, error) {
	fp := gofeed.NewParser()
	allSources, err := GetAllSources(db)
	if err != nil {
		return nil, err
	}

	newsMap := make(map[int64][]NewsItem)

	for _, src := range allSources {
		feed, err := fp.ParseURL(src)
		if err != nil {
			log.Printf("Ошибка парсинга %s: %v", src, err)
			continue
		}

		for _, item := range feed.Items {
			pub := item.PublishedParsed
			if pub == nil {
				now := time.Now()
				pub = &now
			}

			// сохраняем в базу
			_, err := db.Exec(`INSERT INTO news (link, title, pub_date, source_url) 
				VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
				item.Link, item.Title, *pub, src)
			if err != nil {
				log.Printf("Ошибка вставки новости: %v", err)
				continue
			}

			// кому рассылать
			rows, err := db.Query(`SELECT user_id FROM subscriptions WHERE source_url=$1`, src)
			if err != nil {
				continue
			}
			defer rows.Close()

			for rows.Next() {
				var uid int64
				if err := rows.Scan(&uid); err == nil {
					newsMap[uid] = append(newsMap[uid], NewsItem{
						Title:   item.Title,
						Link:    item.Link,
						PubDate: *pub,
					})
				}
			}
		}
	}
	return newsMap, nil
}
func MustGetAllSources(db *sql.DB) []string {
	sources, _ := GetAllSources(db)
	return sources
}
