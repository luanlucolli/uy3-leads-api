package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/luanlucolli/uy3-leads-api/internal/models"
)

var brtLocation = time.FixedZone("BRT", -3*3600)

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

	where, args := buildSummaryWhere(filters)
	var total int64
	if err := h.db.QueryRowContext(
		r.Context(),
		"SELECT COALESCE(SUM(quantidade), 0) FROM leads_summary_daily"+where,
		args...,
	).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar resumo de leads")
		return
	}

	writeJSON(w, http.StatusOK, models.SummaryResponse{Total: total})
}

func (h *LeadsHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	filters, err := models.ParseLeadFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	where, args := buildLeadWhere(filters)

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

	headers := csvHeaders()
	if err := csvWriter.Write(headers); err != nil {
		return
	}
	record := make([]string, len(headers))

	flusher, canFlush := w.(http.Flusher)
	const batchSize = 1000
	lastSeenID := int64(0)
	hasCursor := false

	for {
		batchWhere := where
		batchArgs := append([]any{}, args...)

		if hasCursor {
			keysetClause := "id > ?"
			if filters.Direction == "desc" {
				keysetClause = "id < ?"
			}
			if batchWhere == "" {
				batchWhere = " WHERE " + keysetClause
			} else {
				batchWhere += " AND " + keysetClause
			}
			batchArgs = append(batchArgs, lastSeenID)
		}

		query := fmt.Sprintf(`
			SELECT
				id, cpf, nome_trabalhador, status, elegivel_emprestimo,
				valor_liberado, margem_disponivel, numero_parcelas,
				received_at, data_hora_validade_solicitacao, data_nascimento,
				data_admissao, is_mei, is_judicial_recovery, pep_codigo,
				active_fgts_debts
			FROM leads
			%s
			ORDER BY id %s
			LIMIT %d
		`, batchWhere, filters.Direction, batchSize)

		rows, err := h.db.QueryContext(r.Context(), query, batchArgs...)
		if err != nil {
			log.Printf("Erro na exportação CSV ao buscar lote: %v", err)
			return
		}

		batchCount := 0
		for rows.Next() {
			lead, err := scanLeadForCSV(rows)
			if err != nil {
				log.Printf("Erro na exportação CSV (Lead ID %d): %v", lead.ID, err)
				_ = rows.Close()
				return
			}
			fillCSVRecord(record, lead)
			if err := csvWriter.Write(record); err != nil {
				log.Printf("Erro na exportação CSV (Lead ID %d): %v", lead.ID, err)
				_ = rows.Close()
				return
			}
			lastSeenID = lead.ID
			hasCursor = true
			batchCount++
		}

		if err := rows.Err(); err != nil {
			log.Printf("Erro na exportação CSV após streaming: %v", err)
			_ = rows.Close()
			return
		}
		if err := rows.Close(); err != nil {
			log.Printf("Erro ao encerrar lote da exportação CSV: %v", err)
			return
		}
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			log.Printf("Erro na exportação CSV (Lead ID %d): %v", lastSeenID, err)
			return
		}
		if canFlush {
			flusher.Flush()
		}

		if batchCount < batchSize {
			break
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
	return scanLeadWithOptions(scanner, true)
}

func scanLeadRaw(scanner leadScanner) (models.Lead, error) {
	return scanLeadWithOptions(scanner, false)
}

func scanLeadForCSV(scanner leadScanner) (models.Lead, error) {
	var lead models.Lead
	var cpf, nome, status, elegivel, receivedAt sql.NullString
	var validade, nascimento, admissao sql.NullString
	var isMEI, judicial, pep, fgts sql.NullString
	var valor, margem sql.NullFloat64
	var parcelas sql.NullInt64

	err := scanner.Scan(
		&lead.ID,
		&cpf,
		&nome,
		&status,
		&elegivel,
		&valor,
		&margem,
		&parcelas,
		&receivedAt,
		&validade,
		&nascimento,
		&admissao,
		&isMEI,
		&judicial,
		&pep,
		&fgts,
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
	lead.ReceivedAt = nullString(receivedAt)
	lead.DataHoraValidadeSolicitacao = nullString(validade)
	lead.DataNascimento = nullString(nascimento)
	lead.DataAdmissao = nullString(admissao)
	lead.IsMEI = nullString(isMEI)
	lead.IsJudicialRecovery = nullString(judicial)
	lead.PEPCodigo = nullString(pep)
	lead.ActiveFGTSDebts = nullString(fgts)

	return lead, nil
}

func scanLeadWithOptions(scanner leadScanner, normalizeReceivedAt bool) (models.Lead, error) {
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
	if normalizeReceivedAt {
		lead.ReceivedAt = formatDateForAPI(lead.ReceivedAt)
	}

	return lead, nil
}

func fillCSVRecord(record []string, lead models.Lead) {
	record[0] = lead.CPF
	record[1] = lead.NomeTrabalhador
	record[2] = lead.Status
	record[3] = booleanLabel(lead.ElegivelEmprestimo)
	record[4] = formatDecimalBR(lead.ValorLiberado)
	record[5] = formatDecimalBR(lead.MargemDisponivel)
	record[6] = strconv.FormatInt(lead.NumeroParcelas, 10)
	record[7] = formatDateBR(lead.ReceivedAt, true)
	record[8] = formatDateBR(lead.DataHoraValidadeSolicitacao, true)
	record[9] = formatDateBR(lead.DataNascimento, false)
	record[10] = formatDateBR(lead.DataAdmissao, false)
	record[11] = strconv.FormatInt(lead.ID, 10)
	record[12] = booleanLabel(lead.IsMEI)
	record[13] = booleanLabel(lead.IsJudicialRecovery)
	record[14] = booleanLabel(lead.PEPCodigo)
	record[15] = booleanLabel(lead.ActiveFGTSDebts)
}

func buildLeadWhere(filters models.LeadFilters) (string, []any) {
	loc := brtLocation
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if filters.From != "" || filters.To != "" {
		if filters.From != "" {
			fromTime, err := time.ParseInLocation("2006-01-02 15:04:05", filters.From+" 00:00:00", loc)
			if err == nil {
				clauses = append(clauses, "received_at >= ?")
				args = append(args, fromTime.UTC().Format("2006-01-02 15:04:05"))
			}
		}
		if filters.To != "" {
			toTime, err := time.ParseInLocation("2006-01-02 15:04:05", filters.To+" 23:59:59", loc)
			if err == nil {
				clauses = append(clauses, "received_at <= ?")
				args = append(args, toTime.UTC().Format("2006-01-02 15:04:05"))
			}
		}
	} else {
		now := time.Now().In(loc)
		switch filters.Period {
		case "24h":
			clauses = append(clauses, "received_at >= ?")
			args = append(args, now.Add(-24*time.Hour).UTC().Format("2006-01-02 15:04:05"))
		case "7d":
			cutoff := startOfDayInLocation(now.AddDate(0, 0, -7), loc)
			clauses = append(clauses, "received_at >= ?")
			args = append(args, cutoff.UTC().Format("2006-01-02 15:04:05"))
		case "30d":
			cutoff := startOfDayInLocation(now.AddDate(0, 0, -30), loc)
			clauses = append(clauses, "received_at >= ?")
			args = append(args, cutoff.UTC().Format("2006-01-02 15:04:05"))
		case "90d":
			cutoff := startOfDayInLocation(now.AddDate(0, 0, -90), loc)
			clauses = append(clauses, "received_at >= ?")
			args = append(args, cutoff.UTC().Format("2006-01-02 15:04:05"))
		}
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildSummaryWhere(filters models.LeadFilters) (string, []any) {
	fromDate, toDate := summaryDateRange(filters, time.Now().In(brtLocation))
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if fromDate != "" {
		clauses = append(clauses, "data >= ?")
		args = append(args, fromDate)
	}
	if toDate != "" {
		clauses = append(clauses, "data <= ?")
		args = append(args, toDate)
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func summaryDateRange(filters models.LeadFilters, now time.Time) (string, string) {
	if filters.From != "" || filters.To != "" {
		return filters.From, filters.To
	}

	today := now.In(brtLocation).Format("2006-01-02")
	switch filters.Period {
	case "24h":
		// leads_summary_daily stores BRT day buckets, so a 24h filter must include
		// the daily buckets overlapped by the last 24 hours window.
		return now.Add(-24 * time.Hour).In(brtLocation).Format("2006-01-02"), today
	case "7d":
		return now.AddDate(0, 0, -7).In(brtLocation).Format("2006-01-02"), today
	case "30d":
		return now.AddDate(0, 0, -30).In(brtLocation).Format("2006-01-02"), today
	case "90d":
		return now.AddDate(0, 0, -90).In(brtLocation).Format("2006-01-02"), today
	default:
		return "", ""
	}
}

func startOfDayInLocation(value time.Time, loc *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, loc)
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
		parsed, err := parseUTCDate(raw, layout)
		if err != nil {
			continue
		}
		parsed = parsed.In(brtLocation)
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

func formatDateForAPI(raw string) string {
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
	}

	for _, layout := range layouts {
		parsed, err := parseUTCDate(raw, layout)
		if err != nil {
			continue
		}
		parsed = parsed.In(brtLocation)
		if hasTime(layout, raw) {
			return parsed.Format("2006-01-02 15:04:05")
		}
		return parsed.Format("2006-01-02")
	}

	return raw
}

func parseUTCDate(raw, layout string) (time.Time, error) {
	switch layout {
	case time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999-07:00":
		return time.Parse(layout, raw)
	default:
		return time.ParseInLocation(layout, raw, time.UTC)
	}
}
