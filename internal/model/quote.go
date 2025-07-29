package model

import "time"

type Quote struct {
	ID        string     `db:"id"`
	Currency  string     `db:"currency"`
	Price     *float64   `db:"price"`
	UpdatedAt *time.Time `db:"updated_at"`
	Status    Status     `db:"status"`
}

type Status string

const (
	StatusPending Status = "pending"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)
