package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/luanlucolli/uy3-leads-api/internal/models"
)

type LeadsHandler struct {
	db *sql.DB
}

func NewLeadsHandler(db *sql.DB) *LeadsHandler {
	return &LeadsHandler{db: db}
}

func (h *LeadsHandler) List(w http.ResponseWriter, r *http.Request) {
	filters, err := models.ParseLeadFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	where, args := buildLeadWhere(filters)
	var total int64
	if err := h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM leads"+where, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao contar leads")
		return
	}

	query := fmt.Sprintf(`
		SELECT
			id, cpf, nome_trabalhador, status, elegivel_emprestimo,
			valor_liberado, margem_disponivel, numero_parcelas,
			data_hora_validade_solicitacao, data_nascimento, data_admissao,
			is_mei, is_judicial_recovery, pep_codigo, active_fgts_debts,
			type_webhook, raw_payload, exportado, received_at
		FROM leads
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, where, filters.Sort, filters.Direction)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, filters.PerPage, filters.Offset())

	rows, err := h.db.QueryContext(r.Context(), query, listArgs...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar leads")
		return
	}
	defer rows.Close()

	items := make([]models.Lead, 0, filters.PerPage)
	for rows.Next() {
		lead, err := scanLead(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "erro ao ler lead")
			return
		}
		items = append(items, lead)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar leads")
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(filters.PerPage) - 1) / int64(filters.PerPage))
	}

	writeJSON(w, http.StatusOK, models.Pagination{
		Items:       items,
		Total:       total,
		CurrentPage: filters.Page,
		PerPage:     filters.PerPage,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	})
}

func (h *LeadsHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	filters, err := models.ParseLeadFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	where, args := buildLeadWhere(filters)
	query := fmt.Sprintf(`
		SELECT
			id, cpf, nome_trabalhador, status, elegivel_emprestimo,
			valor_liberado, margem_disponivel, numero_parcelas,
			data_hora_validade_solicitacao, data_nascimento, data_admissao,
			is_mei, is_judicial_recovery, pep_codigo, active_fgts_debts,
			type_webhook, raw_payload, exportado, received_at
		FROM leads
		%s
		ORDER BY %s %s
		LIMIT 10000
	`, where, filters.Sort, filters.Direction)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao exportar leads")
		return
	}
	defer rows.Close()

	filename := "leads_" + time.Now().Format("20060102_150405") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}

	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = ';'
	defer csvWriter.Flush()

	if err := csvWriter.Write(csvHeaders()); err != nil {
		return
	}

	flusher, canFlush := w.(http.Flusher)
	rowsProcessed := 0
	for rows.Next() {
		lead, err := scanLead(rows)
		if err != nil {
			return
		}
		if err := csvWriter.Write([]string{
			lead.CPF,
			lead.NomeTrabalhador,
			lead.Status,
			booleanLabel(lead.ElegivelEmprestimo),
			formatDecimalBR(lead.ValorLiberado),
			formatDecimalBR(lead.MargemDisponivel),
			strconv.FormatInt(lead.NumeroParcelas, 10),
			formatDateBR(lead.ReceivedAt, true),
			formatDateBR(lead.DataHoraValidadeSolicitacao, true),
			formatDateBR(lead.DataNascimento, false),
			formatDateBR(lead.DataAdmissao, false),
			strconv.FormatInt(lead.ID, 10),
			booleanLabel(lead.IsMEI),
			booleanLabel(lead.IsJudicialRecovery),
			booleanLabel(lead.PEPCodigo),
			booleanLabel(lead.ActiveFGTSDebts),
		}); err != nil {
			return
		}
		rowsProcessed++
		if rowsProcessed%500 == 0 {
			csvWriter.Flush()
			if err := csvWriter.Error(); err != nil {
				return
			}
			if canFlush {
				flusher.Flush()
			}
		}
	}
}

func csvHeaders() []string {
	return []string{
		"CPF",
		"Nome do trabalhador",
		"Status",
		"Elegivel para emprestimo",
		"Valor liberado",
		"Margem disponivel",
		"Numero de parcelas",
		"Recebido em",
		"Validade da solicitacao",
		"Data de nascimento",
		"Data de admissao",
		"ID do registro",
		"É MEI",
		"Em recuperacao judicial",
		"Pessoa exposta politicamente",
		"Dividas ativas FGTS",
	}
}

type leadScanner interface {
	Scan(dest ...any) error
}

func scanLead(scanner leadScanner) (models.Lead, error) {
	var lead models.Lead
	var cpf, nome, status, elegivel, validade, nascimento, admissao sql.NullString
	var isMEI, judicial, pep, fgts, typeWebhook, rawPayload, receivedAt sql.NullString
	var valor, margem sql.NullFloat64
	var parcelas, exportado sql.NullInt64

	err := scanner.Scan(
		&lead.ID,
		&cpf,
		&nome,
		&status,
		&elegivel,
		&valor,
		&margem,
		&parcelas,
		&validade,
		&nascimento,
		&admissao,
		&isMEI,
		&judicial,
		&pep,
		&fgts,
		&typeWebhook,
		&rawPayload,
		&exportado,
		&receivedAt,
	)
	if err != nil {
		return models.Lead{}, err
	}

	lead.CPF = nullString(cpf)
	lead.NomeTrabalhador = nullString(nome)
	lead.Status = nullString(status)
	lead.ElegivelEmprestimo = nullString(elegivel)
	lead.ValorLiberado = nullFloat(valor)
	lead.MargemDisponivel = nullFloat(margem)
	lead.NumeroParcelas = nullInt(parcelas)
	lead.DataHoraValidadeSolicitacao = nullString(validade)
	lead.DataNascimento = nullString(nascimento)
	lead.DataAdmissao = nullString(admissao)
	lead.IsMEI = nullString(isMEI)
	lead.IsJudicialRecovery = nullString(judicial)
	lead.PEPCodigo = nullString(pep)
	lead.ActiveFGTSDebts = nullString(fgts)
	lead.TypeWebhook = nullString(typeWebhook)
	lead.RawPayload = nullString(rawPayload)
	lead.Exportado = nullInt(exportado)
	lead.ReceivedAt = nullString(receivedAt)

	return lead, nil
}

func buildLeadWhere(filters models.LeadFilters) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if filters.From != "" || filters.To != "" {
		if filters.From != "" {
			clauses = append(clauses, "date(datetime(received_at, '-3 hours')) >= ?")
			args = append(args, filters.From)
		}
		if filters.To != "" {
			clauses = append(clauses, "date(datetime(received_at, '-3 hours')) <= ?")
			args = append(args, filters.To)
		}
	} else {
		switch filters.Period {
		case "24h":
			clauses = append(clauses, "datetime(received_at, '-3 hours') >= datetime('now', '-3 hours', '-1 day')")
		case "7d":
			clauses = append(clauses, "datetime(received_at, '-3 hours') >= datetime('now', '-3 hours', '-7 days')")
		case "30d":
			clauses = append(clauses, "datetime(received_at, '-3 hours') >= datetime('now', '-3 hours', '-30 days')")
		case "90d":
			clauses = append(clauses, "datetime(received_at, '-3 hours') >= datetime('now', '-3 hours', '-90 days')")
		}
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func nullFloat(value sql.NullFloat64) float64 {
	if value.Valid {
		return value.Float64
	}
	return 0
}

func nullInt(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

func booleanLabel(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "true", "1", "yes", "sim", "s":
		return "Sim"
	case "false", "0", "no", "nao", "não", "n":
		return "Não"
	default:
		return value
	}
}

func formatDecimalBR(value float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(value, 'f', 2, 64), ".", ",")
}

func formatDateBR(raw string, includeTime bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}
		if includeTime && hasTime(layout, raw) {
			return parsed.Format("02/01/2006 15:04:05")
		}
		return parsed.Format("02/01/2006")
	}

	return raw
}

func hasTime(layout, raw string) bool {
	return strings.Contains(layout, "15:04") || strings.Contains(raw, "T") || strings.Contains(raw, ":")
}
