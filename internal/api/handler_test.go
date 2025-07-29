package api

import (
	"FinQuotesService/internal/model"
	"FinQuotesService/internal/worker"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockQuoteService struct {
	InsertPendingQuoteFunc func(currency string) (string, error)
	UpdateQuoteFunc        func(id string, price float64, status model.Status) error
	GetQuoteByIdFunc       func(id string) (model.Quote, error)
	GetLastQuoteFunc       func(currency string, status model.Status) (model.Quote, error)
}

func (m *MockQuoteService) InsertPendingQuote(currency string) (string, error) {
	return m.InsertPendingQuoteFunc(currency)
}
func (m *MockQuoteService) UpdateQuote(id string, price float64, status model.Status) error {
	return m.UpdateQuoteFunc(id, price, status)
}
func (m *MockQuoteService) GetQuoteById(id string) (model.Quote, error) {
	return m.GetQuoteByIdFunc(id)
}
func (m *MockQuoteService) GetLastQuote(currency string, status model.Status) (model.Quote, error) {
	return m.GetLastQuoteFunc(currency, status)
}

func TestPostStartAsyncUpdateQuote_NewPending(t *testing.T) {
	supported := map[string]bool{"USD/EUR": true}
	jobChan := make(chan worker.QuoteJob, 1)
	mock := &MockQuoteService{
		GetLastQuoteFunc: func(currency string, status model.Status) (model.Quote, error) {
			return model.Quote{}, sql.ErrNoRows
		},
		InsertPendingQuoteFunc: func(currency string) (string, error) {
			if currency != "USD/EUR" {
				t.Errorf("expected USD/EUR, got %s", currency)
			}
			return "uuid-123", nil
		},
	}
	h := &Handler{SupportedCurrency: supported, Srv: mock, JobChan: jobChan}

	body := []byte(`{"currency":"USD/EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/quotes/update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.PostStartAsyncUpdateQuote(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	var out UpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if out.RequestId != "uuid-123" {
		t.Fatalf("expected request_id uuid-123, got %s", out.RequestId)
	}

	select {
	case job := <-jobChan:
		if job.Id != "uuid-123" {
			t.Errorf("wrong job.Id: %s", job.Id)
		}
		if job.Currency != "USD/EUR" {
			t.Errorf("wrong job.Currency: %s", job.Currency)
		}
	default:
		t.Errorf("no job sent to channel")
	}
}

func TestPostStartAsyncUpdateQuote_ExistingPending(t *testing.T) {
	supported := map[string]bool{"USD/EUR": true}
	jobChan := make(chan worker.QuoteJob, 1)
	mock := &MockQuoteService{
		GetLastQuoteFunc: func(currency string, status model.Status) (model.Quote, error) {
			return model.Quote{ID: "uuid-999"}, nil
		},
	}
	h := &Handler{SupportedCurrency: supported, Srv: mock, JobChan: jobChan}
	body := []byte(`{"currency":"USD/EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/quotes/update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.PostStartAsyncUpdateQuote(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	var out UpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if out.RequestId != "uuid-999" {
		t.Fatalf("expected request_id uuid-999, got %s", out.RequestId)
	}
	select {
	case <-jobChan:
		t.Errorf("should not push job when already pending")
	default:
		// ок
	}
}

func TestPostStartAsyncUpdateQuote_UnsupportedCurrency(t *testing.T) {
	supported := map[string]bool{"USD/EUR": true}
	jobChan := make(chan worker.QuoteJob, 1)
	mock := &MockQuoteService{}
	h := &Handler{SupportedCurrency: supported, Srv: mock, JobChan: jobChan}

	body := []byte(`{"currency":"GBP/USD"}`)
	req := httptest.NewRequest(http.MethodPost, "/quotes/update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.PostStartAsyncUpdateQuote(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostStartAsyncUpdateQuote_ServerErrorOnInsert(t *testing.T) {
	supported := map[string]bool{"USD/EUR": true}
	jobChan := make(chan worker.QuoteJob, 1)
	mock := &MockQuoteService{
		GetLastQuoteFunc: func(currency string, status model.Status) (model.Quote, error) {
			return model.Quote{}, sql.ErrNoRows
		},
		InsertPendingQuoteFunc: func(currency string) (string, error) {
			return "", errors.New("db error")
		},
	}
	h := &Handler{SupportedCurrency: supported, Srv: mock, JobChan: jobChan}
	body := []byte(`{"currency":"USD/EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/quotes/update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.PostStartAsyncUpdateQuote(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestGetQuoteByRequestId_Success(t *testing.T) {
	mock := &MockQuoteService{
		GetQuoteByIdFunc: func(id string) (model.Quote, error) {
			price := 10.0
			now := time.Now()
			return model.Quote{
				ID:        "uuid-1",
				Currency:  "USD/EUR",
				Price:     &price,
				UpdatedAt: &now,
				Status:    model.StatusDone,
			}, nil
		},
	}
	h := &Handler{Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/update/uuid-1", nil)
	w := httptest.NewRecorder()

	h.GetQuoteByRequestId(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var qr QuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if qr.Currency != "USD/EUR" {
		t.Errorf("wrong currency: %s", qr.Currency)
	}
}

func TestGetQuoteByRequestId_NotFound(t *testing.T) {
	mock := &MockQuoteService{
		GetQuoteByIdFunc: func(id string) (model.Quote, error) {
			return model.Quote{}, sql.ErrNoRows
		},
	}
	h := &Handler{Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/update/not-exist", nil)
	w := httptest.NewRecorder()

	h.GetQuoteByRequestId(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetQuoteByRequestId_Pending(t *testing.T) {
	mock := &MockQuoteService{
		GetQuoteByIdFunc: func(id string) (model.Quote, error) {
			return model.Quote{Status: model.StatusPending}, nil
		},
	}
	h := &Handler{Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/update/pending", nil)
	w := httptest.NewRecorder()

	h.GetQuoteByRequestId(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooEarly {
		t.Fatalf("expected 425, got %d", resp.StatusCode)
	}
}

func TestGetQuoteByRequestId_ServerError(t *testing.T) {
	mock := &MockQuoteService{
		GetQuoteByIdFunc: func(id string) (model.Quote, error) {
			return model.Quote{}, errors.New("db error")
		},
	}
	h := &Handler{Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/update/some", nil)
	w := httptest.NewRecorder()

	h.GetQuoteByRequestId(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestGetLastQuote_Success(t *testing.T) {
	mock := &MockQuoteService{
		GetLastQuoteFunc: func(currency string, status model.Status) (model.Quote, error) {
			price := 1.1
			now := time.Now()
			return model.Quote{
				ID:        "uuid-2",
				Currency:  "EUR/USD",
				Price:     &price,
				UpdatedAt: &now,
				Status:    model.StatusDone,
			}, nil
		},
	}
	supported := map[string]bool{"EUR/USD": true}
	h := &Handler{SupportedCurrency: supported, Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/last/EUR/USD", nil)
	w := httptest.NewRecorder()

	h.GetLastQuote(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var qr QuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if qr.Currency != "EUR/USD" {
		t.Errorf("wrong currency: %s", qr.Currency)
	}
}

func TestGetLastQuote_NotSupported(t *testing.T) {
	mock := &MockQuoteService{}
	supported := map[string]bool{"EUR/USD": true}
	h := &Handler{SupportedCurrency: supported, Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/last/GBP/USD", nil)
	w := httptest.NewRecorder()

	h.GetLastQuote(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetLastQuote_NotFound(t *testing.T) {
	mock := &MockQuoteService{
		GetLastQuoteFunc: func(currency string, status model.Status) (model.Quote, error) {
			return model.Quote{}, sql.ErrNoRows
		},
	}
	supported := map[string]bool{"EUR/USD": true}
	h := &Handler{SupportedCurrency: supported, Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/last/EUR/USD", nil)
	w := httptest.NewRecorder()

	h.GetLastQuote(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetLastQuote_ServerError(t *testing.T) {
	mock := &MockQuoteService{
		GetLastQuoteFunc: func(currency string, status model.Status) (model.Quote, error) {
			return model.Quote{}, errors.New("db error")
		},
	}
	supported := map[string]bool{"EUR/USD": true}
	h := &Handler{SupportedCurrency: supported, Srv: mock}
	req := httptest.NewRequest(http.MethodGet, "/quotes/last/EUR/USD", nil)
	w := httptest.NewRecorder()

	h.GetLastQuote(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}
