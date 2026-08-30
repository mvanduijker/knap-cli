package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountsUnwrapsTheDataEnvelope(t *testing.T) {
	var gotAuth, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		io.WriteString(w, `{"data":[{"id":"acc_1","title":"Emma","currency":"EUR","interest_rate":12}]}`)
	}))
	defer server.Close()

	accounts, err := New(server.URL, "knap_secret").Accounts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "Bearer knap_secret" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotPath != "/api/v1/allowance-accounts" {
		t.Errorf("path = %q", gotPath)
	}
	if len(accounts) != 1 || accounts[0].Title != "Emma" || accounts[0].InterestRate != 12 {
		t.Errorf("accounts = %+v", accounts)
	}
}

func TestCreateTransactionSendsTypeAndPositiveAmount(t *testing.T) {
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"data":{"id":"trn_1","day":"2026-01-02","type":"withdraw","amount":2.5,"description":"sweets"}}`)
	}))
	defer server.Close()

	transaction, err := New(server.URL, "t").CreateTransaction("acc_1", map[string]any{
		"type":   "withdraw",
		"day":    "2026-01-02",
		"amount": 2.5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["type"] != "withdraw" || body["amount"] != 2.5 {
		t.Errorf("body = %+v", body)
	}
	if transaction.ID != "trn_1" || transaction.Amount != 2.5 {
		t.Errorf("transaction = %+v", transaction)
	}
}

func TestValidationErrorsCarryFieldMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		io.WriteString(w, `{"message":"The given data was invalid.","errors":{"amount":["The amount field is required."],"day":["The day field is required."]}}`)
	}))
	defer server.Close()

	_, err := New(server.URL, "t").CreateAccount(map[string]any{})

	apiError, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiError.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d", apiError.Status)
	}

	want := "The given data was invalid.\n  amount: The amount field is required.\n  day: The day field is required."
	if apiError.Error() != want {
		t.Errorf("Error() = %q, want %q", apiError.Error(), want)
	}
}

// Laravel sets the top-level message to the first field error, so a lone
// failure must not be printed twice.
func TestASingleValidationErrorIsNotRepeated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		io.WriteString(w, `{"message":"The amount field is required.","errors":{"amount":["The amount field is required."]}}`)
	}))
	defer server.Close()

	_, err := New(server.URL, "t").CreateAccount(map[string]any{})

	if err.Error() != "The amount field is required." {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestUnauthorizedFallsBackToTheStatusText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := New(server.URL, "t").Accounts()

	apiError, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiError.Error() != "Unauthorized" {
		t.Errorf("Error() = %q", apiError.Error())
	}
}

func TestLedgerRowsReadsBothDataAndMeta(t *testing.T) {
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"data":[
			{"day":"2025-01-02","interest":0.1,"interest_total":0.1,"total":100.1,"transactions":[]},
			{"day":"2025-01-01","interest":0,"interest_total":0,"total":100,
			 "transactions":[{"id":"trn_1","day":"2025-01-01","type":"deposit","amount":100,"description":"start"}]}
		],"meta":{"year":2025,"available_years":[2024,2025]}}`)
	}))
	defer server.Close()

	ledger, err := New(server.URL, "t").LedgerRows("acc_1", 2025)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotQuery != "year=2025" {
		t.Errorf("query = %q", gotQuery)
	}
	if ledger.Year != 2025 || len(ledger.AvailableYears) != 2 {
		t.Errorf("meta = %+v", ledger)
	}
	if len(ledger.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(ledger.Rows))
	}
	// A day carrying interest but no transaction is the whole point.
	if ledger.Rows[0].Interest != 0.1 || len(ledger.Rows[0].Transactions) != 0 {
		t.Errorf("row 0 = %+v", ledger.Rows[0])
	}
	if len(ledger.Rows[1].Transactions) != 1 || ledger.Rows[1].Transactions[0].ID != "trn_1" {
		t.Errorf("row 1 = %+v", ledger.Rows[1])
	}
}

func TestLedgerRowsOmitsTheYearWhenUnset(t *testing.T) {
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"data":[],"meta":{"year":2026,"available_years":[2026]}}`)
	}))
	defer server.Close()

	if _, err := New(server.URL, "t").LedgerRows("acc_1", 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty", gotQuery)
	}
}

func TestDeleteAcceptsAnEmptyBody(t *testing.T) {
	var gotMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := New(server.URL, "t").DeleteTransaction("trn_1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q", gotMethod)
	}
}
