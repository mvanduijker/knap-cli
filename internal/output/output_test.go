package output

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

var table = Table{
	Headers: []string{"ID", "TITLE", "CURRENCY", "INTEREST"},
	Rows: [][]string{
		{"acc_1", "Emma", "EUR", "12.00%"},
		{"acc_22", "Lucas van Dam", "USD", "8.50%"},
	},
	IDs: []string{"acc_1", "acc_22"},
}

var payload = []map[string]any{
	{"id": "acc_1", "title": "Emma"},
	{"id": "acc_22", "title": "Lucas van Dam"},
}

func TestRender(t *testing.T) {
	for name, mode := range map[string]Mode{
		"table": {},
		"json":  {JSON: true},
		"quiet": {Quiet: true},
	} {
		t.Run(name, func(t *testing.T) {
			var buffer bytes.Buffer
			if err := Render(&buffer, mode, payload, table); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertGolden(t, name+".golden", buffer.Bytes())
		})
	}
}

func TestRenderEmptyTable(t *testing.T) {
	var buffer bytes.Buffer
	if err := Render(&buffer, Mode{}, []any{}, Table{Headers: []string{"ID"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buffer.String() != "No results.\n" {
		t.Errorf("got %q", buffer.String())
	}
}

func TestMoney(t *testing.T) {
	for _, testCase := range []struct {
		amount   float64
		currency string
		want     string
	}{
		{12.5, "EUR", "€ 12.50"},
		{0, "USD", "$ 0.00"},
		{3.456, "GBP", "£ 3.46"},
		{1, "CHF", "CHF 1.00"},
	} {
		if got := Money(testCase.amount, testCase.currency); got != testCase.want {
			t.Errorf("Money(%v, %q) = %q, want %q", testCase.amount, testCase.currency, got, testCase.want)
		}
	}
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run `go test ./... -update` to create it)", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
