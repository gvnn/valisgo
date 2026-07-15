package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove authentication tokens from your system keyring",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := keyring.Delete("valisgo", "refresh_token")
		if err != nil {
			if err == keyring.ErrNotFound {
				slog.Info("You are already logged out.")
				return nil
			}
			return fmt.Errorf("failed to remove token from keyring: %w", err)
		}

		slog.Info("Successfully logged out. Token removed from system keyring.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
