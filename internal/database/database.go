package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tursodatabase/libsql-client-go/libsql"
)

func Open(databaseURL string) (*sql.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL nao configurado")
	}

	cleanURL, token, err := normalizeDatabaseURL(databaseURL)
	if err != nil {
		return nil, err
	}
	if token == "" {
		token = firstNonEmpty(
			os.Getenv("TURSO_AUTH_TOKEN"),
			os.Getenv("DATABASE_AUTH_TOKEN"),
			os.Getenv("LIBSQL_AUTH_TOKEN"),
		)
	}

	var opts []libsql.Option
	if token != "" {
		opts = append(opts, libsql.WithAuthToken(token))
	}

	connector, err := libsql.NewConnector(cleanURL, opts...)
	if err != nil {
		return nil, err
	}

	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", 3))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 1))
	db.SetConnMaxIdleTime(time.Duration(envInt("DB_CONN_MAX_IDLE_SECONDS", 30)) * time.Second)
	db.SetConnMaxLifetime(time.Duration(envInt("DB_CONN_MAX_LIFETIME_SECONDS", 600)) * time.Second)

	return db, nil
}

func normalizeDatabaseURL(databaseURL string) (string, string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", "", fmt.Errorf("DATABASE_URL invalida: %w", err)
	}

	query := u.Query()
	token := firstNonEmpty(query.Get("auth_token"), query.Get("authToken"), query.Get("jwt"))
	query.Del("auth_token")
	query.Del("authToken")
	query.Del("jwt")
	u.RawQuery = query.Encode()

	return u.String(), token, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
