package cmd

import (
	"fmt"
	"os"

	"valisgo/cmd/cli/client"

	"github.com/spf13/cobra"
)

var (
	address string
	format  string
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
		defaultAddr = "http://127.0.0.1:8080/manage"
	}

	rootCmd.PersistentFlags().StringVar(&address, "address", defaultAddr, "Address of the Valisgo server (env: VALISGO_ADDR)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "json", "Output format (json, csv)")
}

func newAPIClient() (*client.ClientWithResponses, error) {
	return client.NewClientWithResponses(address)
}
