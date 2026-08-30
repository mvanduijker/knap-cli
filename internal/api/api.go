// Package api is a hand-written client for the knap.app JSON API.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://knap.app"

// Account mirrors AllowanceAccountResource. InterestRate is a percentage.
type Account struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Currency     string  `json:"currency"`
	InterestRate float64 `json:"interest_rate"`
}

// Transaction mirrors TransactionResource. Amount is always positive; Type
// carries the direction.
type Transaction struct {
	ID          string  `json:"id"`
	Day         string  `json:"day"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// Ledger mirrors LedgerResource.
type Ledger struct {
	CurrentTotal             float64 `json:"current_total"`
	TodayInterest            float64 `json:"today_interest"`
	TotalInterest            float64 `json:"total_interest"`
	DaysCount                int     `json:"days_count"`
	WeeklyInterestProjection float64 `json:"weekly_interest_projection"`
	AverageDailyInterest     float64 `json:"average_daily_interest"`
	AverageWeeklyInterest    float64 `json:"average_weekly_interest"`
	TotalDeposits            float64 `json:"total_deposits"`
	TotalWithdrawals         float64 `json:"total_withdrawals"`
}

// LedgerRow is one day of the ledger, including the interest that accrued
// without any transaction behind it.
type LedgerRow struct {
	Day           string        `json:"day"`
	Interest      float64       `json:"interest"`
	InterestTotal float64       `json:"interest_total"`
	Total         float64       `json:"total"`
	Transactions  []Transaction `json:"transactions"`
}

// LedgerRows is a year of rows plus which years the account has at all.
type LedgerRows struct {
	Rows           []LedgerRow `json:"rows"`
	Year           int         `json:"year"`
	AvailableYears []int       `json:"available_years"`
}

// Error is a non-2xx response. The server speaks Laravel's {message, errors}
// shape, so validation failures carry a field-keyed detail map.
type Error struct {
	Status  int
	Message string
	Errors  map[string][]string
}

func (e *Error) Error() string {
	if len(e.Errors) == 0 {
		return e.Message
	}

	fields := make([]string, 0, len(e.Errors))
	for field := range e.Errors {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	var lines []string
	for _, field := range fields {
		for _, message := range e.Errors[field] {
			lines = append(lines, fmt.Sprintf("  %s: %s", field, message))
		}
	}

	// Laravel repeats the first field error as the top-level message, so a
	// lone field would otherwise be printed twice.
	if len(lines) == 1 && strings.HasSuffix(lines[0], ": "+e.Message) {
		return e.Message
	}

	return e.Message + "\n" + strings.Join(lines, "\n")
}

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Accounts() ([]Account, error) {
	var accounts []Account
	err := c.do(http.MethodGet, "/api/v1/allowance-accounts", nil, &accounts)

	return accounts, err
}

func (c *Client) Account(id string) (Account, error) {
	var account Account
	err := c.do(http.MethodGet, "/api/v1/allowance-accounts/"+url.PathEscape(id), nil, &account)

	return account, err
}

func (c *Client) CreateAccount(body any) (Account, error) {
	var account Account
	err := c.do(http.MethodPost, "/api/v1/allowance-accounts", body, &account)

	return account, err
}

func (c *Client) UpdateAccount(id string, body any) (Account, error) {
	var account Account
	err := c.do(http.MethodPatch, "/api/v1/allowance-accounts/"+url.PathEscape(id), body, &account)

	return account, err
}

func (c *Client) DeleteAccount(id string) error {
	return c.do(http.MethodDelete, "/api/v1/allowance-accounts/"+url.PathEscape(id), nil, nil)
}

func (c *Client) Ledger(accountID string) (Ledger, error) {
	var ledger Ledger
	err := c.do(http.MethodGet, "/api/v1/allowance-accounts/"+url.PathEscape(accountID)+"/ledger", nil, &ledger)

	return ledger, err
}

// LedgerRows fetches one year of ledger rows. Year 0 asks for the current one.
func (c *Client) LedgerRows(accountID string, year int) (LedgerRows, error) {
	path := "/api/v1/allowance-accounts/" + url.PathEscape(accountID) + "/ledger-rows"
	if year > 0 {
		path += "?year=" + strconv.Itoa(year)
	}

	raw, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return LedgerRows{}, err
	}

	var envelope struct {
		Data []LedgerRow `json:"data"`
		Meta struct {
			Year           int   `json:"year"`
			AvailableYears []int `json:"available_years"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return LedgerRows{}, fmt.Errorf("unexpected response: %s", strings.TrimSpace(string(raw)))
	}

	return LedgerRows{
		Rows:           envelope.Data,
		Year:           envelope.Meta.Year,
		AvailableYears: envelope.Meta.AvailableYears,
	}, nil
}

func (c *Client) Transactions(accountID string) ([]Transaction, error) {
	var transactions []Transaction
	err := c.do(http.MethodGet, "/api/v1/allowance-accounts/"+url.PathEscape(accountID)+"/transactions", nil, &transactions)

	return transactions, err
}

func (c *Client) CreateTransaction(accountID string, body any) (Transaction, error) {
	var transaction Transaction
	err := c.do(http.MethodPost, "/api/v1/allowance-accounts/"+url.PathEscape(accountID)+"/transactions", body, &transaction)

	return transaction, err
}

func (c *Client) UpdateTransaction(id string, body any) (Transaction, error) {
	var transaction Transaction
	err := c.do(http.MethodPatch, "/api/v1/transactions/"+url.PathEscape(id), body, &transaction)

	return transaction, err
}

func (c *Client) DeleteTransaction(id string) error {
	return c.do(http.MethodDelete, "/api/v1/transactions/"+url.PathEscape(id), nil, nil)
}

// do sends the request and unwraps the API's {"data": …} envelope into out.
func (c *Client) do(method, path string, body, out any) error {
	raw, err := c.request(method, path, body)
	if err != nil {
		return err
	}

	if out == nil || len(raw) == 0 {
		return nil
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("unexpected response: %s", strings.TrimSpace(string(raw)))
	}

	return json.Unmarshal(envelope.Data, out)
}

// request sends the request and returns the raw body, or a typed *Error.
func (c *Client) request(method, path string, body any) ([]byte, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, c.BaseURL+path, payload)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.HTTP.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, decodeError(response.StatusCode, raw)
	}

	return raw, nil
}

func decodeError(status int, raw []byte) error {
	apiError := &Error{Status: status}

	var payload struct {
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil && payload.Message != "" {
		apiError.Message = payload.Message
		apiError.Errors = payload.Errors

		return apiError
	}

	apiError.Message = http.StatusText(status)
	if apiError.Message == "" {
		apiError.Message = fmt.Sprintf("request failed with status %d", status)
	}

	return apiError
}
