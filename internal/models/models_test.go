package models

import (
	"net/http/httptest"
	"testing"
)

func TestParseLeadFiltersDefaults(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads", nil)

	filters, err := ParseLeadFilters(request)

	if err != nil {
		t.Fatalf("ParseLeadFilters default error = %v", err)
	}
	if filters.Period != "all" {
		t.Fatalf("ParseLeadFilters default period = %q", filters.Period)
	}
}

func TestParseLeadFiltersWithCustomDates(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?period=custom&from=2026-04-01&to=2026-04-30", nil)

	filters, err := ParseLeadFilters(request)

	if err != nil {
		t.Fatalf("ParseLeadFilters custom error = %v", err)
	}
	if filters.Period != "custom" || filters.From != "2026-04-01" || filters.To != "2026-04-30" {
		t.Fatalf("ParseLeadFilters custom = %+v", filters)
	}
}

func TestParseLeadFiltersRejectsInvalidPeriod(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?period=forever", nil)

	if _, err := ParseLeadFilters(request); err == nil {
		t.Fatal("ParseLeadFilters invalid period expected error")
	}
}

func TestParseLeadFiltersRejectsInvalidDate(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?from=04-01-2026", nil)

	if _, err := ParseLeadFilters(request); err == nil {
		t.Fatal("ParseLeadFilters invalid date expected error")
	}
}
