package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luanlucolli/uy3-leads-api/internal/models"
)

func TestFormatDateForAPIConvertsUTCToBRT(t *testing.T) {
	raw := "2026-05-01 00:46:51"

	got := formatDateForAPI(raw)

	if got != "2026-04-30 21:46:51" {
		t.Fatalf("formatDateForAPI(%q) = %q", raw, got)
	}
}

func TestFormatUTCDateBRConvertsUTCToBRTOnce(t *testing.T) {
	raw := "2026-05-01 00:46:51"

	got := formatUTCDateBR(raw, true)

	if got != "30/04/2026 21:46:51" {
		t.Fatalf("formatUTCDateBR(%q, true) = %q", raw, got)
	}
}

func TestFormatDateBRKeepsDateOnlyWithoutTimezoneShift(t *testing.T) {
	raw := "1990-05-20"

	got := formatDateBR(raw, false)

	if got != "20/05/1990" {
		t.Fatalf("formatDateBR(%q, false) = %q", raw, got)
	}
}

func TestFormatDateBRSupportsCompactDate(t *testing.T) {
	raw := "09072006"

	got := formatDateBR(raw, false)

	if got != "09/07/2006" {
		t.Fatalf("formatDateBR(%q, false) = %q", raw, got)
	}
}

func TestFormatDateBRSupportsCompactDateTime(t *testing.T) {
	raw := "05052026111027"

	got := formatDateBR(raw, true)

	if got != "05/05/2026 11:10:27" {
		t.Fatalf("formatDateBR(%q, true) = %q", raw, got)
	}
}

func TestFormatDateBRKeepsLocalDateTimeWithoutTimezoneShift(t *testing.T) {
	raw := "2026-05-05 11:10:27"

	got := formatDateBR(raw, true)

	if got != "05/05/2026 11:10:27" {
		t.Fatalf("formatDateBR(%q, true) = %q", raw, got)
	}
}

func TestFormatDateBRKeepsRFC3339DateWithoutTimezoneShiftWhenTimeHidden(t *testing.T) {
	raw := "1990-05-20T00:00:00Z"

	got := formatDateBR(raw, false)

	if got != "20/05/1990" {
		t.Fatalf("formatDateBR(%q, false) = %q", raw, got)
	}
}

func TestFormatDateForAPIDateOnlyDoesNotShiftTimezone(t *testing.T) {
	raw := "2026-05-01"

	got := formatDateForAPI(raw)

	if got != "2026-05-01" {
		t.Fatalf("formatDateForAPI(%q) = %q", raw, got)
	}
}

func TestBooleanLabelSupportsZeroAndOne(t *testing.T) {
	if got := booleanLabel("0"); got != "Não" {
		t.Fatalf("booleanLabel(%q) = %q", "0", got)
	}
	if got := booleanLabel("1"); got != "Sim" {
		t.Fatalf("booleanLabel(%q) = %q", "1", got)
	}
}

func TestFillCSVRecordFormatsOfficialCompactDates(t *testing.T) {
	record := make([]string, len(csvHeaders()))
	lead := models.Lead{
		ID:                          42,
		CPF:                         "63430832357",
		NomeTrabalhador:             "KAUE DO NASCIMENTO MORAIS",
		Status:                      "ATIVA",
		ElegivelEmprestimo:          "true",
		ValorLiberado:               1500,
		MargemDisponivel:            320.76,
		NumeroParcelas:              7,
		ReceivedAt:                  "2026-05-04 18:00:00",
		DataHoraValidadeSolicitacao: "2026-05-05 11:10:27",
		DataNascimento:              "2006-07-09",
		DataAdmissao:                "2025-10-08",
		IsMEI:                       "false",
		IsJudicialRecovery:          "false",
		PEPCodigo:                   "0",
	}

	fillCSVRecord(record, lead)

	if record[8] != "05/05/2026 11:10:27" {
		t.Fatalf("fillCSVRecord validade = %q", record[8])
	}
	if record[9] != "09/07/2006" {
		t.Fatalf("fillCSVRecord nascimento = %q", record[9])
	}
	if record[10] != "08/10/2025" {
		t.Fatalf("fillCSVRecord admissao = %q", record[10])
	}
	if record[14] != "Não" {
		t.Fatalf("fillCSVRecord pep = %q", record[14])
	}
}

