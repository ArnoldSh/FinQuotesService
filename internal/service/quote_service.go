package service

import (
	"FinQuotesService/internal/model"
	"database/sql"
)

type QuoteServiceInterface interface {
	InsertPendingQuote(currency string) (string, error)
	UpdateQuote(id string, price float64, status model.Status) error
	GetQuoteById(id string) (model.Quote, error)
	GetLastQuote(currency string, status model.Status) (model.Quote, error)
}

type QuoteService struct {
	InsertPendingStmt *sql.Stmt
	UpdateQuoteStmt   *sql.Stmt
	GetQuoteByIdStmt  *sql.Stmt
	GetLastQuoteStmt  *sql.Stmt
}

func NewQuoteService(db *sql.DB) *QuoteService {
	insertPendingStmt, err := db.Prepare(`INSERT INTO quotes (currency, status) VALUES ($1, 'pending') ON CONFLICT (currency) WHERE status = 'pending' DO NOTHING RETURNING id;`)
	updateQuoteStmt, err := db.Prepare(`UPDATE quotes SET price=$1, updated_at=now(), status=$2 WHERE id=$3`)
	getQuoteByIdStmt, err := db.Prepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE id =$1`)
	getLastQuoteStmt, err := db.Prepare(`SELECT id, currency, price, updated_at, status FROM quotes WHERE currency=$1 AND status=$2 ORDER BY updated_at DESC LIMIT 1`)
	if err != nil {
		panic(err)
	}
	return &QuoteService{
		InsertPendingStmt: insertPendingStmt,
		UpdateQuoteStmt:   updateQuoteStmt,
		GetQuoteByIdStmt:  getQuoteByIdStmt,
		GetLastQuoteStmt:  getLastQuoteStmt,
	}
}

func (s *QuoteService) InsertPendingQuote(currency string) (string, error) {
	var id string
	row := s.InsertPendingStmt.QueryRow(currency)
	err := row.Scan(&id)
	return id, err
}

func (s *QuoteService) UpdateQuote(id string, price float64, status model.Status) error {
	_, err := s.UpdateQuoteStmt.Exec(price, status, id)
	return err
}

func (s *QuoteService) GetQuoteById(id string) (model.Quote, error) {
	row := s.GetQuoteByIdStmt.QueryRow(id)
	var q model.Quote
	err := row.Scan(&q.ID, &q.Currency, &q.Price, &q.UpdatedAt, &q.Status)
	return q, err
}

func (s *QuoteService) GetLastQuote(currency string, status model.Status) (model.Quote, error) {
	row := s.GetLastQuoteStmt.QueryRow(currency, status)
	var q model.Quote
	err := row.Scan(&q.ID, &q.Currency, &q.Price, &q.UpdatedAt, &q.Status)
	return q, err
}
