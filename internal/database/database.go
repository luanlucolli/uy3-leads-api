package database

import (
	"database/sql"
	"fmt"
	"net/url"
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

	var opts []libsql.Option
	if token != "" {
		opts = append(opts, libsql.WithAuthToken(token))
	}

	connector, err := libsql.NewConnector(cleanURL, opts...)
	if err != nil {
		return nil, err
	}

	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(30 * time.Second)
	db.SetConnMaxLifetime(10 * time.Minute)

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