func TestSummaryDateRangeForCustomDates(t *testing.T) {
	filters := models.LeadFilters{
		From: "2026-04-01",
		To:   "2026-04-30",
	}

	from, to := summaryDateRange(filters, time.Date(2026, 5, 1, 12, 0, 0, 0, brtLocation))

	if from != "2026-04-01" || to != "2026-04-30" {
		t.Fatalf("summaryDateRange(custom) = %q, %q", from, to)
	}
}

func TestSummaryDateRangeForFixedPeriods(t *testing.T) {
	now := time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation)

	from24h, to24h := summaryDateRange(models.LeadFilters{Period: "24h"}, now)
	if from24h != "2026-05-09" || to24h != "2026-05-10" {
		t.Fatalf("summaryDateRange(24h) = %q, %q", from24h, to24h)
	}

	from7d, to7d := summaryDateRange(models.LeadFilters{Period: "7d"}, now)
	if from7d != "2026-05-04" || to7d != "2026-05-10" {
		t.Fatalf("summaryDateRange(7d) = %q, %q", from7d, to7d)
	}

	from30d, to30d := summaryDateRange(models.LeadFilters{Period: "30d"}, now)
	if from30d != "2026-04-11" || to30d != "2026-05-10" {
		t.Fatalf("summaryDateRange(30d) = %q, %q", from30d, to30d)
	}

	from90d, to90d := summaryDateRange(models.LeadFilters{Period: "90d"}, now)
	if from90d != "2026-02-10" || to90d != "2026-05-10" {
		t.Fatalf("summaryDateRange(90d) = %q, %q", from90d, to90d)
	}
}

func TestBuildLeadWhereWithFromTo(t *testing.T) {
	filters := models.LeadFilters{
		From: "2026-04-01",
		To:   "2026-04-30",
	}

	where, args := buildLeadWhere(filters)

	if where != " WHERE received_at >= ? AND received_at < ?" {
		t.Fatalf("buildLeadWhere where = %q", where)
	}
	if len(args) != 2 {
		t.Fatalf("buildLeadWhere args length = %d", len(args))
	}
	if args[0] != "2026-04-01 03:00:00" {
		t.Fatalf("buildLeadWhere from arg = %v", args[0])
	}
	if args[1] != "2026-05-01 03:00:00" {
		t.Fatalf("buildLeadWhere endExclusive arg = %v", args[1])
	}
}

func TestBuildLeadWhereAtUsesYesterdayAndTodayFor24h(t *testing.T) {
	now := time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation)

	where, args := buildLeadWhereAt(models.LeadFilters{Period: "24h"}, now)

	if where != " WHERE received_at >= ? AND received_at < ?" {
		t.Fatalf("buildLeadWhereAt(24h) where = %q", where)
	}
	if len(args) != 2 {
		t.Fatalf("buildLeadWhereAt(24h) args length = %d", len(args))
	}
	if args[0] != "2026-05-09 03:00:00" {
		t.Fatalf("buildLeadWhereAt(24h) from arg = %v", args[0])
	}
	if args[1] != "2026-05-11 03:00:00" {
		t.Fatalf("buildLeadWhereAt(24h) endExclusive arg = %v", args[1])
	}
}

func TestLeadDateTimeRangeUsesExclusiveEndForCustomRange(t *testing.T) {
	startUTC, endExclusiveUTC, ok := leadDateTimeRange(models.LeadFilters{
		From: "2026-04-01",
		To:   "2026-04-30",
	}, time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation))

	if !ok {
		t.Fatal("leadDateTimeRange(custom) expected range")
	}
	if startUTC != "2026-04-01 03:00:00" {
		t.Fatalf("leadDateTimeRange(custom) startUTC = %q", startUTC)
	}
	if endExclusiveUTC != "2026-05-01 03:00:00" {
		t.Fatalf("leadDateTimeRange(custom) endExclusiveUTC = %q", endExclusiveUTC)
	}
}

func TestValidateExportFiltersBlocksAllWithoutDate(t *testing.T) {
	err := validateExportFilters(models.LeadFilters{Period: "all"})

	if err == nil {
		t.Fatal("validateExportFilters(all) expected error")
	}
}

