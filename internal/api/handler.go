package api

import (
	"FinQuotesService/internal/model"
	"FinQuotesService/internal/service"
	"FinQuotesService/internal/worker"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type SupportedCurrency map[string]bool

type Handler struct {
	SupportedCurrency map[string]bool
	Srv               service.QuoteServiceInterface
	JobChan           chan worker.QuoteJob
}

type UpdateRequest struct {
	Currency string `json:"currency"`
}

type UpdateResponse struct {
	RequestId string `json:"request_id"`
}

type QuoteResponse struct {
	Currency  string     `json:"currency"`
	Price     *float64   `json:"price,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func (h *Handler) PostStartAsyncUpdateQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		httpMethodNotAllowed(w, "POST")
		return
	}
	var req UpdateRequest
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &req); err != nil || !h.SupportedCurrency[req.Currency] {
		unsupportedCurrencyPair(w)
		return
	}
	var quoteId string
	quote, err := h.Srv.GetLastQuote(req.Currency, model.StatusPending)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			quoteId, err = h.Srv.InsertPendingQuote(req.Currency)
			if err != nil {
				serverInternalError(w)
				return
			}
			h.JobChan <- worker.QuoteJob{Id: quoteId, Currency: req.Currency}
			log.Println("[Handler] Job pushed to queue, job_id = " + quoteId)
		} else {
			serverInternalError(w)
			return
		}
	} else {
		quoteId = quote.ID
		log.Println("[Handler] Existing pending job found, job_id = " + quoteId)
	}

	resp := UpdateResponse{RequestId: quoteId}
	successResponse(w, resp)
}

func (h *Handler) GetQuoteByRequestId(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpMethodNotAllowed(w, "GET")
		return
	}
	requestId := strings.TrimPrefix(r.URL.Path, "/quotes/update/")
	q, err := h.Srv.GetQuoteById(requestId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			quoteNotFoundError(w)
		} else {
			serverInternalError(w)
		}
		return
	}
	if q.Status == model.StatusPending {
		quoteOnPendingError(w)
		return
	}
	resp := mapToQuoteResponse(q)
	successResponse(w, resp)
}

func (h *Handler) GetLastQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpMethodNotAllowed(w, "GET")
		return
	}
	currency := strings.TrimPrefix(r.URL.Path, "/quotes/last/")
	if !h.SupportedCurrency[currency] {
		unsupportedCurrencyPair(w)
		return
	}
	q, err := h.Srv.GetLastQuote(currency, model.StatusDone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			quoteNotFoundError(w)
		} else {
			serverInternalError(w)
		}
		return
	}
	resp := mapToQuoteResponse(q)
	successResponse(w, resp)
}

func httpMethodNotAllowed(w http.ResponseWriter, targetMethod string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	err := json.NewEncoder(w).Encode(targetMethod + " only")
	if err != nil {
		return
	}
}

func unsupportedCurrencyPair(w http.ResponseWriter) {
	errorResponse(w, http.StatusBadRequest, UnsupportedCurrencyPair)
}

func quoteNotFoundError(w http.ResponseWriter) {
	errorResponse(w, http.StatusNotFound, QuoteNotFound)
}

func serverInternalError(w http.ResponseWriter) {
	errorResponse(w, http.StatusInternalServerError, ServerInternalError)
}

func quoteOnPendingError(w http.ResponseWriter) {
	errorResponse(w, http.StatusTooEarly, QuoteOnPending)
}

func successResponse(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(payload)
	if err != nil {
		return
	}
}

func errorResponse(w http.ResponseWriter, code int, svcErrorMsg ServiceError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Message: svcErrorMsg,
	})
	if err != nil {
		return
	}
}

func mapToQuoteResponse(q model.Quote) QuoteResponse {
	return QuoteResponse{
		Currency:  q.Currency,
		Price:     q.Price,
		UpdatedAt: q.UpdatedAt,
	}
}
