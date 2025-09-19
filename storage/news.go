package storage

import (
	"database/sql"
	"log"
	"time"

	"github.com/mmcdole/gofeed"
)

// NewsItem представляет новость
type NewsItem struct {
	Title   string
	Link    string
	PubDate time.Time
	Source  string
}

// Добавление источника
func AddSource(db *sql.DB, url string) error {
	_, err := db.Exec(`INSERT INTO sources (url) VALUES ($1) ON CONFLICT DO NOTHING`, url)
	return err
}

// Удаление источника
func RemoveSource(db *sql.DB, url string) error {
	_, err := db.Exec(`DELETE FROM sources WHERE url=$1`, url)
	return err
}

// Получение всех источников
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

// MustGetAllSources возвращает все источники без ошибки
func MustGetAllSources(db *sql.DB) []string {
	sources, _ := GetAllSources(db)
	return sources
}

// Загрузка и сохранение новостей из RSS
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

			_, err := db.Exec(`INSERT INTO news (link, title, pub_date, source_url) 
				VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
				item.Link, item.Title, *pub, src)
			if err != nil {
				log.Printf("Ошибка вставки новости: %v", err)
				continue
			}

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
						Source:  src,
					})
				}
			}
		}
	}

	return newsMap, nil
}

// Получить новости за сегодня для пользователя (кол-во)
func GetTodayNewsCountForUser(db *sql.DB, userID int64) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM news
		WHERE source_url IN (SELECT source_url FROM subscriptions WHERE user_id = $1)
		AND pub_date::date = CURRENT_DATE
	`, userID).Scan(&count)
	return count, err
}

// Получить новости за сегодня с пагинацией для пользователя
func GetTodayNewsPageForUser(db *sql.DB, userID int64, page, pageSize int) ([]NewsItem, error) {
	offset := (page - 1) * pageSize

	rows, err := db.Query(`
		SELECT title, link, pub_date, source_url
		FROM news
		WHERE source_url IN (SELECT source_url FROM subscriptions WHERE user_id = $1)
		AND pub_date::date = CURRENT_DATE
		ORDER BY pub_date DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []NewsItem
	for rows.Next() {
		var n NewsItem
		if err := rows.Scan(&n.Title, &n.Link, &n.PubDate, &n.Source); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, nil
}

// Получить последние новости с пагинацией для пользователя
func GetLatestNewsPageForUser(db *sql.DB, userID int64, page, pageSize int) ([]NewsItem, error) {
	offset := (page - 1) * pageSize

	rows, err := db.Query(`
		SELECT title, link, pub_date, source_url
		FROM news
		WHERE source_url IN (SELECT source_url FROM subscriptions WHERE user_id = $1)
		ORDER BY pub_date DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []NewsItem
	for rows.Next() {
		var n NewsItem
		if err := rows.Scan(&n.Title, &n.Link, &n.PubDate, &n.Source); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, nil
}
