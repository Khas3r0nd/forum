package models

import "database/sql"

type DbAccess struct {
	db *sql.DB
}

func NewDbAccess(db *sql.DB) *DbAccess {
	return &DbAccess{db: db}
}
