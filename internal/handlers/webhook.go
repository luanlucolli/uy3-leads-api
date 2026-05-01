package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type WebhookHandler struct {
	db *sql.DB
}

func NewWebhookHandler(db *sql.DB) *WebhookHandler {
	return &WebhookHandler{db: db}
}

func (h *WebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 512<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "payload muito grande ou invalido")
		return
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		writeError(w, http.StatusBadRequest, "payload vazio")
		return
	}

	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	lead := webhookLead{
		CPF:                         textField(payload, "cpf", "CPF"),
		NomeTrabalhador:             textField(payload, "nome_trabalhador", "nomeTrabalhador", "NomeTrabalhador", "nome", "name"),
		Status:                      textField(payload, "status", "Status"),
		ElegivelEmprestimo:          textField(payload, "elegivel_emprestimo", "elegivelEmprestimo", "elegivelParaEmprestimo", "eligibleLoan"),
		ValorLiberado:               floatField(payload, "valor_liberado", "valorLiberado", "releasedAmount"),
		MargemDisponivel:            floatField(payload, "margem_disponivel", "margemDisponivel", "availableMargin"),
		NumeroParcelas:              intField(payload, "numero_parcelas", "numeroParcelas", "numberOfInstallments"),
		DataHoraValidadeSolicitacao: textField(payload, "data_hora_validade_solicitacao", "dataHoraValidadeSolicitacao", "dataHoraValidadeDaSolicitacao", "validadeSolicitacao", "requestExpirationDate"),
		DataNascimento:              textField(payload, "data_nascimento", "dataNascimento", "birthDate"),
		DataAdmissao:                textField(payload, "data_admissao", "dataAdmissao", "admissionDate"),
		IsMEI:                       textField(payload, "is_mei", "isMei", "isMEI"),
		IsJudicialRecovery:          textField(payload, "is_judicial_recovery", "isJudicialRecovery"),
		PEPCodigo:                   textField(payload, "pep_codigo", "pepCodigo", "pepCode"),
		ActiveFGTSDebts:             textField(payload, "active_fgts_debts", "activeFgtsDebts", "activeFGTSDebts"),
		TypeWebhook:                 textField(payload, "typeWebook", "typeWebhook", "type_webhook"),
		RawPayload:                  string(raw),
	}

	result, err := h.db.ExecContext(r.Context(), `
		INSERT INTO leads (
			cpf, nome_trabalhador, status, elegivel_emprestimo,
			valor_liberado, margem_disponivel, numero_parcelas,
			data_hora_validade_solicitacao, data_nascimento, data_admissao,
			is_mei, is_judicial_recovery, pep_codigo, active_fgts_debts,
			type_webhook, raw_payload
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		lead.CPF, lead.NomeTrabalhador, lead.Status, lead.ElegivelEmprestimo,
		lead.ValorLiberado, lead.MargemDisponivel, lead.NumeroParcelas,
		lead.DataHoraValidadeSolicitacao, lead.DataNascimento, lead.DataAdmissao,
		lead.IsMEI, lead.IsJudicialRecovery, lead.PEPCodigo, lead.ActiveFGTSDebts,
		lead.TypeWebhook, lead.RawPayload,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar lead")
		return
	}

	id, _ := result.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": "received"})
}

type webhookLead struct {
	CPF                         string
	NomeTrabalhador             string
	Status                      string
	ElegivelEmprestimo          string
	ValorLiberado               float64
	MargemDisponivel            float64
	NumeroParcelas              int64
	DataHoraValidadeSolicitacao string
	DataNascimento              string
	DataAdmissao                string
	IsMEI                       string
	IsJudicialRecovery          string
	PEPCodigo                   string
	ActiveFGTSDebts             string
	TypeWebhook                 string
	RawPayload                  string
}

func textField(payload map[string]any, keys ...string) string {
	value, ok := lookup(payload, keys...)
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func floatField(payload map[string]any, keys ...string) float64 {
	value, ok := lookup(payload, keys...)
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case json.Number:
		number, _ := v.Float64()
		return number
	case float64:
		return v
	case string:
		number, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(v), ",", "."), 64)
		return number
	default:
		return 0
	}
}

func intField(payload map[string]any, keys ...string) int64 {
	value, ok := lookup(payload, keys...)
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case json.Number:
		number, _ := v.Int64()
		return number
	case float64:
		return int64(v)
	case string:
		number, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return number
	default:
		return 0
	}
}

func lookup(payload map[string]any, keys ...string) (any, bool) {
	if value, ok := lookupCurrentLevel(payload, keys...); ok {
		return value, true
	}

	for _, container := range []string{"data", "payload", "lead"} {
		value, ok := lookupCurrentLevel(payload, container)
		if !ok {
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			if nestedValue, ok := lookupCurrentLevel(nested, keys...); ok {
				return nestedValue, true
			}
		}
	}

	return nil, false
}

func lookupCurrentLevel(payload map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value, true
		}
	}

	lowerKeys := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		lowerKeys[strings.ToLower(key)] = struct{}{}
	}
	for key, value := range payload {
		if _, ok := lowerKeys[strings.ToLower(key)]; ok {
			return value, true
		}
	}

	return nil, false
}
