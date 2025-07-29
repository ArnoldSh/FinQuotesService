package api

type ErrorResponse struct {
	Message ServiceError `json:"error_message"`
}

type ServiceError string

const (
	QuoteNotFound           ServiceError = "Quote or request not found"
	ServerInternalError     ServiceError = "Server internal error"
	QuoteOnPending          ServiceError = "Quote on pending"
	UnsupportedCurrencyPair ServiceError = "Unsupported currency pair"
)
