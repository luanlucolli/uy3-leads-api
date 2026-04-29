package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/luanlucolli/uy3-leads-api/internal/models"
)

func Load() (models.Config, error) {
	_ = loadDotenv()

	cfg := models.Config{
		Port:             valueOrDefault(os.Getenv("PORT"), "8080"),
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		Uy3WebhookSecret: strings.TrimSpace(os.Getenv("UY3_WEBHOOK_SECRET")),
		JWTSecret:        strings.TrimSpace(os.Getenv("JWT_SECRET")),
	}

	if cfg.DatabaseURL == "" {
		return models.Config{}, fmt.Errorf("DATABASE_URL nao configurado")
	}
	if cfg.Uy3WebhookSecret == "" {
		return models.Config{}, fmt.Errorf("UY3_WEBHOOK_SECRET nao configurado")
	}
	if cfg.JWTSecret == "" {
		return models.Config{}, fmt.Errorf("JWT_SECRET nao configurado")
	}

	return cfg, nil
}

func loadDotenv() error {
	if err := godotenv.Load(); err == nil {
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	for {
		envPath := filepath.Join(wd, ".env")
		if _, statErr := os.Stat(envPath); statErr == nil {
			return godotenv.Load(envPath)
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return nil
}

func valueOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
