package handlers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/luanlucolli/uy3-leads-api/internal/models"
)

const (
	defaultExportBatchSize = 500
	maxExportWindowDays    = 180
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
	if err := validateSummaryFilters(filters); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().In(brtLocation)
	where, args := buildSummaryWhereAt(filters, now)
	response := models.SummaryResponse{}
	summaryCtx, cancelSummary := context.WithTimeout(r.Context(), leadsSummaryTimeout)
	err = h.db.QueryRowContext(
		summaryCtx,
		"SELECT COALESCE(SUM(quantidade), 0) FROM leads_summary_daily"+where,
		args...,
	).Scan(&response.Total)
	cancelSummary()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar resumo de leads")
		return
	}
	if response.Total == 0 {
		writeJSON(w, http.StatusOK, response)
		return
	}

	var lastLeadAt sql.NullString
	lastLeadQuery, lastLeadArgs := buildLastLeadQueryAt(filters, now)
	lastLeadCtx, cancelLastLead := context.WithTimeout(r.Context(), leadsSummaryTimeout)
	err = h.db.QueryRowContext(lastLeadCtx, lastLeadQuery, lastLeadArgs...).Scan(&lastLeadAt)
	cancelLastLead()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "erro ao carregar ultimo lead")
		return
	}
	if lastLeadAt.Valid {
		response.LastLeadAt = formatDateForAPI(lastLeadAt.String)
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *LeadsHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	filters, err := models.ParseLeadFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateExportFilters(filters); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now().In(brtLocation)
	where, args := buildLeadWhereAt(filters, now)
	if err := h.ensureExportRowsExist(r.Context(), where, args); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "nenhum lead encontrado para exportar")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao preparar exportacao CSV")
		return
	}

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
	batchSize := defaultExportBatchSize
	var lastDate string
	var lastID int64
	hasCursor := false

	for {
		if err := r.Context().Err(); err != nil {
			return
		}

		batchWhere := where
		batchArgs := append([]any{}, args...)
		if hasCursor {
			cursorClause := "(received_at, id) < (?, ?)"
			if batchWhere == "" {
				batchWhere = " WHERE " + cursorClause
			} else {
				batchWhere += " AND " + cursorClause
			}
			batchArgs = append(batchArgs, lastDate, lastID)
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
			ORDER BY received_at DESC, id DESC
			LIMIT %d
		`, batchWhere, batchSize)

		batchCtx, cancelBatch := context.WithTimeout(r.Context(), exportBatchDBTimeout)
		rows, err := h.db.QueryContext(batchCtx, query, batchArgs...)
		if err != nil {
			cancelBatch()
			log.Printf("Erro na exportação CSV ao buscar leads: %v", err)
			return
		}

		batchCount := 0
		for rows.Next() {
			lead, err := scanLeadForCSV(rows)
			if err != nil {
				log.Printf("Erro na exportação CSV (Lead ID %d): %v", lead.ID, err)
				_ = rows.Close()
				cancelBatch()
				return
			}
			fillCSVRecord(record, lead)
			if err := csvWriter.Write(record); err != nil {
				log.Printf("Erro na exportação CSV (Lead ID %d): %v", lead.ID, err)
				_ = rows.Close()
				cancelBatch()
				return
			}
			lastDate = lead.ReceivedAt
			lastID = lead.ID
			hasCursor = true
			batchCount++
		}

		if err := rows.Err(); err != nil {
			log.Printf("Erro na exportação CSV após streaming: %v", err)
			_ = rows.Close()
			cancelBatch()
			return
		}
		if err := rows.Close(); err != nil {
			log.Printf("Erro ao encerrar exportação CSV: %v", err)
			cancelBatch()
			return
		}
		cancelBatch()

		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			log.Printf("Erro na exportação CSV: %v", err)
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

func validateExportFilters(filters models.LeadFilters) error {
	return validateDateFilters(filters, dateFilterValidationOptions{
		action:             "exportar CSV",
		allowAll:           false,
		maxCustomRangeDays: maxExportWindowDays,
	})
}

func validateSummaryFilters(filters models.LeadFilters) error {
	return validateDateFilters(filters, dateFilterValidationOptions{
		action:   "consultar o resumo",
		allowAll: true,
	})
}

type dateFilterValidationOptions struct {
	action             string
	allowAll           bool
	maxCustomRangeDays int
}

func validateDateFilters(filters models.LeadFilters, opts dateFilterValidationOptions) error {
	if filters.From == "" && filters.To == "" {
		switch filters.Period {
		case "24h", "7d", "30d", "90d":
			return nil
		case "all":
			if opts.allowAll {
				return nil
			}
			return fmt.Errorf("informe um periodo ou intervalo de datas para %s", opts.action)
		case "custom":
			return fmt.Errorf("informe data inicial e final para %s", opts.action)
		default:
			return fmt.Errorf("periodo invalido")
		}
	}
	if filters.From == "" || filters.To == "" {
		return fmt.Errorf("informe data inicial e final para %s", opts.action)
	}

	from, err := time.Parse("2006-01-02", filters.From)
	if err != nil {
		return fmt.Errorf("from deve estar no formato YYYY-MM-DD")
	}
	to, err := time.Parse("2006-01-02", filters.To)
	if err != nil {
		return fmt.Errorf("to deve estar no formato YYYY-MM-DD")
	}
	if to.Before(from) {
		return fmt.Errorf("to deve ser maior ou igual a from")
	}
	if opts.maxCustomRangeDays > 0 && to.Sub(from) > time.Duration(opts.maxCustomRangeDays)*24*time.Hour {
		return fmt.Errorf("intervalo maximo para exportacao CSV e de %d dias", opts.maxCustomRangeDays)
	}

	return nil
}

func (h *LeadsHandler) ensureExportRowsExist(parent context.Context, where string, args []any) error {
	checkCtx, cancel := context.WithTimeout(parent, leadsSummaryTimeout)
	defer cancel()

	var found int
	err := h.db.QueryRowContext(checkCtx, "SELECT 1 FROM leads"+where+" ORDER BY received_at DESC, id DESC LIMIT 1", args...).Scan(&found)
	if err != nil {
		return err
	}

	return nil
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

func fillCSVRecord(record []string, lead models.Lead) {
	record[0] = lead.CPF
	record[1] = lead.NomeTrabalhador
	record[2] = lead.Status
	record[3] = booleanLabel(lead.ElegivelEmprestimo)
	record[4] = formatDecimalBR(lead.ValorLiberado)
	record[5] = formatDecimalBR(lead.MargemDisponivel)
	record[6] = strconv.FormatInt(lead.NumeroParcelas, 10)
	record[7] = formatUTCDateBR(lead.ReceivedAt, true)
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
	return buildLeadWhereAt(filters, time.Now().In(brtLocation))
}

func buildLeadWhereAt(filters models.LeadFilters, now time.Time) (string, []any) {
	fromUTC, endExclusiveUTC, hasRange := leadDateTimeRange(filters, now)
	if !hasRange {
		return "", nil
	}

	return " WHERE received_at >= ? AND received_at < ?", []any{fromUTC, endExclusiveUTC}
}

func buildSummaryWhere(filters models.LeadFilters) (string, []any) {
	return buildSummaryWhereAt(filters, time.Now().In(brtLocation))
}

func buildSummaryWhereAt(filters models.LeadFilters, now time.Time) (string, []any) {
	fromDate, toDate := summaryDateRange(filters, now)
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

func buildLastLeadQueryAt(filters models.LeadFilters, now time.Time) (string, []any) {
	where, args := buildLeadWhereAt(filters, now)
	return "SELECT received_at FROM leads" + where + " ORDER BY received_at DESC, id DESC LIMIT 1", args
}

func summaryDateRange(filters models.LeadFilters, now time.Time) (string, string) {
	startDay, endDay, hasRange := resolveBRTDayRange(filters, now)
	if !hasRange {
		return "", ""
	}

	return startDay.Format("2006-01-02"), endDay.Format("2006-01-02")
}

func leadDateTimeRange(filters models.LeadFilters, now time.Time) (string, string, bool) {
	startDay, endDay, hasRange := resolveBRTDayRange(filters, now)
	if !hasRange {
		return "", "", false
	}

	start := startOfDayInLocation(startDay, brtLocation)
	endExclusive := startOfDayInLocation(endDay, brtLocation).AddDate(0, 0, 1)
	return start.UTC().Format("2006-01-02 15:04:05"), endExclusive.UTC().Format("2006-01-02 15:04:05"), true
}

func resolveBRTDayRange(filters models.LeadFilters, now time.Time) (time.Time, time.Time, bool) {
	loc := brtLocation
	now = now.In(loc)

	if filters.From != "" && filters.To != "" {
		fromDate, err := time.ParseInLocation("2006-01-02", filters.From, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		toDate, err := time.ParseInLocation("2006-01-02", filters.To, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		return startOfDayInLocation(fromDate, loc), startOfDayInLocation(toDate, loc), true
	}
	if filters.From != "" || filters.To != "" {
		return time.Time{}, time.Time{}, false
	}

	today := startOfDayInLocation(now, loc)
	switch filters.Period {
	case "24h":
		return today.AddDate(0, 0, -1), today, true
	case "7d":
		return today.AddDate(0, 0, -6), today, true
	case "30d":
		return today.AddDate(0, 0, -29), today, true
	case "90d":
		return today.AddDate(0, 0, -89), today, true
	default:
		return time.Time{}, time.Time{}, false
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

	zonedLayouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
	}
	localLayouts := []string{
		"02012006150405",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"02/01/2006 15:04:05",
	}
	dateOnlyLayouts := []string{
		"02012006",
		"2006-01-02",
		"02/01/2006",
	}

	for _, layout := range zonedLayouts {
		parsed, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}
		if includeTime && hasTime(layout, raw) {
			return parsed.In(brtLocation).Format("02/01/2006 15:04:05")
		}
		return parsed.Format("02/01/2006")
	}

	for _, layout := range localLayouts {
		parsed, err := time.ParseInLocation(layout, raw, brtLocation)
		if err != nil {
			continue
		}
		if includeTime && hasTime(layout, raw) {
			return parsed.Format("02/01/2006 15:04:05")
		}
		return parsed.Format("02/01/2006")
	}

	for _, layout := range dateOnlyLayouts {
		parsed, err := time.ParseInLocation(layout, raw, brtLocation)
		if err != nil {
			continue
		}
		return parsed.Format("02/01/2006")
	}

	return raw
}

func formatUTCDateBR(raw string, includeTime bool) string {
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
		if includeTime && hasTime(layout, raw) {
			return parsed.In(brtLocation).Format("02/01/2006 15:04:05")
		}
		return parsed.Format("02/01/2006")
	}

	return raw
}

func hasTime(layout, raw string) bool {
	return strings.Contains(layout, "15:04") || strings.Contains(layout, "150405") || strings.Contains(raw, "T") || strings.Contains(raw, ":")
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
		if hasTime(layout, raw) {
			return parsed.In(brtLocation).Format("2006-01-02 15:04:05")
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
