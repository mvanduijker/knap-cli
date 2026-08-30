// Package output renders one command result three ways: a human table, raw
// JSON, or the bare identifier under --quiet.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Mode struct {
	JSON  bool
	Quiet bool
}

// Table is what a command hands the renderer: column headers, the rows, and
// the identifiers --quiet prints instead.
type Table struct {
	Headers []string
	Rows    [][]string
	IDs     []string
}

// Render is the single formatting path every command goes through.
func Render(w io.Writer, mode Mode, payload any, table Table) error {
	switch {
	case mode.JSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")

		return encoder.Encode(payload)
	case mode.Quiet:
		for _, id := range table.IDs {
			if _, err := fmt.Fprintln(w, id); err != nil {
				return err
			}
		}

		return nil
	default:
		return renderTable(w, table)
	}
}

func renderTable(w io.Writer, table Table) error {
	if len(table.Rows) == 0 {
		_, err := fmt.Fprintln(w, "No results.")

		return err
	}

	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if len(table.Headers) > 0 {
		fmt.Fprintln(writer, strings.Join(table.Headers, "\t"))
	}
	for _, row := range table.Rows {
		fmt.Fprintln(writer, strings.Join(row, "\t"))
	}

	return writer.Flush()
}

func Money(amount float64, currency string) string {
	return fmt.Sprintf("%s %.2f", symbol(currency), amount)
}

func symbol(currency string) string {
	switch currency {
	case "EUR":
		return "€"
	case "USD":
		return "$"
	case "GBP":
		return "£"
	default:
		return currency
	}
}
