package models

import (
	"fmt"
	"net/http"
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
	ID                          int64   `json:"id"`
	CPF                         string  `json:"cpf"`
	NomeTrabalhador             string  `json:"nome_trabalhador"`
	Status                      string  `json:"status"`
	ElegivelEmprestimo          string  `json:"elegivel_emprestimo"`
	ValorLiberado               float64 `json:"valor_liberado"`
	MargemDisponivel            float64 `json:"margem_disponivel"`
	NumeroParcelas              int64   `json:"numero_parcelas"`
	DataHoraValidadeSolicitacao string  `json:"data_hora_validade_solicitacao"`
	DataNascimento              string  `json:"data_nascimento"`
	DataAdmissao                string  `json:"data_admissao"`
	IsMEI                       string  `json:"is_mei"`
	IsJudicialRecovery          string  `json:"is_judicial_recovery"`
	PEPCodigo                   string  `json:"pep_codigo"`
	ActiveFGTSDebts             string  `json:"active_fgts_debts"`
	TypeWebhook                 string  `json:"type_webhook"`
	Exportado                   int64   `json:"exportado"`
	ReceivedAt                  string  `json:"received_at"`
}

type SummaryResponse struct {
	Total      int64  `json:"total"`
	LastLeadAt string `json:"last_lead_at,omitempty"`
}

type LeadFilters struct {
	Period string
	From   string
	To     string
}

func ParseLeadFilters(r *http.Request) (LeadFilters, error) {
	q := r.URL.Query()
	f := LeadFilters{
		Period: strings.ToLower(strings.TrimSpace(q.Get("period"))),
		From:   strings.TrimSpace(q.Get("from")),
		To:     strings.TrimSpace(q.Get("to")),
	}
	if f.Period == "" {
		f.Period = "all"
	}

	if !validPeriod(f.Period) {
		return LeadFilters{}, fmt.Errorf("period invalido")
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

func validPeriod(period string) bool {
	switch period {
	case "all", "24h", "7d", "30d", "90d", "custom":
		return true
	default:
		return false
	}
}
