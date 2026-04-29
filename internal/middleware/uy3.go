package middleware

import (
	"crypto/hmac"
	"encoding/json"
	"net/http"
	"strings"
)

func VerifyUy3Webhook(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" || !secretMatches(extractUy3Secret(r), secret) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractUy3Secret(r *http.Request) string {
	headers := []string{
		"X-UY3-Secret-Key",
		"X-Secret-Key",
		"Secret-Key",
	}
	for _, header := range headers {
		if value := strings.TrimSpace(r.Header.Get(header)); value != "" {
			return value
		}
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(authHeader)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}

	return ""
}

func secretMatches(got, want string) bool {
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if got == "" || want == "" {
		return false
	}
	return hmac.Equal([]byte(got), []byte(want))
}