func TestExportCSVStreamsWithoutCursorOrBatchLimit(t *testing.T) {
	db, queries := openLeadExportTestDB(t, [][]driver.Value{
		{
			int64(2),
			"22222222222",
			"ANA SILVA",
			"ATIVA",
			"true",
			float64(1234.5),
			float64(30.1),
			int64(12),
			"2026-05-02 15:30:00",
			"2026-05-03 10:00:00",
			"1990-01-02",
			"2020-04-05",
			"false",
			"0",
			"1",
			"true",
		},
		{
			int64(1),
			"11111111111",
			"BRUNO COSTA",
			"INATIVA",
			"false",
			float64(0),
			float64(0),
			int64(0),
			"2026-05-01 12:00:00",
			"",
			"",
			"",
			"true",
			"false",
			"0",
			"false",
		},
	})
	handler := NewLeadsHandler(db)

	request := httptest.NewRequest(http.MethodGet, "/leads/export?period=custom&from=2026-05-01&to=2026-05-02", nil)
	response := httptest.NewRecorder()

	handler.ExportCSV(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if len(*queries) != 2 {
		t.Fatalf("query count = %d", len(*queries))
	}

	mainQuery := normalizeSQL((*queries)[1].query)
	if strings.Contains(mainQuery, "(received_at, id) < (?, ?)") {
		t.Fatalf("export query still uses cursor: %q", mainQuery)
	}
	if strings.Contains(mainQuery, "LIMIT 500") {
		t.Fatalf("export query still uses batch limit: %q", mainQuery)
	}
	if strings.Contains(mainQuery, " LIMIT ") {
		t.Fatalf("export query should not use SQL limit: %q", mainQuery)
	}
	if !strings.Contains(mainQuery, "ORDER BY received_at DESC, id DESC") {
		t.Fatalf("export query missing stable order: %q", mainQuery)
	}

	body := response.Body.String()
	if !strings.HasPrefix(body, "\ufeff") {
		t.Fatalf("CSV missing UTF-8 BOM")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\ufeff")))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("csv record count = %d, records = %#v", len(records), records)
	}
	if strings.Join(records[0], "\x00") != strings.Join(csvHeaders(), "\x00") {
		t.Fatalf("csv headers = %#v", records[0])
	}
	if records[1][0] != "22222222222" || records[1][1] != "ANA SILVA" || records[1][7] != "02/05/2026 12:30:00" || records[1][11] != "2" {
		t.Fatalf("first csv row = %#v", records[1])
	}
	if records[2][0] != "11111111111" || records[2][1] != "BRUNO COSTA" || records[2][7] != "01/05/2026 09:00:00" || records[2][11] != "1" {
		t.Fatalf("second csv row = %#v", records[2])
	}
}

func TestValidateExportFiltersAcceptsFixedPeriods(t *testing.T) {
	periods := []string{"24h", "7d", "30d", "90d"}

	for _, period := range periods {
		if err := validateExportFilters(models.LeadFilters{Period: period}); err != nil {
			t.Fatalf("validateExportFilters(%q) unexpected error = %v", period, err)
		}
	}
}

func TestValidateExportFiltersRequireCompleteCustomRange(t *testing.T) {
	tests := []models.LeadFilters{
		{Period: "custom", From: "2026-01-01"},
		{Period: "custom", To: "2026-01-31"},
	}

	for _, filters := range tests {
		if err := validateExportFilters(filters); err == nil {
			t.Fatalf("validateExportFilters(%+v) expected error", filters)
		}
	}
}

func TestValidateExportFiltersRejectsToBeforeFrom(t *testing.T) {
	err := validateExportFilters(models.LeadFilters{
		From: "2026-02-01",
		To:   "2026-01-31",
	})

	if err == nil {
		t.Fatal("validateExportFilters(to before from) expected error")
	}
}

func TestValidateExportFiltersLimitsCustomWindow(t *testing.T) {
	err := validateExportFilters(models.LeadFilters{
		From: "2026-01-01",
		To:   "2026-07-01",
	})

	if err == nil {
		t.Fatal("validateExportFilters(large custom window) expected error")
	}
}

func TestBuildLastLeadQueryAtUsesFilterRange(t *testing.T) {
	now := time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation)

	query, args := buildLastLeadQueryAt(models.LeadFilters{Period: "7d"}, now)

	if !strings.Contains(query, "WHERE received_at >= ? AND received_at < ?") {
		t.Fatalf("buildLastLeadQueryAt query = %q", query)
	}
	if !strings.Contains(query, "ORDER BY received_at DESC, id DESC LIMIT 1") {
		t.Fatalf("buildLastLeadQueryAt order = %q", query)
	}
	if len(args) != 2 {
		t.Fatalf("buildLastLeadQueryAt args length = %d", len(args))
	}
	if args[0] != "2026-05-04 03:00:00" || args[1] != "2026-05-11 03:00:00" {
		t.Fatalf("buildLastLeadQueryAt args = %#v", args)
	}
}

