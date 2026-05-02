package handlers

import "time"

var brtLocation = loadBrazilLocation()

const (
	loginDBTimeout       = 10 * time.Second
	leadsSummaryTimeout  = 10 * time.Second
	webhookDBTimeout     = 10 * time.Second
	exportBatchDBTimeout = 30 * time.Second
)

func loadBrazilLocation() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err == nil {
		return loc
	}

	return time.FixedZone("BRT", -3*3600)
}
