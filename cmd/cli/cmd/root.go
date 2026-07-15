package cmd

import (
	"fmt"
	"os"
	"strings"

	"valisgo/cmd/cli/client"

	"github.com/spf13/cobra"
)

var (
	// Global API config
	address string
	format  string

	// Global Authentication config (Used by login, proxy, and the API client)
	oidcIssuer   string
	oidcClientID string
	wifFile      string
	wifEnv       string
)

var rootCmd = &cobra.Command{
	Use:   "valisgo-cli",
	Short: "Valisgo CLI is a management tool for Valisgo registries",
	Long:  `A fast and flexible CLI to manage your Valisgo registries and repositories.`,
	// Do not print usage automatically when an error (like an API failure) occurs.
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Fail fast if the output format isn't supported globally
		if format != "json" && format != "csv" {
			return fmt.Errorf("unsupported format '%s': must be 'json' or 'csv'", format)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra already prints errors by default unless SilenceErrors=true.
		// Simply exiting with 1 is sufficient.
		os.Exit(1)
	}
}

func init() {
	defaultAddr := os.Getenv("VALISGO_ADDR")
	if defaultAddr == "" {
		defaultAddr = "http://127.0.0.1:8080"
	}

	defaultIssuer := os.Getenv("OIDC_ISSUER")
	if defaultIssuer == "" {
		defaultIssuer = "http://dex:5556/dex"
	}

	defaultClientID := os.Getenv("OIDC_CLIENT_ID")
	if defaultClientID == "" {
		defaultClientID = "valisgo-cli"
	}

	rootCmd.PersistentFlags().StringVar(&address, "address", defaultAddr, "Address of the Valisgo server (env: VALISGO_ADDR)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "json", "Output format (json, csv)")

	rootCmd.PersistentFlags().StringVar(&oidcIssuer, "issuer", defaultIssuer, "OIDC Issuer URL (env: OIDC_ISSUER)")
	rootCmd.PersistentFlags().StringVar(&oidcClientID, "client-id", defaultClientID, "OIDC Client ID (env: OIDC_CLIENT_ID)")
	rootCmd.PersistentFlags().StringVar(&wifFile, "wif-file", os.Getenv("OIDC_WIF_FILE"), "Path to Workload Identity Token file")
	rootCmd.PersistentFlags().StringVar(&wifEnv, "wif-env", os.Getenv("OIDC_WIF_ENV"), "Env Var name containing Workload Identity Token")
}

func newAPIClient() (*client.ClientWithResponses, error) {
	manageAddr := strings.TrimRight(address, "/") + "/manage"
	return client.NewClientWithResponses(manageAddr)
}
