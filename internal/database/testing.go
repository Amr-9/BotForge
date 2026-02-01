package database

import "github.com/jmoiron/sqlx"

// NewMySQLFromDB creates a MySQL wrapper from an existing sqlx.DB
// This is useful for testing with mock databases
func NewMySQLFromDB(db *sqlx.DB) *MySQL {
	return &MySQL{db: db}
}
