package cmd

import (
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Interact with registries",
	Long:  `The registry command groups subcommands for creating, listing, and managing registries.`,
}

func init() {
	rootCmd.AddCommand(registryCmd)
}
