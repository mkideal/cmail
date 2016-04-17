package repository

import (
	"database/sql"
	"os"
	"testing"
)

func TestRepo(t *testing.T) {
	dbsource := os.Getenv("SMTPD_DB_SOURCE")
	if dbsource == "" {
		t.Errorf("please set non-empty env SMTPD_DB_SOURCE")
		return
	}
	db, err := sql.Open("mysql", dbsource)
	if err != nil {
		t.Errorf("open db error: %v", err)
		return
	}
	repo, err := NewMysqlRepository(db)
	if err != nil {
		t.Errorf("NewMysqlRepository error: %v", err)
		return
	}
	if err := repo.SaveEmail("from", "to", []byte("hello")); err != nil {
		t.Errorf("SaveEmail error: %v", err)
		return
	}
}
