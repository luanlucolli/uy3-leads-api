package handlers

import "testing"

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
