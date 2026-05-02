package handlers

import (
	"strings"
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

func TestFormatDateBRKeepsDateOnlyWithoutTimezoneShift(t *testing.T) {
	raw := "1990-05-20"

	got := formatDateBR(raw, false)

	if got != "20/05/1990" {
		t.Fatalf("formatDateBR(%q, false) = %q", raw, got)
	}
}

func TestFormatDateBRKeepsRFC3339DateWithoutTimezoneShiftWhenTimeHidden(t *testing.T) {
	raw := "1990-05-20T00:00:00Z"

	got := formatDateBR(raw, false)

	if got != "20/05/1990" {
		t.Fatalf("formatDateBR(%q, false) = %q", raw, got)
	}
}

func TestFormatDateForAPIDateOnlyDoesNotShiftTimezone(t *testing.T) {
	raw := "2026-05-01"

	got := formatDateForAPI(raw)

	if got != "2026-05-01" {
		t.Fatalf("formatDateForAPI(%q) = %q", raw, got)
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

func TestSummaryDateRangeForFixedPeriods(t *testing.T) {
	now := time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation)

	from24h, to24h := summaryDateRange(models.LeadFilters{Period: "24h"}, now)
	if from24h != "2026-05-09" || to24h != "2026-05-10" {
		t.Fatalf("summaryDateRange(24h) = %q, %q", from24h, to24h)
	}

	from7d, to7d := summaryDateRange(models.LeadFilters{Period: "7d"}, now)
	if from7d != "2026-05-04" || to7d != "2026-05-10" {
		t.Fatalf("summaryDateRange(7d) = %q, %q", from7d, to7d)
	}

	from30d, to30d := summaryDateRange(models.LeadFilters{Period: "30d"}, now)
	if from30d != "2026-04-11" || to30d != "2026-05-10" {
		t.Fatalf("summaryDateRange(30d) = %q, %q", from30d, to30d)
	}

	from90d, to90d := summaryDateRange(models.LeadFilters{Period: "90d"}, now)
	if from90d != "2026-02-10" || to90d != "2026-05-10" {
		t.Fatalf("summaryDateRange(90d) = %q, %q", from90d, to90d)
	}
}

func TestBuildLeadWhereWithFromTo(t *testing.T) {
	filters := models.LeadFilters{
		From: "2026-04-01",
		To:   "2026-04-30",
	}

	where, args := buildLeadWhere(filters)

	if where != " WHERE received_at >= ? AND received_at < ?" {
		t.Fatalf("buildLeadWhere where = %q", where)
	}
	if len(args) != 2 {
		t.Fatalf("buildLeadWhere args length = %d", len(args))
	}
	if args[0] != "2026-04-01 03:00:00" {
		t.Fatalf("buildLeadWhere from arg = %v", args[0])
	}
	if args[1] != "2026-05-01 03:00:00" {
		t.Fatalf("buildLeadWhere endExclusive arg = %v", args[1])
	}
}

func TestBuildLeadWhereAtUsesYesterdayAndTodayFor24h(t *testing.T) {
	now := time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation)

	where, args := buildLeadWhereAt(models.LeadFilters{Period: "24h"}, now)

	if where != " WHERE received_at >= ? AND received_at < ?" {
		t.Fatalf("buildLeadWhereAt(24h) where = %q", where)
	}
	if len(args) != 2 {
		t.Fatalf("buildLeadWhereAt(24h) args length = %d", len(args))
	}
	if args[0] != "2026-05-09 03:00:00" {
		t.Fatalf("buildLeadWhereAt(24h) from arg = %v", args[0])
	}
	if args[1] != "2026-05-11 03:00:00" {
		t.Fatalf("buildLeadWhereAt(24h) endExclusive arg = %v", args[1])
	}
}

func TestLeadDateTimeRangeUsesExclusiveEndForCustomRange(t *testing.T) {
	startUTC, endExclusiveUTC, ok := leadDateTimeRange(models.LeadFilters{
		From: "2026-04-01",
		To:   "2026-04-30",
	}, time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation))

	if !ok {
		t.Fatal("leadDateTimeRange(custom) expected range")
	}
	if startUTC != "2026-04-01 03:00:00" {
		t.Fatalf("leadDateTimeRange(custom) startUTC = %q", startUTC)
	}
	if endExclusiveUTC != "2026-05-01 03:00:00" {
		t.Fatalf("leadDateTimeRange(custom) endExclusiveUTC = %q", endExclusiveUTC)
	}
}

func TestValidateExportFiltersBlocksAllWithoutDate(t *testing.T) {
	err := validateExportFilters(models.LeadFilters{Period: "all"})

	if err == nil {
		t.Fatal("validateExportFilters(all) expected error")
	}
}

func TestValidateExportFiltersAcceptsFixedPeriods(t *testing.T) {
	periods := []string{"24h", "7d", "30d", "90d"}

	for _, period := range periods {
		if err := validateExportFilters(models.LeadFilters{Period: period}); err != nil {
			t.Fatalf("validateExportFilters(%q) unexpected error = %v", period, err)
		}
	}
}

func TestValidateExportFiltersRequireCompleteCustomRange(t *testing.T) {
	tests := []models.LeadFilters{
		{Period: "custom", From: "2026-01-01"},
		{Period: "custom", To: "2026-01-31"},
	}

	for _, filters := range tests {
		if err := validateExportFilters(filters); err == nil {
			t.Fatalf("validateExportFilters(%+v) expected error", filters)
		}
	}
}

func TestValidateExportFiltersRejectsToBeforeFrom(t *testing.T) {
	err := validateExportFilters(models.LeadFilters{
		From: "2026-02-01",
		To:   "2026-01-31",
	})

	if err == nil {
		t.Fatal("validateExportFilters(to before from) expected error")
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

func TestBuildLastLeadQueryAtUsesFilterRange(t *testing.T) {
	now := time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation)

	query, args := buildLastLeadQueryAt(models.LeadFilters{Period: "7d"}, now)

	if !strings.Contains(query, "WHERE received_at >= ? AND received_at < ?") {
		t.Fatalf("buildLastLeadQueryAt query = %q", query)
	}
	if !strings.Contains(query, "ORDER BY received_at DESC, id DESC LIMIT 1") {
		t.Fatalf("buildLastLeadQueryAt order = %q", query)
	}
	if len(args) != 2 {
		t.Fatalf("buildLastLeadQueryAt args length = %d", len(args))
	}
	if args[0] != "2026-05-04 03:00:00" || args[1] != "2026-05-11 03:00:00" {
		t.Fatalf("buildLastLeadQueryAt args = %#v", args)
	}
}

func TestBuildLastLeadQueryAtForAllUsesGlobalLatest(t *testing.T) {
	query, args := buildLastLeadQueryAt(models.LeadFilters{Period: "all"}, time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation))

	if query != "SELECT received_at FROM leads ORDER BY received_at DESC, id DESC LIMIT 1" {
		t.Fatalf("buildLastLeadQueryAt(all) query = %q", query)
	}
	if len(args) != 0 {
		t.Fatalf("buildLastLeadQueryAt(all) args = %#v", args)
	}
}
