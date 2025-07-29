package worker

import (
	"FinQuotesService/internal/model"
	"FinQuotesService/internal/service"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type QuoteJob struct {
	Id       string
	Currency string
}

type ratesResponse struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

func StartWorker(ctx context.Context, jobs <-chan QuoteJob, srv *service.QuoteService) {
	for {
		select {
		case <-ctx.Done():
			log.Println("[Worker] Worker stopped by context")
			// To-do: handle all possible pending quote requests Or cancel them
			return
		case job := <-jobs:
			log.Println("[Worker] Job processing started, job_id = " + job.Id)
			price, err := fetchExternalQuote(job.Currency)
			log.Println("[Worker] Job processing finished, job_id = " + job.Id)
			status := model.StatusDone
			if err != nil {
				status = model.StatusError
				log.Printf("[Worker] failed to fetch quote for %s: %v", job.Currency, err)
			}
			if err := srv.UpdateQuote(job.Id, price, status); err != nil {
				log.Printf("[Worker] db update error: %v", err)
			}
		}
	}
}

func fetchExternalQuote(currencyPair string) (float64, error) {
	// emulation of processing
	time.Sleep(30 * time.Second)
	split := strings.Split(currencyPair, "/")
	if len(split) != 2 {
		return 0, errors.New("bad currency pair")
	}
	base, target := split[0], split[1]
	url := fmt.Sprintf("https://api.vatcomply.com/rates?base=%s", base)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fetcher: http error: %v", resp.Status)
	}

	var r ratesResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}

	rate, ok := r.Rates[target]
	if !ok {
		return 0, fmt.Errorf("no rate found for %s/%s", base, target)
	}

	return rate, nil
}
