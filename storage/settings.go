package storage

import "database/sql"

// Установить значение
func SetSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	return err
}

// Получить значение
func GetSetting(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM settings WHERE key=$1`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// Получить все настройки
func GetAllSettings(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			settings[k] = v
		}
	}
	return settings, nil
}
