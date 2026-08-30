package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/mvanduijker/knap-cli/internal/api"
	"github.com/mvanduijker/knap-cli/internal/config"
	"github.com/mvanduijker/knap-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	version = "dev"

	flagJSON    bool
	flagQuiet   bool
	flagAccount string
)

func Execute(buildVersion string) int {
	if buildVersion != "" {
		version = buildVersion
	}

	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "knap: "+err.Error())

		return 1
	}

	return 0
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "knap",
		Short:         "Manage your knap.app allowance accounts from the terminal",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "print the full payload as JSON")
	root.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "print only ids")
	root.PersistentFlags().StringVar(&flagAccount, "account", "", "allowance account id or title prefix")

	root.AddCommand(newAuthCommand(), newAccountCommand(), newTxCommand(), newLedgerCommand())

	return root
}

func mode() output.Mode {
	return output.Mode{JSON: flagJSON, Quiet: flagQuiet}
}

func render(payload any, table output.Table) error {
	return output.Render(os.Stdout, mode(), payload, table)
}

// client builds an authenticated client, or explains how to get one.
func client() (*api.Client, error) {
	token, err := config.Token()
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, fmt.Errorf("not logged in — run `knap auth login`")
	}

	return api.New(baseURL(), token), nil
}

func baseURL() string {
	if url := os.Getenv("KNAP_API_URL"); url != "" {
		return url
	}

	if stored, err := config.Load(); err == nil && stored.APIURL != "" {
		return stored.APIURL
	}

	return api.DefaultBaseURL
}

// resolveAccount accepts a sqid or a case-insensitive title prefix, falling
// back to the account saved in the config, then to the only account there is.
func resolveAccount(c *api.Client, given string) (api.Account, error) {
	if given == "" {
		given = flagAccount
	}
	if given == "" {
		if stored, err := config.Load(); err == nil {
			given = stored.Account
		}
	}

	accounts, err := c.Accounts()
	if err != nil {
		return api.Account{}, err
	}

	if given == "" {
		switch len(accounts) {
		case 1:
			return accounts[0], nil
		case 0:
			return api.Account{}, fmt.Errorf("no allowance accounts yet — run `knap account create <title>`")
		default:
			return api.Account{}, fmt.Errorf("more than one account — pass --account <id or title>")
		}
	}

	var matches []api.Account
	for _, account := range accounts {
		if account.ID == given {
			return account, nil
		}
		if strings.HasPrefix(strings.ToLower(account.Title), strings.ToLower(given)) {
			matches = append(matches, account)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.Account{}, fmt.Errorf("no allowance account matches %q", given)
	default:
		return api.Account{}, fmt.Errorf("%q matches more than one account — use the id", given)
	}
}
