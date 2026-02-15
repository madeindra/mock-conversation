package data

import (
	"database/sql"

	"github.com/google/uuid"
)

type ChatUser struct {
	ID               string `json:"id"`
	Secret           string `json:"secret"`
	Language         string `json:"language"`
	SubtitleLanguage string `json:"subtitle_language"`
	Voice            string `json:"voice"`
}

func (d *Database) CreateChatUser(tx *sql.Tx, secret, language, subtitleLanguage, voice string) (*ChatUser, error) {
	id := uuid.New().String()
	_, err := tx.Exec("INSERT INTO chat_users (id, secret, language, subtitle_language, voice) VALUES (?, ?, ?, ?, ?)", id, secret, language, subtitleLanguage, voice)
	if err != nil {
		return nil, err
	}

	return &ChatUser{ID: id, Secret: secret, Language: language, SubtitleLanguage: subtitleLanguage, Voice: voice}, nil
}

func (d *Database) GetChatUser(id string) (*ChatUser, error) {
	var user ChatUser
	err := d.conn.QueryRow("SELECT id, secret, language, subtitle_language, voice FROM chat_users WHERE id = ?", id).Scan(&user.ID, &user.Secret, &user.Language, &user.SubtitleLanguage, &user.Voice)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
