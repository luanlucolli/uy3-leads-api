package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

var brtLocation = loadBrazilLocation()

const (
	loginDBTimeout      = 45 * time.Second
	leadsSummaryTimeout = 45 * time.Second
	webhookDBTimeout    = 45 * time.Second
	exportDBTimeout     = 90 * time.Second
)

func isContextDeadlineOrCancel(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func logHandlerError(r *http.Request, operation string, err error) {
	requestID := chimiddleware.GetReqID(r.Context())
	log.Printf("request_id=%s operation=%s error=%v", requestID, operation, err)
}

func loadBrazilLocation() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err == nil {
		return loc
	}

	return time.FixedZone("BRT", -3*3600)
}
