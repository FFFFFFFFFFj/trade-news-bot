package storage

import "database/sql"

// Работа с пользователями
func GetAllUsers(db *sql.DB) ([]int64, error) {
	rows, err := db.Query(`SELECT id FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		users = append(users, id)
	}
	return users, nil
}

func GetUsersCount(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func GetActiveUsersCount(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM subscriptions`).Scan(&count)
	return count, err
}

func GetAutopostUsersCount(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM user_autopost WHERE times IS NOT NULL AND times <> '[]'`).Scan(&count)
	return count, err
}