func TestBuildLastLeadQueryAtForAllUsesGlobalLatest(t *testing.T) {
	query, args := buildLastLeadQueryAt(models.LeadFilters{Period: "all"}, time.Date(2026, 5, 10, 15, 30, 0, 0, brtLocation))

	if query != "SELECT received_at FROM leads ORDER BY received_at DESC, id DESC LIMIT 1" {
		t.Fatalf("buildLastLeadQueryAt(all) query = %q", query)
	}
	if len(args) != 0 {
		t.Fatalf("buildLastLeadQueryAt(all) args = %#v", args)
	}
}

type leadExportQueryCall struct {
	query string
	args  []driver.NamedValue
}

type leadExportDriver struct{}

type leadExportConn struct {
	queries    *[]leadExportQueryCall
	exportRows [][]driver.Value
}

type leadExportRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

type leadExportTx struct{}

var (
	leadExportDriverOnce    sync.Once
	leadExportDriverName    = "lead_export_test_driver"
	leadExportDriverMu      sync.Mutex
	leadExportDriverCounter int
	leadExportDriverCases   = map[string]leadExportDriverCase{}
)

type leadExportDriverCase struct {
	queries    *[]leadExportQueryCall
	exportRows [][]driver.Value
}

func openLeadExportTestDB(t *testing.T, exportRows [][]driver.Value) (*sql.DB, *[]leadExportQueryCall) {
	t.Helper()

	leadExportDriverOnce.Do(func() {
		sql.Register(leadExportDriverName, leadExportDriver{})
	})

	leadExportDriverMu.Lock()
	leadExportDriverCounter++
	dsn := fmt.Sprintf("lead-export-test-%d", leadExportDriverCounter)
	queries := &[]leadExportQueryCall{}
	leadExportDriverCases[dsn] = leadExportDriverCase{
		queries:    queries,
		exportRows: exportRows,
	}
	leadExportDriverMu.Unlock()

	db, err := sql.Open(leadExportDriverName, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		leadExportDriverMu.Lock()
		delete(leadExportDriverCases, dsn)
		leadExportDriverMu.Unlock()
	})

	return db, queries
}

func (leadExportDriver) Open(name string) (driver.Conn, error) {
	leadExportDriverMu.Lock()
	defer leadExportDriverMu.Unlock()

	testCase, ok := leadExportDriverCases[name]
	if !ok {
		return nil, fmt.Errorf("unknown dsn %q", name)
	}

	return &leadExportConn{
		queries:    testCase.queries,
		exportRows: testCase.exportRows,
	}, nil
}

func (c *leadExportConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepare not supported")
}

func (c *leadExportConn) Close() error {
	return nil
}

func (c *leadExportConn) Begin() (driver.Tx, error) {
	return leadExportTx{}, nil
}

func (c *leadExportConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return leadExportTx{}, nil
}

func (c *leadExportConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	clonedArgs := append([]driver.NamedValue(nil), args...)
	*c.queries = append(*c.queries, leadExportQueryCall{
		query: query,
		args:  clonedArgs,
	})

	normalized := normalizeSQL(query)
	if strings.HasPrefix(normalized, "SELECT 1 FROM leads") {
		return &leadExportRows{
			columns: []string{"1"},
			rows:    [][]driver.Value{{int64(1)}},
		}, nil
	}

	return &leadExportRows{
		columns: []string{
			"id",
			"cpf",
			"nome_trabalhador",
			"status",
			"elegivel_emprestimo",
			"valor_liberado",
			"margem_disponivel",
			"numero_parcelas",
			"received_at",
			"data_hora_validade_solicitacao",
			"data_nascimento",
			"data_admissao",
			"is_mei",
			"is_judicial_recovery",
			"pep_codigo",
			"active_fgts_debts",
		},
		rows: c.exportRows,
	}, nil
}

func (leadExportTx) Commit() error {
	return nil
}

func (leadExportTx) Rollback() error {
	return nil
}

func (r *leadExportRows) Columns() []string {
	return r.columns
}

func (r *leadExportRows) Close() error {
	return nil
}

func (r *leadExportRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func normalizeSQL(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
