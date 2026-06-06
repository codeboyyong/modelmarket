package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRunMigrationsAppliesPendingFiles(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001_test.sql"), []byte("create table test_table(id text primary key);"), 0644); err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec("create table if not exists schema_migrations").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("select exists").WithArgs("001_test.sql").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectExec("create table test_table").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("insert into schema_migrations").WithArgs("001_test.sql").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := RunMigrations(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunMigrationsSkipsAppliedFiles(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001_test.sql"), []byte("select 1;"), 0644); err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec("create table if not exists schema_migrations").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("select exists").WithArgs("001_test.sql").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	if err := RunMigrations(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
