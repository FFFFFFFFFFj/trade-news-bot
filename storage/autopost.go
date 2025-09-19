package storage

import (
	"database/sql"
	"encoding/json"
)

func SetUserAutopost(db *sql.DB, userID int64, times []string) error {
	timesJSON, _ := json.Marshal(times)
	_, err := db.Exec(`
		INSERT INTO user_autopost (user_id, times)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET times = EXCLUDED.times
	`, userID, string(timesJSON))
	return err
}

func GetUserAutopost(db *sql.DB, userID int64) ([]string, error) {
	var timesJSON string
	err := db.QueryRow(`SELECT times FROM user_autopost WHERE user_id=$1`, userID).Scan(&timesJSON)
	if err != nil {
		return []string{}, nil
	}
	var times []string
	_ = json.Unmarshal([]byte(timesJSON), &times)
	return times, nil
}

func GetAllAutopostUsers(db *sql.DB) (map[int64][]string, error) {
	rows, err := db.Query(`SELECT user_id, times FROM user_autopost`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]string)
	for rows.Next() {
		var uid int64
		var timesJSON string
		if err := rows.Scan(&uid, &timesJSON); err != nil {
			continue
		}
		var times []string
		_ = json.Unmarshal([]byte(timesJSON), &times)
		result[uid] = times
	}
	return result, nil
}
