package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/mvanduijker/knap-cli/internal/api"
	"github.com/mvanduijker/knap-cli/internal/config"
	"github.com/mvanduijker/knap-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAccountCommand() *cobra.Command {
	account := &cobra.Command{
		Use:     "account",
		Aliases: []string{"accounts"},
		Short:   "List and manage allowance accounts",
	}

	account.AddCommand(
		newAccountListCommand(),
		newAccountShowCommand(),
		newAccountCreateCommand(),
		newAccountEditCommand(),
		newAccountDeleteCommand(),
		newAccountDefaultCommand(),
	)

	return account
}

func newAccountListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your allowance accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			accounts, err := c.Accounts()
			if err != nil {
				return err
			}

			return render(accounts, accountTable(accounts))
		},
	}
}

func newAccountShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [account]",
		Short: "Show one allowance account",
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

			return render(account, accountTable([]api.Account{account}))
		},
	}
}

func newAccountCreateCommand() *cobra.Command {
	var (
		interestRate float64
		currency     string
		amount       float64
		day          string
	)

	create := &cobra.Command{
		Use:   "create <title>",
		Short: "Start a new allowance account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			if day == "" {
				day = time.Now().Format("2006-01-02")
			}

			account, err := c.CreateAccount(map[string]any{
				"title":         args[0],
				"interest_rate": interestRate,
				"currency":      currency,
				"amount":        amount,
				"day":           day,
			})
			if err != nil {
				return err
			}

			return render(account, accountTable([]api.Account{account}))
		},
	}

	create.Flags().Float64Var(&interestRate, "interest-rate", 12, "yearly interest rate as a percentage")
	create.Flags().StringVar(&currency, "currency", "EUR", "EUR, USD or GBP")
	create.Flags().Float64Var(&amount, "amount", 0, "the starting balance")
	create.Flags().StringVar(&day, "day", "", "the day the account starts (YYYY-MM-DD, defaults to today)")

	return create
}

func newAccountEditCommand() *cobra.Command {
	var (
		title        string
		interestRate float64
	)

	edit := &cobra.Command{
		Use:   "edit [account]",
		Short: "Change an allowance account's title or interest rate",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			current, err := resolveAccount(c, firstArg(args))
			if err != nil {
				return err
			}

			if title == "" {
				title = current.Title
			}
			if !cmd.Flags().Changed("interest-rate") {
				interestRate = current.InterestRate
			}

			account, err := c.UpdateAccount(current.ID, map[string]any{
				"title":         title,
				"interest_rate": interestRate,
			})
			if err != nil {
				return err
			}

			return render(account, accountTable([]api.Account{account}))
		},
	}

	edit.Flags().StringVar(&title, "title", "", "the new title")
	edit.Flags().Float64Var(&interestRate, "interest-rate", 0, "the new yearly interest rate as a percentage")

	return edit
}

func newAccountDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [account]",
		Short: "Delete an allowance account",
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

			if err := c.DeleteAccount(account.ID); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Deleted %s.\n", account.Title)

			return nil
		},
	}
}

// newAccountDefaultCommand persists the account the other commands assume, so
// day-to-day use is `knap tx add 5.00`.
func newAccountDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default [account]",
		Short: "Save the account the other commands default to",
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

			stored, err := config.Load()
			if err != nil {
				return err
			}
			stored.Account = account.ID

			if err := config.Save(stored); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Default account is now %s.\n", account.Title)

			return nil
		},
	}
}

func accountTable(accounts []api.Account) output.Table {
	table := output.Table{Headers: []string{"ID", "TITLE", "CURRENCY", "INTEREST"}}
	for _, account := range accounts {
		table.Rows = append(table.Rows, []string{
			account.ID,
			account.Title,
			account.Currency,
			fmt.Sprintf("%.2f%%", account.InterestRate),
		})
		table.IDs = append(table.IDs, account.ID)
	}

	return table
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	return ""
}
