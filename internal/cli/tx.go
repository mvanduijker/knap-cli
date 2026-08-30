package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mvanduijker/knap-cli/internal/api"
	"github.com/mvanduijker/knap-cli/internal/output"
	"github.com/spf13/cobra"
)

func newTxCommand() *cobra.Command {
	tx := &cobra.Command{
		Use:     "tx",
		Aliases: []string{"transaction", "transactions"},
		Short:   "List and manage transactions",
	}

	tx.AddCommand(newTxListCommand(), newTxAddCommand(), newTxEditCommand(), newTxDeleteCommand())

	return tx
}

func newTxListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [account]",
		Short: "List the transactions of an allowance account",
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

			transactions, err := c.Transactions(account.ID)
			if err != nil {
				return err
			}

			return render(transactions, transactionTable(transactions, account.Currency))
		},
	}
}

func newTxAddCommand() *cobra.Command {
	var (
		description string
		day         string
		withdraw    bool
	)

	add := &cobra.Command{
		Use:   "add <amount>",
		Short: "Add a transaction (a deposit unless --withdraw)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return fmt.Errorf("%q is not an amount", args[0])
			}

			c, err := client()
			if err != nil {
				return err
			}

			account, err := resolveAccount(c, "")
			if err != nil {
				return err
			}

			if day == "" {
				day = time.Now().Format("2006-01-02")
			}

			transaction, err := c.CreateTransaction(account.ID, map[string]any{
				"type":        transactionType(withdraw),
				"day":         day,
				"amount":      amount,
				"description": description,
			})
			if err != nil {
				return err
			}

			return render(transaction, transactionTable([]api.Transaction{transaction}, account.Currency))
		},
	}

	add.Flags().StringVar(&description, "description", "", "what the transaction is for")
	add.Flags().StringVar(&day, "day", "", "the day of the transaction (YYYY-MM-DD, defaults to today)")
	add.Flags().BoolVar(&withdraw, "withdraw", false, "record money going out instead of in")

	return add
}

func newTxEditCommand() *cobra.Command {
	var (
		description string
		day         string
		amount      float64
		withdraw    bool
		deposit     bool
	)

	edit := &cobra.Command{
		Use:   "edit <transaction>",
		Short: "Change a transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			account, err := resolveAccount(c, "")
			if err != nil {
				return err
			}

			transactions, err := c.Transactions(account.ID)
			if err != nil {
				return err
			}

			current, err := findTransaction(transactions, args[0])
			if err != nil {
				return err
			}

			if day == "" {
				day = current.Day
			}
			if !cmd.Flags().Changed("amount") {
				amount = current.Amount
			}
			if !cmd.Flags().Changed("description") {
				description = current.Description
			}

			transactionType := current.Type
			if withdraw {
				transactionType = "withdraw"
			}
			if deposit {
				transactionType = "deposit"
			}

			transaction, err := c.UpdateTransaction(current.ID, map[string]any{
				"type":        transactionType,
				"day":         day,
				"amount":      amount,
				"description": description,
			})
			if err != nil {
				return err
			}

			return render(transaction, transactionTable([]api.Transaction{transaction}, account.Currency))
		},
	}

	edit.Flags().StringVar(&description, "description", "", "what the transaction is for")
	edit.Flags().StringVar(&day, "day", "", "the day of the transaction (YYYY-MM-DD)")
	edit.Flags().Float64Var(&amount, "amount", 0, "the amount, always positive")
	edit.Flags().BoolVar(&withdraw, "withdraw", false, "turn it into money going out")
	edit.Flags().BoolVar(&deposit, "deposit", false, "turn it into money coming in")

	return edit
}

func newTxDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <transaction>",
		Short: "Delete a transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			if err := c.DeleteTransaction(args[0]); err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, "Deleted "+args[0]+".")

			return nil
		},
	}
}

func transactionType(withdraw bool) string {
	if withdraw {
		return "withdraw"
	}

	return "deposit"
}

func findTransaction(transactions []api.Transaction, id string) (api.Transaction, error) {
	for _, transaction := range transactions {
		if transaction.ID == id {
			return transaction, nil
		}
	}

	return api.Transaction{}, fmt.Errorf("no transaction %q on this account", id)
}

func transactionTable(transactions []api.Transaction, currency string) output.Table {
	table := output.Table{Headers: []string{"ID", "DAY", "TYPE", "AMOUNT", "DESCRIPTION"}}
	for _, transaction := range transactions {
		table.Rows = append(table.Rows, []string{
			transaction.ID,
			transaction.Day,
			transaction.Type,
			output.Money(transaction.Amount, currency),
			transaction.Description,
		})
		table.IDs = append(table.IDs, transaction.ID)
	}

	return table
}
