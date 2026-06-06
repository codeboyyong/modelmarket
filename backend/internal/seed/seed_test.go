package seed

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestResetTruncatesSeedTables(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("delete from audit_logs").WillReturnResult(sqlmock.NewResult(0, 1))

	if err := Reset(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResetReportsDatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("delete from audit_logs").WillReturnError(sql.ErrConnDone)

	if err := Reset(context.Background(), db); err == nil {
		t.Fatal("expected reset error")
	}
}
