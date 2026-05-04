package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const officialWebhookPayload = `{
  "cpf": "63430832357",
  "is_mei": false,
  "status": "ATIVA",
  "typeWebook": "LEADS_CLT",
  "dataAdmissao": "08102025",
  "valorLiberado": 1500,
  "dataNascimento": "09072006",
  "numeroParcelas": 7,
  "nomeTrabalhador": "KAUE DO NASCIMENTO MORAIS",
  "codigoRequisicao": "23fafd80-07e1-492d-b671-a202e4b51ee8",
  "margemDisponivel": 320.76,
  "active_fgts_debts": null,
  "elegivelEmprestimo": true,
  "all_branch_employees": null,
  "is_judicial_recovery": false,
  "numeroInscricaoEmpregador": "30045533",
  "pessoaExpostaPoliticamente": {
    "Codigo": 0,
    "Descricao": "Pessoa Não Exposta Politicamente"
  },
  "dataHoraValidadeSolicitacao": "05052026111027"
}`

func TestParseWebhookLeadOfficialPayloadNormalizesCompactFields(t *testing.T) {
	payload := decodeWebhookPayload(t, officialWebhookPayload)

	lead := parseWebhookLead(payload)

	if lead.CPF != "63430832357" {
		t.Fatalf("CPF = %q", lead.CPF)
	}
	if lead.TypeWebhook != "LEADS_CLT" {
		t.Fatalf("TypeWebhook = %q", lead.TypeWebhook)
	}
	if lead.DataNascimento != "2006-07-09" {
		t.Fatalf("DataNascimento = %q", lead.DataNascimento)
	}
	if lead.DataAdmissao != "2025-10-08" {
		t.Fatalf("DataAdmissao = %q", lead.DataAdmissao)
	}
	if lead.DataHoraValidadeSolicitacao != "2026-05-05 11:10:27" {
		t.Fatalf("DataHoraValidadeSolicitacao = %q", lead.DataHoraValidadeSolicitacao)
	}
	if lead.PEPCodigo != "0" {
		t.Fatalf("PEPCodigo = %q", lead.PEPCodigo)
	}
	if lead.ActiveFGTSDebts != "" {
		t.Fatalf("ActiveFGTSDebts = %q", lead.ActiveFGTSDebts)
	}
}

func TestWebhookReceiveAcceptsOfficialPayload(t *testing.T) {
	db, execs := openWebhookTestDB(t)
	handler := NewWebhookHandler(db)

	request := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(officialWebhookPayload))
	response := httptest.NewRecorder()

	handler.Receive(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if len(*execs) != 2 {
		t.Fatalf("exec count = %d", len(*execs))
	}

	insertLead := (*execs)[0]
	if !strings.Contains(insertLead.query, "INSERT INTO leads") {
		t.Fatalf("first query = %q", insertLead.query)
	}
	assertNamedValue(t, insertLead.args, 0, "63430832357")
	assertNamedValue(t, insertLead.args, 7, "2026-05-05 11:10:27")
	assertNamedValue(t, insertLead.args, 8, "2006-07-09")
	assertNamedValue(t, insertLead.args, 9, "2025-10-08")
	assertNamedValue(t, insertLead.args, 12, "0")
	assertNamedValue(t, insertLead.args, 14, "LEADS_CLT")

	insertSummary := (*execs)[1]
	if !strings.Contains(insertSummary.query, "INSERT INTO leads_summary_daily") {
		t.Fatalf("second query = %q", insertSummary.query)
	}
	if len(insertSummary.args) != 1 {
		t.Fatalf("summary args length = %d", len(insertSummary.args))
	}
}

func decodeWebhookPayload(t *testing.T, raw string) map[string]any {
	t.Helper()

	var payload map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	return payload
}

func assertNamedValue(t *testing.T, args []driver.NamedValue, index int, want any) {
	t.Helper()

	if len(args) <= index {
		t.Fatalf("args length = %d, want index %d", len(args), index)
	}
	if got := args[index].Value; got != want {
		t.Fatalf("args[%d] = %#v, want %#v", index, got, want)
	}
}

type webhookExecCall struct {
	query string
	args  []driver.NamedValue
}

type webhookDriver struct{}

type webhookConn struct {
	execs *[]webhookExecCall
}

type webhookTx struct{}

type webhookResult int64

var (
	webhookDriverOnce sync.Once
	webhookDriverName = "webhook_test_driver"

	webhookDriverMu      sync.Mutex
	webhookDriverCounter int
	webhookDriverExecs   = map[string]*[]webhookExecCall{}
)

func openWebhookTestDB(t *testing.T) (*sql.DB, *[]webhookExecCall) {
	t.Helper()

	webhookDriverOnce.Do(func() {
		sql.Register(webhookDriverName, webhookDriver{})
	})

	webhookDriverMu.Lock()
	webhookDriverCounter++
	dsn := fmt.Sprintf("webhook-test-%d", webhookDriverCounter)
	execs := &[]webhookExecCall{}
	webhookDriverExecs[dsn] = execs
	webhookDriverMu.Unlock()

	db, err := sql.Open(webhookDriverName, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		webhookDriverMu.Lock()
		delete(webhookDriverExecs, dsn)
		webhookDriverMu.Unlock()
	})

	return db, execs
}

func (webhookDriver) Open(name string) (driver.Conn, error) {
	webhookDriverMu.Lock()
	defer webhookDriverMu.Unlock()

	execs, ok := webhookDriverExecs[name]
	if !ok {
		return nil, fmt.Errorf("unknown dsn %q", name)
	}

	return &webhookConn{execs: execs}, nil
}

func (c *webhookConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepare not supported")
}

func (c *webhookConn) Close() error {
	return nil
}

func (c *webhookConn) Begin() (driver.Tx, error) {
	return webhookTx{}, nil
}

func (c *webhookConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return webhookTx{}, nil
}

func (c *webhookConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	clonedArgs := append([]driver.NamedValue(nil), args...)
	*c.execs = append(*c.execs, webhookExecCall{
		query: query,
		args:  clonedArgs,
	})

	return webhookResult(len(*c.execs)), nil
}

func (webhookTx) Commit() error {
	return nil
}

func (webhookTx) Rollback() error {
	return nil
}

func (r webhookResult) LastInsertId() (int64, error) {
	return int64(r), nil
}

func (r webhookResult) RowsAffected() (int64, error) {
	return 1, nil
}
