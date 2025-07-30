package main

import (
	"FinQuotesService/internal/api"
	"FinQuotesService/internal/db"
	"FinQuotesService/internal/service"
	"FinQuotesService/internal/tools"
	"FinQuotesService/internal/worker"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const workersCount = 10
const jobBufferSize = 32

func setupRoutes(h *api.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/quotes/update", h.PostStartAsyncUpdateQuote)
	mux.HandleFunc("/quotes/update/", h.GetQuoteByRequestId)
	mux.HandleFunc("/quotes/last/", h.GetLastQuote)
	return mux
}

func runServer() error {
	database := db.InitializeDb()
	defer database.Close()

	supportedCurrency, err := tools.LoadSupportedCurrencies("./supported_currency.json")
	if err != nil {
		return err
	}

	jobChan := make(chan worker.QuoteJob, jobBufferSize)
	srv := service.NewQuoteService(database)
	h := &api.Handler{
		SupportedCurrency: supportedCurrency,
		Srv:               srv,
		JobChan:           jobChan,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for i := 0; i < workersCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.StartWorker(jobChan, srv)
		}()
	}

	mux := setupRoutes(h)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Printf("Server listening on %s...", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received, stopping server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	close(jobChan)
	wg.Wait()

	log.Println("All workers done. Server stopped.")
	return nil
}

func main() {
	if err := runServer(); err != nil {
		log.Fatalf("Startup error: %v", err)
	}
}
