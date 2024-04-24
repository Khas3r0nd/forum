package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
)

const (
	insertSessionSQL = `INSERT INTO sessions (user_id, session_token, expires_at) VALUES ($1, $2, $3);`
	getSessionSQL    = `SELECT session_token, expires_at FROM sessions WHERE user_id = ?;`
	deleteSessionSQL = `DELETE FROM sessions WHERE session_token = ?;`
	updateSessionSQL = `UPDATE sessions SET session_token = ?, expires_at = ? WHERE user_id = ?`
)

type Session struct {
	UserID       int
	SessionToken string
	ExpiryAt     time.Time
}

type SessionManager struct {
	DB *sql.DB
}

func (sm *SessionManager) CreateSession(w http.ResponseWriter, r *http.Request, UserID int) error {
	var err error

	sessionToken, err := generateToken()
	if err != nil {
		return err
	}
	expiryAt := time.Now().Add(10 * time.Second)

	// Проверка наличия записи с UserID
	var existingSessionToken string
	err = sm.DB.QueryRow(getSessionSQL, UserID).Scan(&existingSessionToken)
	if err == sql.ErrNoRows {
		// Записи с UserID не существует, выполняем вставку новой записи
		_, err := sm.DB.Exec(insertSessionSQL, UserID, sessionToken, expiryAt)
		if err != nil {
			return err
		}
	} else {
		// Запись с UserID существует, выполняем обновление
		sessionToken, err = generateToken()
		if err != nil {
			return err
		}

		expiryAt = time.Now().Add(30 * time.Second)
		_, err = sm.DB.Exec(updateSessionSQL, sessionToken, expiryAt, UserID)
		if err != nil {
			return err
		}
	}
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  expiryAt,
		HttpOnly: true,
		MaxAge:   3600,
		Path:     "/",
	}

	http.SetCookie(w, cookie)
	return nil
}

func (sm *SessionManager) GetUserIDBySessionToken(sessionToken string) int {
	// Выполните запрос к базе данных, чтобы найти UserID по sessionToken
	var userID int
	err := sm.DB.QueryRow("SELECT user_id FROM sessions WHERE session_token = $1", sessionToken).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Сессия с указанным токеном не найдена
			return 0
		}
		// Обработка других ошибок базы данных
		return 0
	}

	// Возврат найденного UserID
	return userID
}

func (sm *SessionManager) DeleteSession(sessionID string) error {
	_, err := sm.DB.Exec(deleteSessionSQL, sessionID)
	return err
}

func generateToken() (string, error) {
	token, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return token.String(), nil
}
