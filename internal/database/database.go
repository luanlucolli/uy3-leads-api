package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tursodatabase/libsql-client-go/libsql"
)

// Operational note: pure date filters on leads list/export benefit from a
// dedicated index on received_at.
// Recommended migration:
// CREATE INDEX IF NOT EXISTS idx_leads_received_at ON leads(received_at);
// The composite index (exportado, received_at) is less efficient when the
// query predicate only constrains received_at.
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
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)

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
