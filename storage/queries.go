package storage

import (
	"database/sql"
	"time"
)

type User struct {
	ID int64
}

type News struct {
	Title string
	URL   string
}

// Вернуть всех пользователей
func GetAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT id FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Новости за сегодня для пользователя (с пагинацией)
func GetTodayNewsForUser(db *sql.DB, userID int64, page, pageSize int) ([]News, error) {
	offset := (page - 1) * pageSize
	rows, err := db.Query(`
		SELECT title, url 
		FROM news 
		WHERE date = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`, time.Now().Format("2006-01-02"), pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []News
	for rows.Next() {
		var n News
		if err := rows.Scan(&n.Title, &n.URL); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, nil
}

// Кол-во новостей за сегодня для пользователя
func GetTodayNewsCountForUser(db *sql.DB, userID int64) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM news 
		WHERE date = $1
	`, time.Now().Format("2006-01-02")).Scan(&count)
	return count, err
}
