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

func TestBuildLeadWhereWithFromTo(t *testing.T) {
	filters := models.LeadFilters{
		From: "2026-04-01",
		To:   "2026-04-30",
	}

	where, args := buildLeadWhere(filters)

	if where != " WHERE received_at >= ? AND received_at <= ?" {
		t.Fatalf("buildLeadWhere where = %q", where)
	}
	if len(args) != 2 {
		t.Fatalf("buildLeadWhere args length = %d", len(args))
	}
	if args[0] != "2026-04-01 03:00:00" {
		t.Fatalf("buildLeadWhere from arg = %v", args[0])
	}
	if args[1] != "2026-05-01 02:59:59" {
		t.Fatalf("buildLeadWhere to arg = %v", args[1])
	}
}

func TestValidateExportFiltersBlocksAllWithoutDate(t *testing.T) {
	err := validateExportFilters(models.LeadFilters{Period: "all"})

	if err == nil {
		t.Fatal("validateExportFilters(all) expected error")
	}
}

func TestValidateExportFiltersLimitsCustomWindow(t *testing.T) {
	err := validateExportFilters(models.LeadFilters{
		From: "2026-01-01",
		To:   "2026-07-01",
	})

	if err == nil {
		t.Fatal("validateExportFilters(large custom window) expected error")
	}
}
