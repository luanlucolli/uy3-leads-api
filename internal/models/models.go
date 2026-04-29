package models

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	DatabaseURL      string
	Uy3WebhookSecret string
	JWTSecret        string
}

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}

type Lead struct {
	ID                         int64   `json:"id"`
	CPF                        string  `json:"cpf"`
	NomeTrabalhador            string  `json:"nome_trabalhador"`
	Status                     string  `json:"status"`
	ElegivelEmprestimo         string  `json:"elegivel_emprestimo"`
	ValorLiberado              float64 `json:"valor_liberado"`
	MargemDisponivel           float64 `json:"margem_disponivel"`
	NumeroParcelas             int64   `json:"numero_parcelas"`
	DataHoraValidadeSolicitacao string  `json:"data_hora_validade_solicitacao"`
	DataNascimento             string  `json:"data_nascimento"`
	DataAdmissao               string  `json:"data_admissao"`
	IsMEI                      string  `json:"is_mei"`
	IsJudicialRecovery         string  `json:"is_judicial_recovery"`
	PEPCodigo                  string  `json:"pep_codigo"`
	ActiveFGTSDebts            string  `json:"active_fgts_debts"`
	TypeWebhook                string  `json:"type_webhook"`
	RawPayload                 string  `json:"raw_payload,omitempty"`
	Exportado                  int64   `json:"exportado"`
	ReceivedAt                 string  `json:"received_at"`
}

type Pagination struct {
	Items       []Lead `json:"items"`
	Total       int64  `json:"total"`
	CurrentPage int    `json:"current_page"`
	PerPage     int    `json:"per_page"`
	TotalPages  int    `json:"total_pages"`
	HasNext     bool   `json:"has_next"`
	HasPrevious bool   `json:"has_previous"`
}

type LeadFilters struct {
	Page      int
	PerPage   int
	Period    string
	From      string
	To        string
	Sort      string
	Direction string
}

func ParseLeadFilters(r *http.Request) (LeadFilters, error) {
	q := r.URL.Query()
	f := LeadFilters{
		Page:      parsePositiveInt(q.Get("page"), 1),
		PerPage:   parsePositiveInt(q.Get("per_page"), 20),
		Period:    strings.ToLower(strings.TrimSpace(q.Get("period"))),
		From:      strings.TrimSpace(q.Get("from")),
		To:        strings.TrimSpace(q.Get("to")),
		Sort:      strings.ToLower(strings.TrimSpace(q.Get("sort"))),
		Direction: strings.ToLower(strings.TrimSpace(q.Get("direction"))),
	}

	if f.PerPage > 100 {
		f.PerPage = 100
	}
	if f.Period == "" {
		f.Period = "all"
	}
	if f.Sort == "" {
		f.Sort = "received_at"
	}
	if f.Direction == "" {
		f.Direction = "desc"
	}

	if !validPeriod(f.Period) {
		return LeadFilters{}, fmt.Errorf("period invalido")
	}
	if f.Sort != "received_at" && f.Sort != "id" {
		return LeadFilters{}, fmt.Errorf("sort invalido")
	}
	if f.Direction != "asc" && f.Direction != "desc" {
		return LeadFilters{}, fmt.Errorf("direction invalida")
	}
	if f.From != "" {
		if _, err := time.Parse("2006-01-02", f.From); err != nil {
			return LeadFilters{}, fmt.Errorf("from deve estar no formato YYYY-MM-DD")
		}
	}
	if f.To != "" {
		if _, err := time.Parse("2006-01-02", f.To); err != nil {
			return LeadFilters{}, fmt.Errorf("to deve estar no formato YYYY-MM-DD")
		}
	}

	return f, nil
}

func (f LeadFilters) Offset() int {
	return (f.Page - 1) * f.PerPage
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func validPeriod(period string) bool {
	switch period {
	case "all", "24h", "7d", "30d", "90d":
		return true
	default:
		return false
	}
}
