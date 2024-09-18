package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/cmd/internal"
)

var configRemoveCmd = &cobra.Command{
	Use:               "remove",
	Short:             "Removes awsx's Configuration",
	Long:              `Removes awsx's Configuration`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configNames := []string{"default"}
		if len(args) > 0 {
			configNames = args
		}
		return internal.RemoveInternalConfig(configNames)
	},
}

func init() {
	configCmd.AddCommand(configRemoveCmd)
}
