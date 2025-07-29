package service

import (
	"FinQuotesService/internal/model"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"testing"
	"time"
)

func initMocks(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return db, mock
}

func TestQuoteService_InsertPendingQuote(t *testing.T) {
	db, mock := initMocks(t)
	defer db.Close()

	testUuid := uuid.New().String()
	testCurrency := "USD/EUR"

	rows := sqlmock.NewRows([]string{"id"}).AddRow(testUuid)

	expectedPrepare := mock.ExpectPrepare(`INSERT INTO quotes \(currency, status\) VALUES \(\$1, 'pending'\) ON CONFLICT \(currency\) WHERE status = 'pending' DO NOTHING RETURNING id;`)
	mock.ExpectPrepare(`UPDATE quotes SET price=\$1, updated_at=now\(\), status=\$2 WHERE id=\$3`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE id =\$1`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE currency=\$1 AND status=\$2 ORDER BY updated_at DESC LIMIT 1`)

	expectedPrepare.ExpectQuery().
		WithArgs(testCurrency).
		WillReturnRows(rows)

	service := NewQuoteService(db)
	quoteId, err := service.InsertPendingQuote(testCurrency)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if quoteId != testUuid {
		t.Errorf("expected ID %s, got %s", testUuid, quoteId)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestQuoteService_UpdateQuote(t *testing.T) {
	db, mock := initMocks(t)
	defer db.Close()

	mock.ExpectPrepare(`INSERT INTO quotes \(currency, status\) VALUES \(\$1, 'pending'\) ON CONFLICT \(currency\) WHERE status = 'pending' DO NOTHING RETURNING id;`)
	expectedPrepare := mock.ExpectPrepare(`UPDATE quotes SET price=\$1, updated_at=now\(\), status=\$2 WHERE id=\$3`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE id =\$1`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE currency=\$1 AND status=\$2 ORDER BY updated_at DESC LIMIT 1`)

	expectedPrepare.ExpectExec().
		WithArgs(1.23, model.StatusDone, "uuid-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	service := NewQuoteService(db)
	err := service.UpdateQuote("uuid-1", 1.23, model.StatusDone)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetQuoteById_Success(t *testing.T) {
	db, mock := initMocks(t)
	defer db.Close()

	testID := "test-uuid"
	testCurrency := "USD/EUR"
	testPrice := 1.23
	testTime := time.Now().Truncate(time.Second)
	testStatus := model.StatusDone

	rows := sqlmock.NewRows([]string{"id", "currency", "price", "updated_at", "status"}).
		AddRow(testID, testCurrency, testPrice, testTime, testStatus)

	mock.ExpectPrepare(`INSERT INTO quotes \(currency, status\) VALUES \(\$1, 'pending'\) ON CONFLICT \(currency\) WHERE status = 'pending' DO NOTHING RETURNING id;`)
	mock.ExpectPrepare(`UPDATE quotes SET price=\$1, updated_at=now\(\), status=\$2 WHERE id=\$3`)
	expectedPrepare := mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE id =\$1`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE currency=\$1 AND status=\$2 ORDER BY updated_at DESC LIMIT 1`)

	expectedPrepare.ExpectQuery().
		WithArgs(testID).
		WillReturnRows(rows)

	srv := NewQuoteService(db)
	quote, err := srv.GetQuoteById(testID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if quote.ID != testID {
		t.Errorf("expected ID %s, got %s", testID, quote.ID)
	}
	if quote.Currency != testCurrency {
		t.Errorf("expected Currency %s, got %s", testCurrency, quote.Currency)
	}
	if *quote.Price != testPrice {
		t.Errorf("expected Price %v, got %v", testPrice, quote.Price)
	}
	if !quote.UpdatedAt.Equal(testTime) {
		t.Errorf("expected UpdatedAt %v, got %v", testTime, quote.UpdatedAt)
	}
	if quote.Status != testStatus {
		t.Errorf("expected Status %s, got %s", testStatus, quote.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetQuoteById_NotFound(t *testing.T) {
	db, mock := initMocks(t)
	defer db.Close()

	notExistID := "not-exist-uuid"

	mock.ExpectPrepare(`INSERT INTO quotes \(currency, status\) VALUES \(\$1, 'pending'\) ON CONFLICT \(currency\) WHERE status = 'pending' DO NOTHING RETURNING id;`)
	mock.ExpectPrepare(`UPDATE quotes SET price=\$1, updated_at=now\(\), status=\$2 WHERE id=\$3`)
	expectedPrepare := mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE id =\$1`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE currency=\$1 AND status=\$2 ORDER BY updated_at DESC LIMIT 1`)

	expectedPrepare.ExpectQuery().
		WithArgs(notExistID).
		WillReturnError(sql.ErrNoRows)

	srv := NewQuoteService(db)
	_, err := srv.GetQuoteById(notExistID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetLastQuote_Success(t *testing.T) {
	db, mock := initMocks(t)
	defer db.Close()

	testID := "test-uuid"
	testCurrency := "USD/EUR"
	testPrice := 1.23
	testTime := time.Now().Truncate(time.Second)
	testStatus := model.StatusDone

	rows := sqlmock.NewRows([]string{"id", "currency", "price", "updated_at", "status"}).
		AddRow(testID, testCurrency, testPrice, testTime, testStatus)

	mock.ExpectPrepare(`INSERT INTO quotes \(currency, status\) VALUES \(\$1, 'pending'\) ON CONFLICT \(currency\) WHERE status = 'pending' DO NOTHING RETURNING id;`)
	mock.ExpectPrepare(`UPDATE quotes SET price=\$1, updated_at=now\(\), status=\$2 WHERE id=\$3`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE id =\$1`)
	expectedPrepare := mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE currency=\$1 AND status=\$2 ORDER BY updated_at DESC LIMIT 1`)

	expectedPrepare.ExpectQuery().
		WithArgs(testCurrency, testStatus).
		WillReturnRows(rows)

	srv := NewQuoteService(db)
	quote, err := srv.GetLastQuote(testCurrency)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if quote.ID != testID {
		t.Errorf("expected ID %s, got %s", testID, quote.ID)
	}
	if quote.Currency != testCurrency {
		t.Errorf("expected Currency %s, got %s", testCurrency, quote.Currency)
	}
	if *quote.Price != testPrice {
		t.Errorf("expected Price %v, got %v", testPrice, quote.Price)
	}
	if !quote.UpdatedAt.Equal(testTime) {
		t.Errorf("expected UpdatedAt %v, got %v", testTime, quote.UpdatedAt)
	}
	if quote.Status != testStatus {
		t.Errorf("expected Status %s, got %s", testStatus, quote.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetLastQuote_NotFound(t *testing.T) {
	db, mock := initMocks(t)
	defer db.Close()

	testCurrency := "USD/EUR"
	testStatus := model.StatusDone

	mock.ExpectPrepare(`INSERT INTO quotes \(currency, status\) VALUES \(\$1, 'pending'\) ON CONFLICT \(currency\) WHERE status = 'pending' DO NOTHING RETURNING id;`)
	mock.ExpectPrepare(`UPDATE quotes SET price=\$1, updated_at=now\(\), status=\$2 WHERE id=\$3`)
	mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE id =\$1`)
	expectedPrepare := mock.ExpectPrepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE currency=\$1 AND status=\$2 ORDER BY updated_at DESC LIMIT 1`)

	expectedPrepare.ExpectQuery().
		WithArgs(testCurrency, testStatus).
		WillReturnError(sql.ErrNoRows)

	srv := NewQuoteService(db)
	_, err := srv.GetLastQuote(testCurrency)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
