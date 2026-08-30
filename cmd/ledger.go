package cmd

import (
	"fmt"
	"strings"

	"github.com/mvanduijker/knap-cli/internal/api"
	"github.com/mvanduijker/knap-cli/internal/output"
	"github.com/spf13/cobra"
)

func newLedgerCommand() *cobra.Command {
	ledger := newLedgerSummaryCommand()
	ledger.AddCommand(newLedgerRowsCommand())

	return ledger
}

func newLedgerSummaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ledger [account]",
		Short: "Show the running totals of an allowance account",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			account, err := resolveAccount(c, firstArg(args))
			if err != nil {
				return err
			}

			ledger, err := c.Ledger(account.ID)
			if err != nil {
				return err
			}

			money := func(amount float64) string { return output.Money(amount, account.Currency) }

			return render(ledger, output.Table{
				Headers: []string{"METRIC", "VALUE"},
				Rows: [][]string{
					{"Current total", money(ledger.CurrentTotal)},
					{"Interest today", money(ledger.TodayInterest)},
					{"Interest total", money(ledger.TotalInterest)},
					{"Interest per week", money(ledger.WeeklyInterestProjection)},
					{"Average per day", money(ledger.AverageDailyInterest)},
					{"Average per week", money(ledger.AverageWeeklyInterest)},
					{"Deposits", money(ledger.TotalDeposits)},
					{"Withdrawals", money(ledger.TotalWithdrawals)},
					{"Days", fmt.Sprint(ledger.DaysCount)},
				},
				IDs: []string{account.ID},
			})
		},
	}
}

// newLedgerRowsCommand shows the day-by-day ledger, where the interest that
// accrues without any transaction behind it is visible.
func newLedgerRowsCommand() *cobra.Command {
	var (
		year  int
		limit int
		all   bool
	)

	rows := &cobra.Command{
		Use:   "rows [account]",
		Short: "Show the ledger day by day, interest included",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			account, err := resolveAccount(c, firstArg(args))
			if err != nil {
				return err
			}

			// A year the account does not have is a 404, so there is nothing to
			// reconcile here — the error propagates and exits non-zero.
			ledger, err := c.LedgerRows(account.ID, year)
			if err != nil {
				return err
			}

			// Rows arrive newest first, so trimming keeps the recent days.
			if !all && limit > 0 && len(ledger.Rows) > limit {
				ledger.Rows = ledger.Rows[:limit]
			}

			money := func(amount float64) string { return output.Money(amount, account.Currency) }

			table := output.Table{Headers: []string{"DAY", "TRANSACTIONS", "INTEREST", "INTEREST TOTAL", "TOTAL"}}
			for _, row := range ledger.Rows {
				table.Rows = append(table.Rows, []string{
					row.Day,
					describeTransactions(row.Transactions, account.Currency),
					money(row.Interest),
					money(row.InterestTotal),
					money(row.Total),
				})
				table.IDs = append(table.IDs, row.Day)
			}

			return render(ledger, table)
		},
	}

	rows.Flags().IntVar(&year, "year", 0, "the year to show (defaults to the current one)")
	rows.Flags().IntVar(&limit, "limit", 30, "how many days to show, newest first")
	rows.Flags().BoolVar(&all, "all", false, "show every day of the year")

	return rows
}

func describeTransactions(transactions []api.Transaction, currency string) string {
	parts := make([]string, 0, len(transactions))
	for _, transaction := range transactions {
		amount := transaction.Amount
		if transaction.Type == "withdraw" {
			amount = -amount
		}

		part := output.Money(amount, currency)
		if transaction.Description != "" {
			part += " (" + transaction.Description + ")"
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, ", ")
}
