package cmd

import (
	"fmt"
	"os"

	"valisgo/cmd/cli/client"
	"valisgo/cmd/cli/printer"

	"github.com/spf13/cobra"
)

var registryCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		registryType, err := cmd.Flags().GetString("type")
		if err != nil {
			return err
		}

		switch registryType {
		case "pypi", "go", "npm", "file":
			// valid
		default:
			return fmt.Errorf("invalid registry type '%s'. Must be one of: pypi, go, npm, file", registryType)
		}

		apiClient, err := newAPIClient()
		if err != nil {
			return fmt.Errorf("failed to initialize client: %w", err)
		}

		ctx := cmd.Context()
		formatEnum := client.RegistryInputFormat(registryType)

		input := client.CreateRegistryJSONRequestBody{
			Name:   name,
			Format: &formatEnum,
		}

		res, err := apiClient.CreateRegistryWithResponse(ctx, input)
		if err != nil {
			return fmt.Errorf("error calling API: %w", err)
		}

		if res.StatusCode() != 201 {
			return fmt.Errorf("failed to create registry (HTTP %d): %s", res.StatusCode(), string(res.Body))
		}

		if res.JSON201 == nil {
			return fmt.Errorf("invalid response from server: empty body")
		}

		// Return the error from the printer
		return printer.Print(os.Stdout, res.JSON201, format)
	},
}

func init() {
	registryCreateCmd.Flags().String("type", "go", "Format type of the registry (pypi, go, npm, file)")
	registryCmd.AddCommand(registryCreateCmd)
}
