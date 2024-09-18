package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/cmd/internal"
	"github.com/vahid-haghighat/awsx/utilities"
)

var configExportPath string

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:               "export",
	Short:             "Exports awsx configs",
	Long:              `Exports awsx configs`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		configExportPath, err = utilities.AbsolutePath(configExportPath)
		if err != nil {
			return err
		}
		return internal.ExportInternalConfig(configExportPath)
	},
}

func init() {
	exportCmd.Flags().StringVarP(&configExportPath, "file", "f", "", "Path to save the exported config file")
	configCmd.AddCommand(exportCmd)
}
