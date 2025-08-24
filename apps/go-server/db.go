package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
)

func openDB(dsn string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dsn+"?_fk=1&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	sqlBytes, err := os.ReadFile("sql/001_init.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}
