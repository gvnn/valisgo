package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"valisgo/cmd/cli/browser"
	"valisgo/internal/auth"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via OIDC to get a refresh token",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfg := auth.OIDCConfig{
			IssuerURL: oidcIssuer,
			ClientID:  oidcClientID,
			Scopes:    []string{"openid", "profile", "email", "offline_access"},
		}

		authenticator, err := auth.NewAuthenticator(ctx, cfg)
		if err != nil {
			return err
		}

		// Run the browser login flow
		refreshToken, err := authenticator.LoginBrowser(ctx, browser.Open)
		if err != nil {
			return err
		}

		err = keyring.Set("valisgo", "refresh_token", refreshToken)
		if err != nil {
			return fmt.Errorf("failed to save token to system keyring: %w", err)
		}

		slog.Info("Refresh token saved securely to system keyring.")
		slog.Info("You can now run 'valisgo-cli proxy' or standard CLI commands!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
