package handlers

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/luanlucolli/uy3-leads-api/internal/auth"
	"github.com/luanlucolli/uy3-leads-api/internal/middleware"
)

const (
	loginBodyLimit        = 8 << 10
	loginRateLimitWindow  = 5 * time.Minute
	loginRateLimitTTL     = 15 * time.Minute
	loginRateLimitMaxHits = 10
	loginRateLimitMaxKeys = 512
)

type AuthHandler struct {
	authService  *auth.Service
	loginLimiter *loginRateLimiter
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		loginLimiter: newLoginRateLimiter(loginRateLimitMaxHits, loginRateLimitWindow, loginRateLimitTTL),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !h.loginLimiter.allow(ip, time.Now()) {
		writeError(w, http.StatusTooManyRequests, "muitas tentativas de login, tente novamente em alguns minutos")
		return
	}

	var req loginRequest
	r.Body = http.MaxBytesReader(w, r.Body, loginBodyLimit)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email e password sao obrigatorios")
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err.Error() == "credenciais invalidas" {
			writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		} else {
			// Erros de banco de dados, timeout, etc.
			writeError(w, http.StatusInternalServerError, "serviço indisponível no momento")
		}
		return
	}

	h.loginLimiter.reset(ip)
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.authService.CurrentUser(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})
}

type loginRateLimiter struct {
	mu       sync.Mutex
	entries  map[string]loginRateEntry
	maxHits  int
	window   time.Duration
	entryTTL time.Duration
}

type loginRateEntry struct {
	count       int
	windowFrom  time.Time
	lastTouched time.Time
}

func newLoginRateLimiter(maxHits int, window, entryTTL time.Duration) *loginRateLimiter {
	return &loginRateLimiter{
		entries:  make(map[string]loginRateEntry),
		maxHits:  maxHits,
		window:   window,
		entryTTL: entryTTL,
	}
}

func (l *loginRateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now)
	if _, ok := l.entries[key]; !ok && len(l.entries) >= loginRateLimitMaxKeys {
		l.dropOldest()
	}
	entry := l.entries[key]
	if entry.windowFrom.IsZero() || now.Sub(entry.windowFrom) > l.window {
		entry = loginRateEntry{windowFrom: now}
	}
	entry.count++
	entry.lastTouched = now
	l.entries[key] = entry

	return entry.count <= l.maxHits
}

func (l *loginRateLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *loginRateLimiter) cleanup(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.lastTouched) > l.entryTTL {
			delete(l.entries, key)
		}
	}
}

func (l *loginRateLimiter) dropOldest() {
	var oldestKey string
	var oldestTime time.Time
	for key, entry := range l.entries {
		if oldestKey == "" || entry.lastTouched.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastTouched
		}
	}
	if oldestKey != "" {
		delete(l.entries, oldestKey)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
