package models

import (
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword string
}

// Define a new UserModel type which wraps a database connection pool.
type UserModel struct {
	DB *sql.DB
}

func (u *UserModel) GetUsername(name string) error {
	stmt := `SELECT username FROM users WHERE username = $1`
	var username string
	err := u.DB.QueryRow(stmt, name).Scan(&username)

	if err == sql.ErrNoRows {
		// Username doesn't exist, it's available
		return nil
	} else if err != nil {
		// An error occurred during the query
		return err
	} else {
		// Username already exists
		return ErrDuplicateUsername
	}
}

func (u *UserModel) GetEmail(email string) error {
	stmt := `SELECT email FROM users WHERE email = $1`
	var mail string
	err := u.DB.QueryRow(stmt, email).Scan(&mail)

	if err == sql.ErrNoRows {
		// Username doesn't exist, it's available
		return nil
	} else if err != nil {
		// An error occurred during the query
		return err
	} else {
		// Username already exists
		return ErrDuplicateEmail
	}
}

func (u *UserModel) ReturnUserID(email string) (int, error) {
	stmt := `SELECT user_id FROM users WHERE email = $1`
	var id int

	err := u.DB.QueryRow(stmt, email).Scan(&id)

	if err == sql.ErrNoRows {
		// Username doesn't exist, it's available
		return 0, err
	} else if err != nil {
		// An error occurred during the query
		return 0, err
	} else {
		// Username already exists
		return id, nil
	}
}

// We'll use the Insert method to add a new record to the "users" table.
func (u *UserModel) Insert(name, email, hashPassword string) {
	// db, err := sql.Open("sqlite3", "./forum.db")
	// defer db.Close()
	stmt := `INSERT INTO users (username, email, hash_password)
	VALUES ($1, $2, $3)`
	u.DB.Exec(stmt, name, email, hashPassword)
}

func (u *UserModel) Authenticate(email, password string) (int, error) {
	var id int
	var hashedPassword []byte

	stmt := "SELECT user_id, hash_password FROM users WHERE email = ?"
	err := u.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}
	fmt.Println(string(hashedPassword))
	fmt.Println(password)
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}
	return id, nil
}

// dlya Google oauth
func (u *UserModel) DoesUserExistByEmail(email string) (bool, error) {
	stmt := `SELECT EXISTS (SELECT 1 FROM users WHERE email=?)`
	var exists bool
	err := u.DB.QueryRow(stmt, email).Scan(&exists)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	} else {
		return false, nil
	}
}

// dlya Github oauth
func (u *UserModel) DoesUserExistByName(name string) (bool, error) {
	stmt := `SELECT EXISTS (SELECT 1 FROM users WHERE username=?)`
	var exists bool
	err := u.DB.QueryRow(stmt, name).Scan(&exists)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	} else {
		return false, nil
	}
}
