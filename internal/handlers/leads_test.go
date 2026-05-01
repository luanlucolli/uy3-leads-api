package handlers

import (
	"testing"
	"time"

	"github.com/luanlucolli/uy3-leads-api/internal/models"
)

func TestFormatDateForAPIConvertsUTCToBRT(t *testing.T) {
	raw := "2026-05-01 00:46:51"

	got := formatDateForAPI(raw)

	if got != "2026-04-30 21:46:51" {
		t.Fatalf("formatDateForAPI(%q) = %q", raw, got)
	}
}

func TestFormatDateBRConvertsUTCToBRTOnce(t *testing.T) {
	raw := "2026-05-01 00:46:51"

	got := formatDateBR(raw, true)

	if got != "30/04/2026 21:46:51" {
		t.Fatalf("formatDateBR(%q, true) = %q", raw, got)
	}
}

func TestSummaryDateRangeForCustomDates(t *testing.T) {
	filters := models.LeadFilters{
		From: "2026-04-01",
		To:   "2026-04-30",
	}

	from, to := summaryDateRange(filters, time.Date(2026, 5, 1, 12, 0, 0, 0, brtLocation))

	if from != "2026-04-01" || to != "2026-04-30" {
		t.Fatalf("summaryDateRange(custom) = %q, %q", from, to)
	}
}

func TestSummaryDateRangeForRollingPeriods(t *testing.T) {
	now := time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation)

	from24h, to24h := summaryDateRange(models.LeadFilters{Period: "24h"}, now)
	if from24h != "2026-05-09" || to24h != "2026-05-10" {
		t.Fatalf("summaryDateRange(24h) = %q, %q", from24h, to24h)
	}

	from7d, to7d := summaryDateRange(models.LeadFilters{Period: "7d"}, now)
	if from7d != "2026-05-03" || to7d != "2026-05-10" {
		t.Fatalf("summaryDateRange(7d) = %q, %q", from7d, to7d)
	}
}
