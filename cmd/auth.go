package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mvanduijker/knap-cli/internal/api"
	"github.com/mvanduijker/knap-cli/internal/config"
	"github.com/mvanduijker/knap-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Log in, check, or clear your API token",
	}

	auth.AddCommand(newAuthLoginCommand(), newAuthStatusCommand(), newAuthLogoutCommand(), newAuthTokenCommand())

	return auth
}

func newAuthLoginCommand() *cobra.Command {
	var token string
	var noBrowser bool

	login := &cobra.Command{
		Use:   "login",
		Short: "Store an API token created on the settings page",
		RunE: func(cmd *cobra.Command, args []string) error {
			settingsURL := baseURL() + "/settings"

			if token == "" {
				fmt.Fprintf(os.Stderr, "Create a token at %s\n", settingsURL)
				if !noBrowser {
					openBrowser(settingsURL)
				}
				fmt.Fprint(os.Stderr, "Paste your token: ")

				line, err := bufio.NewReader(os.Stdin).ReadString('\n')
				if err != nil && strings.TrimSpace(line) == "" {
					return fmt.Errorf("no token given")
				}
				token = strings.TrimSpace(line)
			}

			if token == "" {
				return fmt.Errorf("no token given")
			}

			if _, err := api.New(baseURL(), token).Accounts(); err != nil {
				return fmt.Errorf("that token does not work: %w", err)
			}

			if err := config.SaveToken(token); err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, "Logged in.")

			return nil
		},
	}

	login.Flags().StringVar(&token, "token", "", "the token to store, instead of reading it from stdin")
	login.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open the settings page")

	return login
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check whether the stored token still works",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}

			accounts, err := c.Accounts()
			if err != nil {
				return err
			}

			status := map[string]any{
				"api_url":  baseURL(),
				"accounts": len(accounts),
			}

			return render(status, output.Table{
				Headers: []string{"API URL", "ACCOUNTS"},
				Rows:    [][]string{{baseURL(), fmt.Sprint(len(accounts))}},
				IDs:     []string{baseURL()},
			})
		},
	}
}

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Forget the stored token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.DeleteToken(); err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, "Logged out. Revoke the token at "+baseURL()+"/settings")

			return nil
		},
	}
}

func newAuthTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the stored token",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.Token()
			if err != nil {
				return err
			}
			if token == "" {
				return fmt.Errorf("not logged in — run `knap auth login`")
			}

			fmt.Println(token)

			return nil
		},
	}
}

// openBrowser is best effort: login still works by pasting the token.
func openBrowser(url string) {
	var command string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
	case "windows":
		command = "explorer"
	default:
		command = "xdg-open"
	}

	_ = exec.Command(command, url).Start()
}
