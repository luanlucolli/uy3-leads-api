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

func TestParseLeadFiltersAcceptsDateTimeLocalMinutes(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?period=custom&from=2026-05-09T10:30&to=2026-05-09T12:45", nil)

	filters, err := ParseLeadFilters(request)

	if err != nil {
		t.Fatalf("ParseLeadFilters datetime-local error = %v", err)
	}
	if filters.From != "2026-05-09T10:30" || filters.To != "2026-05-09T12:45" {
		t.Fatalf("ParseLeadFilters datetime-local = %+v", filters)
	}
}

func TestParseLeadFiltersAcceptsSpaceSeparatedDateTime(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?period=custom&from=2026-05-09%2010:30&to=2026-05-09%2012:45", nil)

	filters, err := ParseLeadFilters(request)

	if err != nil {
		t.Fatalf("ParseLeadFilters space datetime error = %v", err)
	}
	if filters.From != "2026-05-09 10:30" || filters.To != "2026-05-09 12:45" {
		t.Fatalf("ParseLeadFilters space datetime = %+v", filters)
	}
}

func TestParseLeadFiltersAcceptsDateTimeLocalSeconds(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?period=custom&from=2026-05-09T10:30:15&to=2026-05-09T12:45:30", nil)

	filters, err := ParseLeadFilters(request)

	if err != nil {
		t.Fatalf("ParseLeadFilters datetime seconds error = %v", err)
	}
	if filters.From != "2026-05-09T10:30:15" || filters.To != "2026-05-09T12:45:30" {
		t.Fatalf("ParseLeadFilters datetime seconds = %+v", filters)
	}
}

func TestParseLeadFiltersAcceptsFutureTo(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?period=custom&from=2026-05-09T10:30&to=2099-05-09T12:45", nil)

	if _, err := ParseLeadFilters(request); err != nil {
		t.Fatalf("ParseLeadFilters future to unexpected error = %v", err)
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

func TestParseLeadFiltersRejectsToBeforeFrom(t *testing.T) {
	request := httptest.NewRequest("GET", "/leads?period=custom&from=2026-05-09T12:45&to=2026-05-09T10:30", nil)

	if _, err := ParseLeadFilters(request); err == nil {
		t.Fatal("ParseLeadFilters to before from expected error")
	}
}
