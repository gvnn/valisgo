package cmd

import (
	"fmt"
	"os"

	"valisgo/cmd/cli/printer"

	"github.com/spf13/cobra"
)

var registryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registries",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := newAPIClient()
		if err != nil {
			return fmt.Errorf("failed to initialize client: %w", err)
		}

		ctx := cmd.Context()
		res, err := apiClient.ListRegistriesWithResponse(ctx)
		if err != nil {
			return fmt.Errorf("error calling API: %w", err)
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("failed to list registries (HTTP %d): %s", res.StatusCode(), string(res.Body))
		}

		if res.JSON200 == nil {
			return fmt.Errorf("invalid response from server: empty body")
		}

		return printer.Print(os.Stdout, *res.JSON200, format)
	},
}

func init() {
	registryCmd.AddCommand(registryListCmd)
}
